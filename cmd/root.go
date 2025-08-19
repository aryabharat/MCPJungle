/*
Package cmd implements the CLI command structure for mcpjungle.

Command Organization:
- Commands are grouped into "basic" and "advanced" categories using annotations
- Within each group, commands are ordered using a numeric "order" annotation
- To add a new command:
 1. Create the command file in the cmd package
 2. Add annotations to specify group and order:
    cmd.Annotations = map[string]string{
    "group": string(subCommandGroupBasic), // or subCommandGroupAdvanced
    "order": "5", // numeric order within the group
    }
 3. Register the command with rootCmd.AddCommand()

Missing annotations will cause groupCommands() to return an error.
*/
package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/mcpjungle/mcpjungle/client"
	"github.com/mcpjungle/mcpjungle/cmd/config"
	"github.com/spf13/cobra"
)

// subCommandGroup defines a type for categorizing subcommands into groups
type subCommandGroup string

const (
	// subCommandGroupBasic represents basic commands that are commonly used and essential for beginners
	subCommandGroupBasic subCommandGroup = "basic"
	// subCommandGroupAdvanced represents advanced commands that are for advanced or enterprise use cases
	subCommandGroupAdvanced subCommandGroup = "advanced"
)

// unorderedCommand is a special value used to indicate that a command does not have any order specified.
const unorderedCommand = -1

// TODO: refactor: all commands should use cmd.Print..() instead of fmt.Print..() statements to produce outputs.

// displayASCIIArt prints the MCPJungle ASCII art banner
func displayASCIIArt(cmd *cobra.Command) {
	cmd.Println(`
  ███╗   ███╗ ██████╗██████╗     ██╗██╗   ██╗███╗   ██╗ ██████╗ ██╗     ███████╗
  ████╗ ████║██╔════╝██╔══██╗    ██║██║   ██║████╗  ██║██╔════╝ ██║     ██╔════╝
  ██╔████╔██║██║     ██████╔╝    ██║██║   ██║██╔██╗ ██║██║  ███╗██║     █████╗  
  ██║╚██╔╝██║██║     ██╔═══╝  ██╗██║██║   ██║██║╚██╗██║██║   ██║██║     ██╔══╝  
  ██║ ╚═╝ ██║╚██████╗██║      ╚█║██║╚██████╔╝██║ ╚████║╚██████╔╝███████╗███████╗
  ╚═╝     ╚═╝ ╚═════╝╚═╝       ╚╝╚═╝ ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝ ╚══════╝╚══════╝`)
}

// SilentErr is a sentinel error used to indicate that the command should not print an error message
// This is useful when we handle error printing internally but want main to exit with a non-zero status.
// See https://github.com/spf13/cobra/issues/914#issuecomment-548411337
var SilentErr = errors.New("SilentErr")

var registryServerURL string

// apiClient is the global API client used by command handlers to interact with the MCPJungle registry server.
// It is not the best choice to rely on a global variable, but cobra doesn't seem to provide any neat way to
// pass an object down the command tree.
var apiClient *client.Client

var rootCmd = &cobra.Command{
	Use:   "mcpjungle",
	Short: "MCP Gateway for AI Agents",

	SilenceErrors: true,
	SilenceUsage:  true,

	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},

	Run: func(cmd *cobra.Command, args []string) {
		// check if version flag was provided
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			// Execute the version command directly
			versionCmd.Run(cmd, args)
			return
		}
		// show custom help message when no subcommand is provided
		displayRootCmdHelpMsg(cmd)
	},
}

func Execute() error {
	// Store the default help function before setting our custom one
	defaultHelpFunc := rootCmd.HelpFunc()

	// Set custom help function that handles both root and subcommands
	rootCmd.SetHelpFunc(customHelpFunc(defaultHelpFunc))

	// only print usage and error messages if the command usage is incorrect
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		// Display ASCII art for unknown flag errors
		displayASCIIArt(cmd)
		cmd.Println()
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return SilentErr
	})

	rootCmd.PersistentFlags().StringVar(
		&registryServerURL,
		"registry",
		"http://127.0.0.1:"+BindPortDefault,
		"Base URL of the MCPJungle registry server",
	)

	// Add version flags (-v and --version)
	rootCmd.Flags().BoolP("version", "v", false, "Display version information")

	// Initialize the API client with the registry server URL & client configuration (if any)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		apiClient = client.NewClient(registryServerURL, cfg.AccessToken, http.DefaultClient)
	}

	return rootCmd.Execute()
}

// displayRootCmdHelpMsg displays custom help message for the root command, ie,
// when the mcpjungle CLI is run without any subcommands.
func displayRootCmdHelpMsg(cmd *cobra.Command) {
	// Display ASCII art
	displayASCIIArt(cmd)
	cmd.Println()
	cmd.Println(cmd.Short)
	cmd.Println()
	cmd.Printf("Usage:\n  %s\n\n", cmd.UseLine())

	// group commands by category
	commandGroups, err := groupCommands(cmd.Commands())
	if err != nil {
		cmd.Println("Error grouping commands:", err)
		return
	}

	// Display each group
	displayCommandGroup(cmd, "Basic Commands:", commandGroups[string(subCommandGroupBasic)])
	displayCommandGroup(cmd, "Advanced Commands:", commandGroups[string(subCommandGroupAdvanced)])

	cmd.Println("Flags:")
	cmd.Print(cmd.LocalFlags().FlagUsages())
	cmd.Printf("Use \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
}

// customHelpFunc returns a help function that displays a custom help message for the root command
// and falls back to the default help for subcommands.
func customHelpFunc(defaultHelpFunc func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.Parent() == nil {
			// this is the root command, display custom help message
			displayRootCmdHelpMsg(cmd)
			return
		}
		// this is a subcommand, use the original default help
		defaultHelpFunc(cmd, args)
	}
}

// groupCommands organizes sub-commands by their group annotation
func groupCommands(commands []*cobra.Command) (map[string][]*cobra.Command, error) {
	groups := map[string][]*cobra.Command{
		string(subCommandGroupBasic):    {},
		string(subCommandGroupAdvanced): {},
	}

	for _, subCmd := range commands {
		// skip non-functional commands
		if !subCmd.IsAvailableCommand() || subCmd.IsAdditionalHelpTopicCommand() {
			continue
		}

		if subCmd.Annotations == nil {
			return nil, fmt.Errorf("subcommand '%s' has no annotations, cannot determine group", subCmd.Name())
		}

		group := subCmd.Annotations["group"]
		if group != string(subCommandGroupBasic) && group != string(subCommandGroupAdvanced) {
			return nil, fmt.Errorf("unknown group '%s' for subcommand '%s'", subCmd.Annotations["group"], subCmd.Name())
		}

		groups[group] = append(groups[group], subCmd)
	}

	// sort each group by order annotation
	for groupName := range groups {
		sortCommandsByOrder(groups[groupName])
	}

	return groups, nil
}

// displayCommandGroup shows a group of commands with an optional header
func displayCommandGroup(cmd *cobra.Command, header string, commands []*cobra.Command) {
	if len(commands) == 0 {
		return
	}
	if header != "" {
		cmd.Println(header)
	}
	for _, subCmd := range commands {
		cmd.Printf("  %-11s %s\n", subCmd.Name(), subCmd.Short)
	}
	cmd.Println()
}

// sortCommandsByOrder sorts sub-commands by their order.
// Two subcommands CAN have the same order.
// If they belong to the same group, they will be displayed one after the other.
// If they belong to different groups, their order only applies within their own group.
func sortCommandsByOrder(commands []*cobra.Command) {
	sort.Slice(commands, func(i, j int) bool {
		orderI := getOrderValue(commands[i])
		orderJ := getOrderValue(commands[j])

		// Handle unordered commands (-1) - they go to the end
		if orderI == unorderedCommand && orderJ == unorderedCommand {
			// if both commands are unordered, sort by name
			return commands[i].Name() < commands[j].Name()
		}
		if orderI == unorderedCommand {
			return false // i goes after j
		}
		if orderJ == unorderedCommand {
			return true // i goes before j
		}

		return orderI < orderJ
	})
}

// getOrderValue returns the order specified for the given command within its group.
// If the command has no specific order, it returns -1 (unordered).
func getOrderValue(cmd *cobra.Command) int {
	if cmd.Annotations == nil {
		return unorderedCommand
	}

	orderStr, exists := cmd.Annotations["order"]
	if !exists {
		return unorderedCommand
	}

	order, err := strconv.Atoi(orderStr)
	if err != nil {
		return unorderedCommand
	}
	return order
}

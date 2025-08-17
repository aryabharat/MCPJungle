package cmd

import (
	"errors"
	"sort"
	"github.com/mcpjungle/mcpjungle/client"
	"github.com/mcpjungle/mcpjungle/cmd/config"
	"github.com/spf13/cobra"
	"net/http"
)

// TODO: refactor: all commands should use cmd.Print..() instead of fmt.Print..() statements to produce outputs.

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
	Short: "MCP tool catalog",

	SilenceErrors: true,
	SilenceUsage:  true,

	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	
	Run: func(cmd *cobra.Command, args []string) {
		// Show custom help when no subcommand is provided
		showRootHelp(cmd)
	},
}

func Execute() error {
	// Store the default help function before setting our custom one
	defaultHelpFunc := rootCmd.HelpFunc()
	
	// Set custom help function that handles both root and subcommands
	rootCmd.SetHelpFunc(CustomHelpFunc(defaultHelpFunc))
	
	// only print usage and error messages if the command usage is incorrect
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
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

	// Initialize the API client with the registry server URL & client configuration (if any)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		apiClient = client.NewClient(registryServerURL, cfg.AccessToken, http.DefaultClient)
	}

	return rootCmd.Execute()
}

// CustomHelpFunc returns a help function that can handle both root and subcommands
func CustomHelpFunc(defaultHelpFunc func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.Parent() == nil {
			// This is the root command
			showRootHelp(cmd)
		} else {
			// This is a subcommand, use the original default help
			defaultHelpFunc(cmd, args)
		}
	}
}

// showRootHelp displays custom help for the root command with grouped commands
func showRootHelp(cmd *cobra.Command) {
	cmd.Println(cmd.Short)
	cmd.Println()
	cmd.Printf("Usage:\n  %s\n\n", cmd.UseLine())

	// Group commands by category
	commandGroups := groupCommands(cmd.Commands())
	
	// Display each group
	displayCommandGroup(cmd, "Basic Commands:", commandGroups["basic"])
	displayCommandGroup(cmd, "Advanced Commands:", commandGroups["advanced"])
	displayCommandGroup(cmd, "", commandGroups["other"]) // Other commands without header

	cmd.Println("Flags:")
	cmd.Print(cmd.LocalFlags().FlagUsages())
	cmd.Printf("Use \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
}

// groupCommands organizes commands by their group annotation
func groupCommands(commands []*cobra.Command) map[string][]*cobra.Command {
	groups := map[string][]*cobra.Command{
		"basic":    {},
		"advanced": {},
		"other":    {},
	}

	for _, subCmd := range commands {
		if !subCmd.IsAvailableCommand() || subCmd.IsAdditionalHelpTopicCommand() {
			continue
		}
		
		group := subCmd.Annotations["group"]
		if group == "" {
			group = "other"
		}
		
		if _, exists := groups[group]; !exists {
			groups[group] = []*cobra.Command{}
		}
		groups[group] = append(groups[group], subCmd)
	}



	// Sort each group by order annotation
	for groupName := range groups {
		sortCommandsByOrder(groups[groupName])
	}

	return groups
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

// sortCommandsByOrder sorts commands by their order
func sortCommandsByOrder(commands []*cobra.Command) {
	sort.Slice(commands, func(i, j int) bool {
		orderI := getOrderValue(commands[i])
		orderJ := getOrderValue(commands[j])
		
		// Handle unordered commands (-1) - they go to the end
		if orderI == UnorderedCommand && orderJ == UnorderedCommand {
			return false // Keep original order for both unordered
		}
		if orderI == UnorderedCommand {
			return false // i goes after j
		}
		if orderJ == UnorderedCommand {
			return true // i goes before j
		}
		
		return orderI < orderJ
	})
}

// Command ordering
var commandOrder = map[string]int{
	// Basic Commands
	"start":      1,
	"register":   2,
	"list":       3,
	"usage":      4,
	"invoke":     5,
	"disable":    6,
	"enable":     7,
	"deregister": 8,
	"version":    9,
	
	// Advanced Commands  
	"create":      1,
	"delete":      2,
	"init-server": 3,
}

const UnorderedCommand = -1

// getOrderValue gets the predefined order for a command, returns -1 for unordered commands
func getOrderValue(cmd *cobra.Command) int {
	if order, exists := commandOrder[cmd.Name()]; exists {
		return order
	}
	return UnorderedCommand // -1 indicates no specific order
}



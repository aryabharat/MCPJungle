package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get resources",
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "4",
	},
}

var getPromptArgs map[string]string

var getGroupCmd = &cobra.Command{
	Use:   "group [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Get information about a specific Tool Group",
	Long: "Get information about a specific Tool Group by name.\n" +
		"This returns the configuration of the Tool Group including which tools are included.\n",
	RunE: runGetGroup,
}

var getPromptCmd = &cobra.Command{
	Use:   "prompt [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Get a prompt template with arguments",
	Long: "Retrieve a prompt template from an MCP server with optional arguments.\n" +
		"The prompt will be rendered with the provided arguments and returned as structured messages.",
	Example: `  # Get a basic prompt
  mcpjungle get prompt github__code-review

  # Get a prompt with arguments
  mcpjungle get prompt github__code-review --arg code="def hello(): print('world')" --arg language="python"`,
	RunE: runGetPrompt,
}

func init() {
	getPromptCmd.Flags().StringToStringVar(
		&getPromptArgs,
		"arg",
		nil,
		"Arguments to pass to the prompt (can be specified multiple times)",
	)

	getCmd.AddCommand(getGroupCmd)
	getCmd.AddCommand(getPromptCmd)
	rootCmd.AddCommand(getCmd)
}

func runGetGroup(cmd *cobra.Command, args []string) error {
	name := args[0]
	group, err := apiClient.GetToolGroup(name)
	if err != nil {
		return fmt.Errorf("failed to get tool group: %w", err)
	}

	cmd.Println(group.Name)
	if group.Description != "" {
		cmd.Println()
		cmd.Println("Description: " + group.Description)
	}

	cmd.Println()
	cmd.Println("MCP Server endpoint: ", group.Endpoint)
	cmd.Println()

	if len(group.IncludedTools) == 0 {
		cmd.Println("This group has no tools.")
	} else {
		cmd.Println("Included Tools:")
		for i, t := range group.IncludedTools {
			cmd.Printf("%d. %s\n", i+1, t)
			// TODO: Also show whether the tool is still active, disabled, or deleted at the moment
			// ie, is it practically available as part of this group?
		}
	}
	cmd.Println()

	cmd.Println(
		"NOTE: If a tool in this group is disabled globally or has been deleted, " +
			"then it will not be available via the group's MCP endpoint.",
	)

	return nil
}

func runGetPrompt(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Convert CLI args to proper format
	arguments := make(map[string]string)
	for k, v := range getPromptArgs {
		arguments[k] = v
	}

	result, err := apiClient.GetPromptWithArgs(name, arguments)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	// Pretty print the result
	fmt.Printf("Prompt: %s\n", name)
	if result.Description != "" {
		fmt.Printf("Description: %s\n", result.Description)
	}
	fmt.Println("\nGenerated Messages:")
	fmt.Println("=" + strings.Repeat("=", 50))

	for i, message := range result.Messages {
		fmt.Printf("\nMessage %d (%s):\n", i+1, message.Role)
		fmt.Println("-" + strings.Repeat("-", 30))

		// Format the content nicely
		contentBytes, err := json.MarshalIndent(message.Content, "", "  ")
		if err != nil {
			fmt.Printf("Content: %+v\n", message.Content)
		} else {
			fmt.Printf("Content: %s\n", string(contentBytes))
		}
	}

	return nil
}

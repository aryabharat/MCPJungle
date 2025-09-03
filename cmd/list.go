package cmd

import (
	"fmt"
	"strings"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources like MCP servers, tools, etc",
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "3",
	},
}

var listToolsCmdServerName string
var listPromptsCmdServerName string

var listToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available tools",
	Long:  "List tools available either from a specific MCP server or across all MCP servers registered in the registry.",
	RunE:  runListTools,
}

var listPromptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "List available prompts",
	Long:  "List prompt templates available either from a specific MCP server or across all MCP servers registered in the registry.",
	RunE:  runListPrompts,
}

var listServersCmd = &cobra.Command{
	Use:   "servers",
	Short: "List registered MCP servers",
	RunE:  runListServers,
}

var listMcpClientsCmd = &cobra.Command{
	Use:   "mcp-clients",
	Short: "List MCP clients (Production mode)",
	Long: "List MCP clients that are authorized to access the MCP Proxy server.\n" +
		"This command is only available in Production mode.",
	RunE: runListMcpClients,
}

var listUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List users (Production mode)",
	Long:  "List users that are authorized to access MCPJungle.",
	RunE:  runListUsers,
}

var listGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "List tool groups",
	RunE:  runListGroups,
}

func init() {
	listToolsCmd.Flags().StringVar(
		&listToolsCmdServerName,
		"server",
		"",
		"Filter tools by server name",
	)

	listPromptsCmd.Flags().StringVar(
		&listPromptsCmdServerName,
		"server",
		"",
		"Filter prompts by server name",
	)

	listCmd.AddCommand(listToolsCmd)
	listCmd.AddCommand(listPromptsCmd)
	listCmd.AddCommand(listServersCmd)
	listCmd.AddCommand(listMcpClientsCmd)
	listCmd.AddCommand(listUsersCmd)
	listCmd.AddCommand(listGroupsCmd)

	rootCmd.AddCommand(listCmd)
}

func runListTools(cmd *cobra.Command, args []string) error {
	tools, err := apiClient.ListTools(listToolsCmdServerName)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if len(tools) == 0 {
		fmt.Println("There are no tools in the registry")
		return nil
	}
	for i, t := range tools {
		ed := "ENABLED"
		if !t.Enabled {
			ed = "DISABLED"
		}
		fmt.Printf("%d. %s  [%s]\n", i+1, t.Name, ed)
		fmt.Println(t.Description)
		fmt.Println()
	}

	fmt.Println("Run 'usage <tool name>' to see a tool's usage or 'invoke <tool name>' to call one")

	return nil
}

func runListServers(cmd *cobra.Command, args []string) error {
	servers, err := apiClient.ListServers()
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Println("There are no MCP servers in the registry")
		return nil
	}
	for i, s := range servers {
		fmt.Printf("%d. %s\n", i+1, s.Name)

		if s.Description != "" {
			fmt.Println(s.Description)
		}

		fmt.Println("Transport: " + s.Transport)

		t, _ := types.ValidateTransport(s.Transport)
		if t == types.TransportStreamableHTTP {
			fmt.Println("URL: " + s.URL)
		} else {
			if len(s.Args) > 0 {
				fmt.Println("Command: " + s.Command + " " + strings.Join(s.Args, " "))
			} else {
				fmt.Println("Command: " + s.Command)
			}

			if len(s.Env) > 0 {
				fmt.Printf("Environment variables: %s\n", s.Env)
			}
		}

		if i < len(servers)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runListMcpClients(cmd *cobra.Command, args []string) error {
	clients, err := apiClient.ListMcpClients()
	if err != nil {
		return fmt.Errorf("failed to list MCP clients: %w", err)
	}

	if len(clients) == 0 {
		fmt.Println("There are no MCP clients in the registry")
		return nil
	}
	for i, c := range clients {
		fmt.Printf("%d. %s\n", i+1, c.Name)

		if c.Description != "" {
			fmt.Println("Description: ", c.Description)
		}

		if len(c.AllowList) > 0 {
			fmt.Println("Allowed servers: " + strings.Join(c.AllowList, ","))
		} else {
			fmt.Println("This client does not have access to any MCP servers.")
		}

		if i < len(clients)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runListUsers(cmd *cobra.Command, args []string) error {
	users, err := apiClient.ListUsers()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(users) == 0 {
		cmd.Println("There are no users in the registry")
		return nil
	}
	for i, u := range users {
		if u.Role == string(types.UserRoleAdmin) {
			cmd.Printf("%d. %s  [ADMIN]\n", i+1, u.Username)
		} else {
			cmd.Printf("%d. %s\n", i+1, u.Username)
		}

		if i < len(users)-1 {
			cmd.Println()
		}
	}

	return nil
}

func runListGroups(cmd *cobra.Command, args []string) error {
	groups, err := apiClient.ListToolGroups()
	if err != nil {
		return fmt.Errorf("failed to list tool groups: %w", err)
	}

	if len(groups) == 0 {
		cmd.Println("There are no tool groups in the registry")
		return nil
	}
	for i, g := range groups {
		cmd.Printf("%d. %s\n", i+1, g.Name)
		if g.Description != "" {
			cmd.Println(g.Description)
		}

		if i < len(groups)-1 {
			cmd.Println()
		}
	}

	return nil
}

func runListPrompts(cmd *cobra.Command, args []string) error {
	prompts, err := apiClient.ListPrompts(listPromptsCmdServerName)
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	if len(prompts) == 0 {
		fmt.Println("There are no prompts in the registry")
		return nil
	}
	for i, p := range prompts {
		ed := "ENABLED"
		if !p.Enabled {
			ed = "DISABLED"
		}
		fmt.Printf("%d. %s  [%s]\n", i+1, p.Name, ed)
		fmt.Println(p.Description)
		fmt.Println()
	}

	fmt.Println("Run 'get prompt <prompt name>' to retrieve a prompt template")

	return nil
}

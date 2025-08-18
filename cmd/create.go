package cmd

import (
	"fmt"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
	"strings"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create resources",
	Annotations: map[string]string{
		"group": "advanced",
	},
}

var createMcpClientCmd = &cobra.Command{
	Use:   "mcp-client [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Create an authenticated MCP client (Production mode)",
	Long: "Create an MCP client that can make authenticated requests to the MCPJungle MCP Proxy.\n" +
		"This returns an access token which should be sent by your client in the " +
		"`Authorization: Bearer {token}` http header.\n" +
		"This also lets you control which MCO servers the client can access.\n" +
		"This command is only available in Production mode.",
	RunE: runCreateMcpClient,
}

var (
	createMcpClientCmdAllowedServers string
	createMcpClientCmdDescription    string
)

func init() {
	createMcpClientCmd.Flags().StringVar(
		&createMcpClientCmdAllowedServers,
		"allow",
		"",
		"Comma-separated list of MCP servers that this client is allowed to access.\n"+
			"By default, the list is empty, meaning the client cannot access any MCP servers.",
	)
	createMcpClientCmd.Flags().StringVar(
		&createMcpClientCmdDescription,
		"description",
		"",
		"Description of the MCP client. This is optional and can be used to provide additional context.",
	)

	createCmd.AddCommand(createMcpClientCmd)
	rootCmd.AddCommand(createCmd)
}

func runCreateMcpClient(cmd *cobra.Command, args []string) error {
	// convert the comma-separated list of allowed servers into a slice
	allowList := make([]string, 0)
	for _, s := range strings.Split(createMcpClientCmdAllowedServers, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			allowList = append(allowList, trimmed)
		}
	}

	c := &types.McpClient{
		Name:        args[0],
		Description: createMcpClientCmdDescription,
		AllowList:   allowList,
	}

	token, err := apiClient.CreateMcpClient(c)
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("server returned an empty token, this was unexpected")
	}

	fmt.Printf("MCP client '%s' created successfully!\n", c.Name)

	if len(c.AllowList) > 0 {
		fmt.Println("Servers accessible: " + strings.Join(c.AllowList, ","))
	} else {
		fmt.Println("This client does not have access to any MCP servers.")
	}

	fmt.Printf("\nAccess token: %s\n", token)
	fmt.Println("Your client should send this token in the `Authorization: Bearer {token}` HTTP header.")

	return nil
}

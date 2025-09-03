package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

// initMCPProxyServer initializes the MCP proxy server.
// It loads all the registered MCP tools and prompts from the database into the proxy server.
func (m *MCPService) initMCPProxyServer() error {
	// Load tools
	tools, err := m.ListTools()
	if err != nil {
		return fmt.Errorf("failed to list tools from DB: %w", err)
	}
	for _, tm := range tools {
		if !tm.Enabled {
			// do not add disabled tools to the proxy
			continue
		}

		// Add tool to the MCP proxy server
		tool, err := convertToolModelToMcpObject(&tm)
		if err != nil {
			return fmt.Errorf("failed to convert tool model to MCP object for tool %s: %w", tm.Name, err)
		}

		m.mcpProxyServer.AddTool(tool, m.MCPProxyToolCallHandler)
	}

	// Load prompts
	prompts, err := m.ListPrompts()
	if err != nil {
		return fmt.Errorf("failed to list prompts from DB: %w", err)
	}
	for _, pm := range prompts {
		if !pm.Enabled {
			// do not add disabled prompts to the proxy
			continue
		}

		// Add prompt to the MCP proxy server
		prompt, err := convertPromptModelToMcpObject(&pm)
		if err != nil {
			return fmt.Errorf("failed to convert prompt model to MCP object for prompt %s: %w", pm.Name, err)
		}

		m.mcpProxyServer.AddPrompt(prompt, m.mcpProxyPromptHandler)
	}

	return nil
}

// MCPProxyToolCallHandler handles tool calls for the MCP proxy server
// by forwarding the request to the appropriate upstream MCP server and
// relaying the response back.
func (m *MCPService) MCPProxyToolCallHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.Params.Name
	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: tool name does not contain a %s separator", serverToolNameSep)
	}

	serverMode := ctx.Value("mode").(model.ServerMode)
	if serverMode == model.ModeProd {
		// In production mode, we need to check whether the MCP client is authorized to access the MCP server.
		// If not, return error Unauthorized.
		c := ctx.Value("client").(*model.McpClient)
		if !c.CheckHasServerAccess(serverName) {
			return nil, fmt.Errorf(
				"client %s is not authorized to access MCP server %s", c.Name, serverName,
			)
		}
	}

	// get the MCP server details from the database
	server, err := m.GetMcpServer(serverName)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w", serverName, err,
		)
	}

	mcpClient, err := newMcpServerSession(ctx, server)
	if err != nil {
		return nil, err
	}
	defer mcpClient.Close()

	// Ensure the tool name is set correctly, ie, without the server name prefix
	request.Params.Name = toolName

	// forward the request to the upstream MCP server and relay the response back
	return mcpClient.CallTool(ctx, request)
}

// mcpProxyPromptHandler handles prompt requests for the MCP proxy server
// by forwarding the request to the appropriate upstream MCP server and
// relaying the response back.
func (m *MCPService) mcpProxyPromptHandler(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	name := request.Params.Name
	serverName, promptName, ok := splitServerPromptName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: prompt name does not contain a %s separator", serverPromptNameSep)
	}

	serverMode := ctx.Value("mode").(model.ServerMode)
	if serverMode == model.ModeProd {
		// In production mode, we need to check whether the MCP client is authorized to access the MCP server.
		// If not, return error Unauthorized.
		c := ctx.Value("client").(*model.McpClient)
		if !c.CheckHasServerAccess(serverName) {
			return nil, fmt.Errorf(
				"client %s is not authorized to access MCP server %s", c.Name, serverName,
			)
		}
	}

	// get the MCP server details from the database
	server, err := m.GetMcpServer(serverName)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w", serverName, err,
		)
	}

	mcpClient, err := newMcpServerSession(ctx, server)
	if err != nil {
		return nil, err
	}
	defer mcpClient.Close()

	// Ensure the prompt name is set correctly, ie, without the server name prefix
	request.Params.Name = promptName

	// forward the request to the upstream MCP server and relay the response back
	return mcpClient.GetPrompt(ctx, request)
}

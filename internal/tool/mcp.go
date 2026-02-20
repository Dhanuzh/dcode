package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// MCPTool provides Model Context Protocol client operations
func MCPTool() *ToolDef {
	return &ToolDef{
		Name:        "MCP",
		Description: "Connect to MCP servers for external tools. Supports HTTP, SSE, and process-based.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type": "string",
					"description": "MCP operation to perform",
					"enum": []string{
						"list_servers",  // List configured MCP servers
						"list_tools",    // List tools from an MCP server
						"call_tool",     // Call a tool on an MCP server
						"get_resource",  // Get a resource from an MCP server
						"list_resources", // List available resources
					},
				},
				"server": map[string]interface{}{
					"type":        "string",
					"description": "MCP server name (from config)",
				},
				"tool": map[string]interface{}{
					"type":        "string",
					"description": "Tool name to call",
				},
				"arguments": map[string]interface{}{
					"type":        "object",
					"description": "Arguments to pass to the tool",
				},
				"resource": map[string]interface{}{
					"type":        "string",
					"description": "Resource URI to fetch",
				},
			},
			"required": []string{"operation"},
		},
		Execute: executeMCP,
	}
}

func executeMCP(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	operation, ok := input["operation"].(string)
	if !ok {
		return &ToolResult{
			Output:  "operation parameter is required",
			IsError: true,
		}, nil
	}

	switch operation {
	case "list_servers":
		return listMCPServers()
	case "list_tools":
		serverName, ok := input["server"].(string)
		if !ok {
			return &ToolResult{
				Output:  "server parameter is required for list_tools",
				IsError: true,
			}, nil
		}
		return listMCPTools(ctx, serverName)
	case "call_tool":
		serverName, ok := input["server"].(string)
		if !ok {
			return &ToolResult{
				Output:  "server parameter is required for call_tool",
				IsError: true,
			}, nil
		}
		toolName, ok := input["tool"].(string)
		if !ok {
			return &ToolResult{
				Output:  "tool parameter is required for call_tool",
				IsError: true,
			}, nil
		}
		args, _ := input["arguments"].(map[string]interface{})
		return callMCPTool(ctx, serverName, toolName, args)
	case "get_resource":
		serverName, ok := input["server"].(string)
		if !ok {
			return &ToolResult{
				Output:  "server parameter is required for get_resource",
				IsError: true,
			}, nil
		}
		resourceURI, ok := input["resource"].(string)
		if !ok {
			return &ToolResult{
				Output:  "resource parameter is required for get_resource",
				IsError: true,
			}, nil
		}
		return getMCPResource(ctx, serverName, resourceURI)
	case "list_resources":
		serverName, ok := input["server"].(string)
		if !ok {
			return &ToolResult{
				Output:  "server parameter is required for list_resources",
				IsError: true,
			}, nil
		}
		return listMCPResources(ctx, serverName)
	default:
		return &ToolResult{
			Output:  fmt.Sprintf("unknown MCP operation: %s", operation),
			IsError: true,
		}, nil
	}
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"` // "http", "sse", "process"
	URL     string            `json:"url,omitempty"`
	Command []string          `json:"command,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Status  string            `json:"status"`
}

// listMCPServers lists configured MCP servers
func listMCPServers() (*ToolResult, error) {
	// This would read from config in a real implementation
	// For now, return example servers
	servers := []MCPServer{
		{
			Name:   "example-http",
			Type:   "http",
			URL:    "http://localhost:3000/mcp",
			Status: "not configured",
		},
		{
			Name:    "example-process",
			Type:    "process",
			Command: []string{"node", "mcp-server.js"},
			Status:  "not configured",
		},
	}

	output := "# Configured MCP Servers\n\n"
	output += "Note: MCP servers must be configured in dcode.yaml under the 'mcp' section.\n\n"
	output += "## Example Configuration:\n"
	output += "```yaml\n"
	output += "mcp:\n"
	output += "  filesystem:\n"
	output += "    type: process\n"
	output += "    command: [\"npx\", \"-y\", \"@modelcontextprotocol/server-filesystem\", \"/path/to/allowed/files\"]\n"
	output += "  github:\n"
	output += "    type: http\n"
	output += "    url: \"https://mcp.github.com\"\n"
	output += "    env:\n"
	output += "      GITHUB_TOKEN: \"your-token\"\n"
	output += "```\n\n"

	if len(servers) > 0 {
		output += "## Currently Configured:\n"
		for _, server := range servers {
			output += fmt.Sprintf("- **%s** (%s) - %s\n", server.Name, server.Type, server.Status)
			if server.URL != "" {
				output += fmt.Sprintf("  URL: %s\n", server.URL)
			}
			if len(server.Command) > 0 {
				output += fmt.Sprintf("  Command: %s\n", strings.Join(server.Command, " "))
			}
		}
	}

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// listMCPTools lists tools available from an MCP server
func listMCPTools(ctx context.Context, serverName string) (*ToolResult, error) {
	// Try to connect to the server and list tools
	server, err := getMCPServerConfig(serverName)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("MCP server not found: %s\n\nUse 'list_servers' to see available servers.", serverName),
			IsError: true,
		}, nil
	}

	switch server.Type {
	case "http":
		return listToolsHTTP(ctx, server)
	case "process":
		return listToolsProcess(ctx, server)
	default:
		return &ToolResult{
			Output:  fmt.Sprintf("unsupported MCP server type: %s", server.Type),
			IsError: true,
		}, nil
	}
}

// listToolsHTTP lists tools from an HTTP MCP server
func listToolsHTTP(ctx context.Context, server *MCPServer) (*ToolResult, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to marshal request: %v", err),
			IsError: true,
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewReader(jsonData))
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to create request: %v", err),
			IsError: true,
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to connect to MCP server: %v", err),
			IsError: true,
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to read response: %v", err),
			IsError: true,
		}, nil
	}

	var mcpResp struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &mcpResp); err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to parse response: %v\nResponse: %s", err, string(body)),
			IsError: true,
		}, nil
	}

	output := fmt.Sprintf("# Tools from MCP Server: %s\n\n", server.Name)
	for _, tool := range mcpResp.Result.Tools {
		output += fmt.Sprintf("- **%s**: %s\n", tool.Name, tool.Description)
	}

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// listToolsProcess lists tools from a process-based MCP server
func listToolsProcess(ctx context.Context, server *MCPServer) (*ToolResult, error) {
	// Start the MCP server process
	cmd := exec.CommandContext(ctx, server.Command[0], server.Command[1:]...)

	// Set environment variables
	for k, v := range server.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Send tools/list request via stdin and read from stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to create stdin pipe: %v", err),
			IsError: true,
		}, nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to create stdout pipe: %v", err),
			IsError: true,
		}, nil
	}

	if err := cmd.Start(); err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to start MCP server: %v", err),
			IsError: true,
		}, nil
	}

	// Send tools/list request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	if err := json.NewEncoder(stdin).Encode(reqBody); err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to send request: %v", err),
			IsError: true,
		}, nil
	}
	stdin.Close()

	// Read response
	var mcpResp struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}

	if err := json.NewDecoder(stdout).Decode(&mcpResp); err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to parse response: %v", err),
			IsError: true,
		}, nil
	}

	cmd.Wait()

	output := fmt.Sprintf("# Tools from MCP Server: %s\n\n", server.Name)
	for _, tool := range mcpResp.Result.Tools {
		output += fmt.Sprintf("- **%s**: %s\n", tool.Name, tool.Description)
	}

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// callMCPTool calls a tool on an MCP server
func callMCPTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (*ToolResult, error) {
	server, err := getMCPServerConfig(serverName)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("MCP server not found: %s", serverName),
			IsError: true,
		}, nil
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	switch server.Type {
	case "http":
		return callToolHTTP(ctx, server, reqBody)
	case "process":
		return callToolProcess(ctx, server, reqBody)
	default:
		return &ToolResult{
			Output:  fmt.Sprintf("unsupported MCP server type: %s", server.Type),
			IsError: true,
		}, nil
	}
}

// callToolHTTP calls a tool via HTTP
func callToolHTTP(ctx context.Context, server *MCPServer, reqBody map[string]interface{}) (*ToolResult, error) {
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("MCP tool call failed: %v", err),
			IsError: true,
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var mcpResp struct {
		Result interface{} `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	json.Unmarshal(body, &mcpResp)

	if mcpResp.Error != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("MCP error: %s", mcpResp.Error.Message),
			IsError: true,
		}, nil
	}

	resultJSON, _ := json.MarshalIndent(mcpResp.Result, "", "  ")
	return &ToolResult{
		Output:  string(resultJSON),
		IsError: false,
	}, nil
}

// callToolProcess calls a tool via process
func callToolProcess(ctx context.Context, server *MCPServer, reqBody map[string]interface{}) (*ToolResult, error) {
	// Similar to listToolsProcess but with tool call
	return &ToolResult{
		Output:  "Process-based tool calls not yet fully implemented",
		IsError: true,
	}, nil
}

// getMCPResource gets a resource from an MCP server
func getMCPResource(ctx context.Context, serverName, resourceURI string) (*ToolResult, error) {
	return &ToolResult{
		Output:  "MCP resources not yet implemented",
		IsError: true,
	}, nil
}

// listMCPResources lists available resources
func listMCPResources(ctx context.Context, serverName string) (*ToolResult, error) {
	return &ToolResult{
		Output:  "MCP resources not yet implemented",
		IsError: true,
	}, nil
}

// getMCPServerConfig gets MCP server config from configuration
func getMCPServerConfig(name string) (*MCPServer, error) {
	// In a real implementation, this would load from config
	// For now, return a not found error
	return nil, fmt.Errorf("server not found: %s", name)
}

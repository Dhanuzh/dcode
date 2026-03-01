package tool

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Dhanuzh/dcode/internal/config"
)

// ─── Global MCP server config registry ──────────────────────────────────────

var (
	mcpConfigMu     sync.RWMutex
	mcpServerConfig map[string]*MCPServer // name → config, set at startup
)

// SetMCPConfig wires the config-loaded MCP map into the tool.
// Call this from main after config.Load().
func SetMCPConfig(cfg map[string]interface{}) {
	mcpConfigMu.Lock()
	defer mcpConfigMu.Unlock()
	mcpServerConfig = make(map[string]*MCPServer)
	for name, raw := range cfg {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		srv := &MCPServer{Name: name}
		if t, ok := m["type"].(string); ok {
			srv.Type = t
		}
		if u, ok := m["url"].(string); ok {
			srv.URL = u
			if srv.Type == "" {
				srv.Type = "http"
			}
		}
		if rawCmd, ok := m["command"]; ok {
			switch v := rawCmd.(type) {
			case []interface{}:
				for _, c := range v {
					if s, ok := c.(string); ok {
						srv.Command = append(srv.Command, s)
					}
				}
			case []string:
				srv.Command = v
			}
			if srv.Type == "" {
				srv.Type = "process"
			}
		}
		if rawEnv, ok := m["env"].(map[string]interface{}); ok {
			srv.Env = make(map[string]string)
			for k, v := range rawEnv {
				if s, ok := v.(string); ok {
					srv.Env[k] = s
				}
			}
		}
		if rawHeaders, ok := m["headers"].(map[string]interface{}); ok {
			srv.Headers = make(map[string]string)
			for k, v := range rawHeaders {
				if s, ok := v.(string); ok {
					srv.Headers[k] = s
				}
			}
		}
		srv.Status = "configured"
		mcpServerConfig[name] = srv
	}
}

// SetMCPConfigFromConfig wires the config.Config.MCP map into the MCP tool.
// This is the primary way to configure MCP servers from the app config.
func SetMCPConfigFromConfig(cfg *config.Config) {
	if cfg == nil || len(cfg.MCP) == 0 {
		return
	}
	entries := make(map[string]MCPServerEntry, len(cfg.MCP))
	for name, mc := range cfg.MCP {
		// Skip disabled servers
		if mc.Enabled != nil && !*mc.Enabled {
			continue
		}
		entries[name] = MCPServerEntry{
			Type:    mc.Type,
			Command: mc.Command,
			URL:     mc.URL,
			Env:     mc.Env,
			Headers: mc.Headers,
		}
	}
	SetMCPConfigTyped(entries)
}

// SetMCPConfigTyped wires typed config map (config.MCPConfig).
// Accepts map[string]MCPConfigEntry where MCPConfigEntry has the same fields.
func SetMCPConfigTyped(servers map[string]MCPServerEntry) {
	mcpConfigMu.Lock()
	defer mcpConfigMu.Unlock()
	mcpServerConfig = make(map[string]*MCPServer)
	for name, entry := range servers {
		srv := &MCPServer{
			Name:    name,
			Type:    entry.Type,
			URL:     entry.URL,
			Command: entry.Command,
			Env:     entry.Env,
			Headers: entry.Headers,
			Status:  "configured",
		}
		if srv.Type == "" {
			if srv.URL != "" {
				srv.Type = "http"
			} else if len(srv.Command) > 0 {
				srv.Type = "process"
			}
		}
		mcpServerConfig[name] = srv
	}
}

// MCPServerEntry is the typed config struct passed from config package.
type MCPServerEntry struct {
	Type    string // "local" | "remote" | "http" | "process"
	Command []string
	URL     string
	Env     map[string]string
	Headers map[string]string
}

// MCPTool provides Model Context Protocol client operations
func MCPTool() *ToolDef {
	return &ToolDef{
		Name:        "MCP",
		Description: "Connect to MCP servers for external tools. Supports HTTP, SSE, and process-based.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "MCP operation to perform",
					"enum": []string{
						"list_servers",
						"list_tools",
						"call_tool",
						"get_resource",
						"list_resources",
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
		return &ToolResult{Output: "operation parameter is required", IsError: true}, nil
	}

	switch operation {
	case "list_servers":
		return listMCPServers()
	case "list_tools":
		serverName, ok := input["server"].(string)
		if !ok {
			return &ToolResult{Output: "server parameter is required for list_tools", IsError: true}, nil
		}
		return listMCPTools(ctx, serverName)
	case "call_tool":
		serverName, ok := input["server"].(string)
		if !ok {
			return &ToolResult{Output: "server parameter is required for call_tool", IsError: true}, nil
		}
		toolName, ok := input["tool"].(string)
		if !ok {
			return &ToolResult{Output: "tool parameter is required for call_tool", IsError: true}, nil
		}
		args, _ := input["arguments"].(map[string]interface{})
		return callMCPTool(ctx, serverName, toolName, args)
	case "get_resource":
		serverName, ok := input["server"].(string)
		if !ok {
			return &ToolResult{Output: "server parameter is required for get_resource", IsError: true}, nil
		}
		resourceURI, ok := input["resource"].(string)
		if !ok {
			return &ToolResult{Output: "resource parameter is required for get_resource", IsError: true}, nil
		}
		return getMCPResource(ctx, serverName, resourceURI)
	case "list_resources":
		serverName, ok := input["server"].(string)
		if !ok {
			return &ToolResult{Output: "server parameter is required for list_resources", IsError: true}, nil
		}
		return listMCPResources(ctx, serverName)
	default:
		return &ToolResult{Output: fmt.Sprintf("unknown MCP operation: %s", operation), IsError: true}, nil
	}
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"` // "http", "sse", "process"
	URL     string            `json:"url,omitempty"`
	Command []string          `json:"command,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Status  string            `json:"status"`
}

// getMCPServerConfig retrieves an MCP server from the config registry.
func getMCPServerConfig(name string) (*MCPServer, error) {
	mcpConfigMu.RLock()
	defer mcpConfigMu.RUnlock()
	if mcpServerConfig == nil {
		return nil, fmt.Errorf("no MCP servers configured (add 'mcp' section to dcode.yaml)")
	}
	srv, ok := mcpServerConfig[name]
	if !ok {
		names := make([]string, 0, len(mcpServerConfig))
		for k := range mcpServerConfig {
			names = append(names, k)
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("no MCP servers configured")
		}
		return nil, fmt.Errorf("MCP server %q not found; configured: %s", name, strings.Join(names, ", "))
	}
	return srv, nil
}

// listMCPServers lists all configured MCP servers.
func listMCPServers() (*ToolResult, error) {
	mcpConfigMu.RLock()
	defer mcpConfigMu.RUnlock()

	if len(mcpServerConfig) == 0 {
		output := "# No MCP Servers Configured\n\n"
		output += "Add servers to your dcode.yaml:\n\n"
		output += "```yaml\n"
		output += "mcp:\n"
		output += "  filesystem:\n"
		output += "    type: process\n"
		output += "    command: [\"npx\", \"-y\", \"@modelcontextprotocol/server-filesystem\", \"/path\"]\n"
		output += "  github:\n"
		output += "    type: http\n"
		output += "    url: \"https://mcp.github.com\"\n"
		output += "    headers:\n"
		output += "      Authorization: \"Bearer your-token\"\n"
		output += "```\n"
		return &ToolResult{Output: output}, nil
	}

	output := "# Configured MCP Servers\n\n"
	for _, srv := range mcpServerConfig {
		output += fmt.Sprintf("## %s (%s) — %s\n", srv.Name, srv.Type, srv.Status)
		if srv.URL != "" {
			output += fmt.Sprintf("  URL: %s\n", srv.URL)
		}
		if len(srv.Command) > 0 {
			output += fmt.Sprintf("  Command: %s\n", strings.Join(srv.Command, " "))
		}
	}
	return &ToolResult{Output: output}, nil
}

// listMCPTools lists tools from an MCP server.
func listMCPTools(ctx context.Context, serverName string) (*ToolResult, error) {
	server, err := getMCPServerConfig(serverName)
	if err != nil {
		return &ToolResult{Output: err.Error(), IsError: true}, nil
	}

	switch server.Type {
	case "http", "remote":
		return listToolsHTTP(ctx, server)
	case "process", "local":
		return listToolsProcess(ctx, server)
	default:
		return &ToolResult{Output: fmt.Sprintf("unsupported MCP server type: %s", server.Type), IsError: true}, nil
	}
}

// listToolsHTTP lists tools from an HTTP MCP server.
func listToolsHTTP(ctx context.Context, server *MCPServer) (*ToolResult, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("failed to marshal request: %v", err), IsError: true}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewReader(jsonData))
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("failed to create request: %v", err), IsError: true}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range server.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("failed to connect to MCP server %s: %v", server.Name, err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var mcpResp struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
		Error *mcpError `json:"error"`
	}

	if err := json.Unmarshal(body, &mcpResp); err != nil {
		return &ToolResult{Output: fmt.Sprintf("failed to parse response: %v\nBody: %s", err, string(body)), IsError: true}, nil
	}
	if mcpResp.Error != nil {
		return &ToolResult{Output: fmt.Sprintf("MCP error: %s", mcpResp.Error.Message), IsError: true}, nil
	}

	output := fmt.Sprintf("# Tools from MCP Server: %s\n\n", server.Name)
	for _, t := range mcpResp.Result.Tools {
		output += fmt.Sprintf("- **%s**: %s\n", t.Name, t.Description)
	}
	if len(mcpResp.Result.Tools) == 0 {
		output += "(no tools returned)\n"
	}
	return &ToolResult{Output: output}, nil
}

// listToolsProcess lists tools from a process-based MCP server (stdio JSON-RPC).
func listToolsProcess(ctx context.Context, server *MCPServer) (*ToolResult, error) {
	if len(server.Command) == 0 {
		return &ToolResult{Output: "process MCP server has no command configured", IsError: true}, nil
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	out, err := runMCPProcess(ctx, server, reqBody)
	if err != nil {
		return &ToolResult{Output: err.Error(), IsError: true}, nil
	}

	var mcpResp struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
		Error *mcpError `json:"error"`
	}

	if err := json.Unmarshal(out, &mcpResp); err != nil {
		return &ToolResult{Output: fmt.Sprintf("failed to parse response: %v", err), IsError: true}, nil
	}
	if mcpResp.Error != nil {
		return &ToolResult{Output: fmt.Sprintf("MCP error: %s", mcpResp.Error.Message), IsError: true}, nil
	}

	output := fmt.Sprintf("# Tools from MCP Server: %s\n\n", server.Name)
	for _, t := range mcpResp.Result.Tools {
		output += fmt.Sprintf("- **%s**: %s\n", t.Name, t.Description)
	}
	return &ToolResult{Output: output}, nil
}

// callMCPTool calls a tool on an MCP server.
func callMCPTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (*ToolResult, error) {
	server, err := getMCPServerConfig(serverName)
	if err != nil {
		return &ToolResult{Output: err.Error(), IsError: true}, nil
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
	case "http", "remote":
		return callToolHTTP(ctx, server, reqBody)
	case "process", "local":
		return callToolProcess(ctx, server, reqBody)
	default:
		return &ToolResult{Output: fmt.Sprintf("unsupported MCP server type: %s", server.Type), IsError: true}, nil
	}
}

// callToolHTTP calls a tool via HTTP MCP server.
func callToolHTTP(ctx context.Context, server *MCPServer, reqBody map[string]interface{}) (*ToolResult, error) {
	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewReader(jsonData))
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("request error: %v", err), IsError: true}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range server.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("MCP tool call failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var mcpResp struct {
		Result interface{} `json:"result"`
		Error  *mcpError   `json:"error"`
	}
	json.Unmarshal(body, &mcpResp)

	if mcpResp.Error != nil {
		return &ToolResult{Output: fmt.Sprintf("MCP error: %s", mcpResp.Error.Message), IsError: true}, nil
	}

	resultJSON, _ := json.MarshalIndent(mcpResp.Result, "", "  ")
	return &ToolResult{Output: string(resultJSON)}, nil
}

// callToolProcess calls a tool on a process-based MCP server via stdio JSON-RPC.
func callToolProcess(ctx context.Context, server *MCPServer, reqBody map[string]interface{}) (*ToolResult, error) {
	out, err := runMCPProcess(ctx, server, reqBody)
	if err != nil {
		return &ToolResult{Output: err.Error(), IsError: true}, nil
	}

	var mcpResp struct {
		Result interface{} `json:"result"`
		Error  *mcpError   `json:"error"`
	}
	json.Unmarshal(out, &mcpResp)

	if mcpResp.Error != nil {
		return &ToolResult{Output: fmt.Sprintf("MCP error: %s", mcpResp.Error.Message), IsError: true}, nil
	}

	resultJSON, _ := json.MarshalIndent(mcpResp.Result, "", "  ")
	return &ToolResult{Output: string(resultJSON)}, nil
}

// getMCPResource fetches a resource from an MCP server.
func getMCPResource(ctx context.Context, serverName, resourceURI string) (*ToolResult, error) {
	server, err := getMCPServerConfig(serverName)
	if err != nil {
		return &ToolResult{Output: err.Error(), IsError: true}, nil
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/read",
		"params":  map[string]interface{}{"uri": resourceURI},
	}

	switch server.Type {
	case "http", "remote":
		return callToolHTTP(ctx, server, reqBody)
	case "process", "local":
		return callToolProcess(ctx, server, reqBody)
	default:
		return &ToolResult{Output: fmt.Sprintf("unsupported MCP server type: %s", server.Type), IsError: true}, nil
	}
}

// listMCPResources lists available resources from an MCP server.
func listMCPResources(ctx context.Context, serverName string) (*ToolResult, error) {
	server, err := getMCPServerConfig(serverName)
	if err != nil {
		return &ToolResult{Output: err.Error(), IsError: true}, nil
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/list",
		"params":  map[string]interface{}{},
	}

	switch server.Type {
	case "http", "remote":
		return callToolHTTP(ctx, server, reqBody)
	case "process", "local":
		return callToolProcess(ctx, server, reqBody)
	default:
		return &ToolResult{Output: fmt.Sprintf("unsupported MCP server type: %s", server.Type), IsError: true}, nil
	}
}

// ─── JSON-RPC process helper ─────────────────────────────────────────────────

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// runMCPProcess runs a process-based MCP server, sends one JSON-RPC request,
// and reads back the first complete JSON line response.
func runMCPProcess(ctx context.Context, server *MCPServer, reqBody map[string]interface{}) ([]byte, error) {
	if len(server.Command) == 0 {
		return nil, fmt.Errorf("no command configured for MCP server %s", server.Name)
	}

	// Allow up to 30 seconds for the process
	ctx30, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx30, server.Command[0], server.Command[1:]...)

	// Inherit environment and add server-specific env vars
	cmd.Env = os.Environ()
	for k, v := range server.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server %s: %v", server.Name, err)
	}
	defer cmd.Wait() //nolint:errcheck

	// Send the request
	if err := json.NewEncoder(stdin).Encode(reqBody); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	stdin.Close()

	// Read the first JSON line (MCP servers use newline-delimited JSON)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MB buffer
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Validate it's JSON
		if line[0] == '{' {
			return []byte(line), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading MCP server output: %v", err)
	}
	return nil, fmt.Errorf("no response from MCP server %s", server.Name)
}

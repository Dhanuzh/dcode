package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// LSPTool provides Language Server Protocol operations for code intelligence
func LSPTool() *ToolDef {
	return &ToolDef{
		Name:        "LSP",
		Description: "Query language servers for definitions, references, hover info, symbols, and diagnostics.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type": "string",
					"description": "LSP operation to perform",
					"enum": []string{
						"definition",     // Go to definition
						"references",     // Find references
						"hover",          // Get hover information
						"symbols",        // List document symbols
						"workspace_symbols", // Search workspace symbols
						"completion",     // Code completion
						"diagnostics",    // Get diagnostics (errors/warnings)
						"format",         // Format document
						"rename",         // Rename symbol
						"server_info",    // Get available language servers
					},
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "File path for the operation",
				},
				"line": map[string]interface{}{
					"type":        "number",
					"description": "Line number (1-indexed) for position-based operations",
				},
				"column": map[string]interface{}{
					"type":        "number",
					"description": "Column number (1-indexed) for position-based operations",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for workspace symbols",
				},
				"new_name": map[string]interface{}{
					"type":        "string",
					"description": "New name for rename operation",
				},
			},
			"required": []string{"operation"},
		},
		Execute: executeLSP,
	}
}

func executeLSP(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	operation, ok := input["operation"].(string)
	if !ok {
		return &ToolResult{
			Output:  "operation parameter is required",
			IsError: true,
		}, nil
	}

	// Handle server_info specially - doesn't need file
	if operation == "server_info" {
		return getServerInfo(tc.WorkDir)
	}

	// Get file path
	file, ok := input["file"].(string)
	if !ok && operation != "workspace_symbols" {
		return &ToolResult{
			Output:  "file parameter is required for this operation",
			IsError: true,
		}, nil
	}

	// Make file path absolute if relative
	if file != "" && !filepath.IsAbs(file) {
		file = filepath.Join(tc.WorkDir, file)
	}

	// Detect language server based on file extension
	var lspServer string
	if file != "" {
		lspServer = detectLanguageServer(file)
		if lspServer == "" {
			return &ToolResult{
				Output:  fmt.Sprintf("no language server found for file: %s", file),
				IsError: true,
			}, nil
		}
	}

	// Execute operation
	switch operation {
	case "definition":
		return executeDefinition(ctx, tc, lspServer, file, input)
	case "references":
		return executeReferences(ctx, tc, lspServer, file, input)
	case "hover":
		return executeHover(ctx, tc, lspServer, file, input)
	case "symbols":
		return executeSymbols(ctx, tc, lspServer, file)
	case "workspace_symbols":
		return executeWorkspaceSymbols(ctx, tc, input)
	case "completion":
		return executeCompletion(ctx, tc, lspServer, file, input)
	case "diagnostics":
		return executeDiagnostics(ctx, tc, lspServer, file)
	case "format":
		return executeFormat(ctx, tc, lspServer, file)
	case "rename":
		return executeRename(ctx, tc, lspServer, file, input)
	default:
		return &ToolResult{
			Output:  fmt.Sprintf("unknown LSP operation: %s", operation),
			IsError: true,
		}, nil
	}
}

// detectLanguageServer detects which LSP server to use based on file extension
func detectLanguageServer(file string) string {
	ext := strings.ToLower(filepath.Ext(file))

	servers := map[string]string{
		".go":   "gopls",
		".ts":   "typescript-language-server",
		".tsx":  "typescript-language-server",
		".js":   "typescript-language-server",
		".jsx":  "typescript-language-server",
		".py":   "pylsp",
		".rs":   "rust-analyzer",
		".c":    "clangd",
		".cpp":  "clangd",
		".cc":   "clangd",
		".h":    "clangd",
		".hpp":  "clangd",
		".java": "jdtls",
		".rb":   "solargraph",
		".php":  "phpactor",
		".lua":  "lua-language-server",
	}

	if server, ok := servers[ext]; ok {
		// Check if server is available
		if isCommandAvailable(server) {
			return server
		}
	}

	return ""
}

// executeDefinition finds the definition of a symbol
func executeDefinition(ctx context.Context, tc *ToolContext, lspServer, file string, input map[string]interface{}) (*ToolResult, error) {
	line, _ := input["line"].(float64)
	column, _ := input["column"].(float64)

	if line == 0 || column == 0 {
		return &ToolResult{
			Output:  "line and column parameters are required for definition",
			IsError: true,
		}, nil
	}

	// For gopls, use the definition command
	if lspServer == "gopls" {
		cmd := exec.CommandContext(ctx, "gopls", "definition",
			fmt.Sprintf("%s:%d:%d", file, int(line), int(column)))
		cmd.Dir = tc.WorkDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return &ToolResult{
				Output:  fmt.Sprintf("gopls error: %s\nOutput: %s", err, string(output)),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Output:  string(output),
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Output:  fmt.Sprintf("definition not yet implemented for %s", lspServer),
		IsError: true,
	}, nil
}

// executeReferences finds all references to a symbol
func executeReferences(ctx context.Context, tc *ToolContext, lspServer, file string, input map[string]interface{}) (*ToolResult, error) {
	line, _ := input["line"].(float64)
	column, _ := input["column"].(float64)

	if line == 0 || column == 0 {
		return &ToolResult{
			Output:  "line and column parameters are required for references",
			IsError: true,
		}, nil
	}

	if lspServer == "gopls" {
		cmd := exec.CommandContext(ctx, "gopls", "references",
			fmt.Sprintf("%s:%d:%d", file, int(line), int(column)))
		cmd.Dir = tc.WorkDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return &ToolResult{
				Output:  fmt.Sprintf("gopls error: %s\nOutput: %s", err, string(output)),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Output:  string(output),
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Output:  fmt.Sprintf("references not yet implemented for %s", lspServer),
		IsError: true,
	}, nil
}

// executeHover gets hover information for a symbol
func executeHover(ctx context.Context, tc *ToolContext, lspServer, file string, input map[string]interface{}) (*ToolResult, error) {
	line, _ := input["line"].(float64)
	column, _ := input["column"].(float64)

	if line == 0 || column == 0 {
		return &ToolResult{
			Output:  "line and column parameters are required for hover",
			IsError: true,
		}, nil
	}

	if lspServer == "gopls" {
		cmd := exec.CommandContext(ctx, "gopls", "hover",
			fmt.Sprintf("%s:%d:%d", file, int(line), int(column)))
		cmd.Dir = tc.WorkDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return &ToolResult{
				Output:  fmt.Sprintf("gopls error: %s\nOutput: %s", err, string(output)),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Output:  string(output),
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Output:  fmt.Sprintf("hover not yet implemented for %s", lspServer),
		IsError: true,
	}, nil
}

// executeSymbols lists symbols in a document
func executeSymbols(ctx context.Context, tc *ToolContext, lspServer, file string) (*ToolResult, error) {
	if lspServer == "gopls" {
		cmd := exec.CommandContext(ctx, "gopls", "symbols", file)
		cmd.Dir = tc.WorkDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return &ToolResult{
				Output:  fmt.Sprintf("gopls error: %s\nOutput: %s", err, string(output)),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Output:  string(output),
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Output:  fmt.Sprintf("symbols not yet implemented for %s", lspServer),
		IsError: true,
	}, nil
}

// executeWorkspaceSymbols searches for symbols in the workspace
func executeWorkspaceSymbols(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return &ToolResult{
			Output:  "query parameter is required for workspace symbols",
			IsError: true,
		}, nil
	}

	// Try gopls first if available
	if isCommandAvailable("gopls") {
		cmd := exec.CommandContext(ctx, "gopls", "workspace_symbol", query)
		cmd.Dir = tc.WorkDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return &ToolResult{
				Output:  fmt.Sprintf("gopls error: %s\nOutput: %s", err, string(output)),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Output:  string(output),
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Output:  "no language server available for workspace symbols",
		IsError: true,
	}, nil
}

// executeCompletion gets code completions
func executeCompletion(ctx context.Context, tc *ToolContext, lspServer, file string, input map[string]interface{}) (*ToolResult, error) {
	return &ToolResult{
		Output:  "completion is not yet implemented (requires interactive LSP session)",
		IsError: true,
	}, nil
}

// executeDiagnostics gets diagnostics for a file
func executeDiagnostics(ctx context.Context, tc *ToolContext, lspServer, file string) (*ToolResult, error) {
	if lspServer == "gopls" {
		cmd := exec.CommandContext(ctx, "gopls", "check", file)
		cmd.Dir = tc.WorkDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			// gopls check returns non-zero if there are errors, which is expected
			return &ToolResult{
				Output:  string(output),
				IsError: false,
			}, nil
		}

		return &ToolResult{
			Output:  string(output),
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Output:  fmt.Sprintf("diagnostics not yet implemented for %s", lspServer),
		IsError: true,
	}, nil
}

// executeFormat formats a document
func executeFormat(ctx context.Context, tc *ToolContext, lspServer, file string) (*ToolResult, error) {
	if lspServer == "gopls" {
		cmd := exec.CommandContext(ctx, "gopls", "format", file)
		cmd.Dir = tc.WorkDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return &ToolResult{
				Output:  fmt.Sprintf("gopls error: %s\nOutput: %s", err, string(output)),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Output:  string(output),
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Output:  fmt.Sprintf("format not yet implemented for %s", lspServer),
		IsError: true,
	}, nil
}

// executeRename renames a symbol
func executeRename(ctx context.Context, tc *ToolContext, lspServer, file string, input map[string]interface{}) (*ToolResult, error) {
	line, _ := input["line"].(float64)
	column, _ := input["column"].(float64)
	newName, ok := input["new_name"].(string)

	if line == 0 || column == 0 || !ok {
		return &ToolResult{
			Output:  "line, column, and new_name parameters are required for rename",
			IsError: true,
		}, nil
	}

	if lspServer == "gopls" {
		cmd := exec.CommandContext(ctx, "gopls", "rename",
			fmt.Sprintf("%s:%d:%d", file, int(line), int(column)), newName)
		cmd.Dir = tc.WorkDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return &ToolResult{
				Output:  fmt.Sprintf("gopls error: %s\nOutput: %s", err, string(output)),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Output:  string(output),
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Output:  fmt.Sprintf("rename not yet implemented for %s", lspServer),
		IsError: true,
	}, nil
}

// getServerInfo returns information about available language servers
func getServerInfo(workDir string) (*ToolResult, error) {
	servers := []string{
		"gopls",
		"typescript-language-server",
		"pylsp",
		"rust-analyzer",
		"clangd",
		"jdtls",
		"solargraph",
		"phpactor",
		"lua-language-server",
	}

	var available []string
	var unavailable []string

	for _, server := range servers {
		if isCommandAvailable(server) {
			// Try to get version
			cmd := exec.Command(server, "--version")
			output, err := cmd.CombinedOutput()
			version := strings.TrimSpace(string(output))
			if err != nil {
				version = "unknown version"
			}
			available = append(available, fmt.Sprintf("✓ %s (%s)", server, version))
		} else {
			unavailable = append(unavailable, fmt.Sprintf("✗ %s (not installed)", server))
		}
	}

	result := "# Language Server Status\n\n"
	result += "## Available:\n"
	for _, s := range available {
		result += fmt.Sprintf("%s\n", s)
	}
	result += "\n## Unavailable:\n"
	for _, s := range unavailable {
		result += fmt.Sprintf("%s\n", s)
	}

	// Add installation instructions
	result += "\n## Installation:\n"
	result += "- gopls: `go install golang.org/x/tools/gopls@latest`\n"
	result += "- typescript-language-server: `npm install -g typescript-language-server typescript`\n"
	result += "- pylsp: `pip install python-lsp-server`\n"
	result += "- rust-analyzer: `rustup component add rust-analyzer`\n"

	return &ToolResult{
		Output:  result,
		IsError: false,
	}, nil
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// LSPPosition represents a position in a file
type LSPPosition struct {
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	File      string `json:"file"`
}

// LSPLocation represents a location in code
type LSPLocation struct {
	File  string `json:"file"`
	Range struct {
		Start LSPPosition `json:"start"`
		End   LSPPosition `json:"end"`
	} `json:"range"`
}

// FormatLSPResult formats LSP results as JSON or markdown
func FormatLSPResult(data interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

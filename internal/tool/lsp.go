package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
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
					"type":        "string",
					"description": "LSP operation to perform",
					"enum": []string{
						"definition",
						"references",
						"hover",
						"symbols",
						"workspace_symbols",
						"diagnostics",
						"format",
						"rename",
						"server_info",
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

// ─── Language server detection ───────────────────────────────────────────────

// lspServerForFile returns the LSP server command and its args for a given file.
// Returns ("", nil) if no server is installed.
type lspServerDef struct {
	binary  string
	args    []string
	version string // --version flag if different
}

func detectLSPServerForFile(file string) *lspServerDef {
	ext := strings.ToLower(filepath.Ext(file))

	candidates := map[string]*lspServerDef{
		".go":   {binary: "gopls"},
		".ts":   {binary: "typescript-language-server", args: []string{"--stdio"}},
		".tsx":  {binary: "typescript-language-server", args: []string{"--stdio"}},
		".js":   {binary: "typescript-language-server", args: []string{"--stdio"}},
		".jsx":  {binary: "typescript-language-server", args: []string{"--stdio"}},
		".py":   {binary: "pylsp"},
		".rs":   {binary: "rust-analyzer"},
		".c":    {binary: "clangd"},
		".cpp":  {binary: "clangd"},
		".cc":   {binary: "clangd"},
		".h":    {binary: "clangd"},
		".hpp":  {binary: "clangd"},
		".java": {binary: "jdtls"},
		".rb":   {binary: "solargraph", args: []string{"stdio"}},
		".php":  {binary: "phpactor", args: []string{"language-server"}},
		".lua":  {binary: "lua-language-server"},
	}

	def, ok := candidates[ext]
	if !ok {
		return nil
	}
	if !isCommandAvailable(def.binary) {
		return nil
	}
	return def
}

func executeLSP(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	operation, ok := input["operation"].(string)
	if !ok {
		return &ToolResult{Output: "operation parameter is required", IsError: true}, nil
	}

	if operation == "server_info" {
		return getServerInfo(tc.WorkDir)
	}

	file, _ := input["file"].(string)
	if file == "" && operation != "workspace_symbols" {
		return &ToolResult{Output: "file parameter is required for this operation", IsError: true}, nil
	}
	if file != "" && !filepath.IsAbs(file) {
		file = filepath.Join(tc.WorkDir, file)
	}

	srv := detectLSPServerForFile(file)
	if srv == nil && file != "" {
		return &ToolResult{
			Output:  fmt.Sprintf("no language server found for file %s.\nRun 'LSP {\"operation\":\"server_info\"}' to see installation instructions.", file),
			IsError: true,
		}, nil
	}

	switch operation {
	case "definition":
		return executeDefinition(ctx, tc, srv, file, input)
	case "references":
		return executeReferences(ctx, tc, srv, file, input)
	case "hover":
		return executeHover(ctx, tc, srv, file, input)
	case "symbols":
		return executeSymbols(ctx, tc, srv, file)
	case "workspace_symbols":
		return executeWorkspaceSymbols(ctx, tc, input)
	case "diagnostics":
		return executeDiagnostics(ctx, tc, srv, file)
	case "format":
		return executeFormat(ctx, tc, srv, file)
	case "rename":
		return executeRename(ctx, tc, srv, file, input)
	default:
		return &ToolResult{Output: fmt.Sprintf("unknown LSP operation: %s", operation), IsError: true}, nil
	}
}

// ─── gopls CLI helpers (fast, no stdio LSP dance needed) ─────────────────────

func goplsRun(ctx context.Context, workDir string, args ...string) (*ToolResult, error) {
	cmd := exec.CommandContext(ctx, "gopls", args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil && output == "" {
		return &ToolResult{Output: fmt.Sprintf("gopls error: %v", err), IsError: true}, nil
	}
	return &ToolResult{Output: output}, nil
}

func executeDefinition(ctx context.Context, tc *ToolContext, srv *lspServerDef, file string, input map[string]interface{}) (*ToolResult, error) {
	line, _ := input["line"].(float64)
	col, _ := input["column"].(float64)
	if line == 0 || col == 0 {
		return &ToolResult{Output: "line and column are required for definition", IsError: true}, nil
	}

	if srv.binary == "gopls" {
		return goplsRun(ctx, tc.WorkDir, "definition", fmt.Sprintf("%s:%d:%d", file, int(line), int(col)))
	}
	return lspStdioRequest(ctx, tc, srv, file, "textDocument/definition", lspPosition(file, line, col))
}

func executeReferences(ctx context.Context, tc *ToolContext, srv *lspServerDef, file string, input map[string]interface{}) (*ToolResult, error) {
	line, _ := input["line"].(float64)
	col, _ := input["column"].(float64)
	if line == 0 || col == 0 {
		return &ToolResult{Output: "line and column are required for references", IsError: true}, nil
	}

	if srv.binary == "gopls" {
		return goplsRun(ctx, tc.WorkDir, "references", fmt.Sprintf("%s:%d:%d", file, int(line), int(col)))
	}
	params := lspPosition(file, line, col)
	params["context"] = map[string]interface{}{"includeDeclaration": true}
	return lspStdioRequest(ctx, tc, srv, file, "textDocument/references", params)
}

func executeHover(ctx context.Context, tc *ToolContext, srv *lspServerDef, file string, input map[string]interface{}) (*ToolResult, error) {
	line, _ := input["line"].(float64)
	col, _ := input["column"].(float64)
	if line == 0 || col == 0 {
		return &ToolResult{Output: "line and column are required for hover", IsError: true}, nil
	}

	if srv.binary == "gopls" {
		return goplsRun(ctx, tc.WorkDir, "hover", fmt.Sprintf("%s:%d:%d", file, int(line), int(col)))
	}
	return lspStdioRequest(ctx, tc, srv, file, "textDocument/hover", lspPosition(file, line, col))
}

func executeSymbols(ctx context.Context, tc *ToolContext, srv *lspServerDef, file string) (*ToolResult, error) {
	if srv.binary == "gopls" {
		return goplsRun(ctx, tc.WorkDir, "symbols", file)
	}
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": fileURI(file)},
	}
	return lspStdioRequest(ctx, tc, srv, file, "textDocument/documentSymbol", params)
}

func executeWorkspaceSymbols(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	query, _ := input["query"].(string)
	if query == "" {
		return &ToolResult{Output: "query parameter is required for workspace_symbols", IsError: true}, nil
	}
	if isCommandAvailable("gopls") {
		return goplsRun(ctx, tc.WorkDir, "workspace_symbol", query)
	}
	return &ToolResult{Output: "no language server available for workspace symbols", IsError: true}, nil
}

func executeDiagnostics(ctx context.Context, tc *ToolContext, srv *lspServerDef, file string) (*ToolResult, error) {
	if srv.binary == "gopls" {
		return goplsRun(ctx, tc.WorkDir, "check", file)
	}
	// For other servers: use the generic stdio protocol to open the file and wait for diagnostics
	return lspStdioDiagnostics(ctx, tc, srv, file)
}

func executeFormat(ctx context.Context, tc *ToolContext, srv *lspServerDef, file string) (*ToolResult, error) {
	if srv.binary == "gopls" {
		return goplsRun(ctx, tc.WorkDir, "format", file)
	}
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": fileURI(file)},
		"options":      map[string]interface{}{"tabSize": 4, "insertSpaces": true},
	}
	return lspStdioRequest(ctx, tc, srv, file, "textDocument/formatting", params)
}

func executeRename(ctx context.Context, tc *ToolContext, srv *lspServerDef, file string, input map[string]interface{}) (*ToolResult, error) {
	line, _ := input["line"].(float64)
	col, _ := input["column"].(float64)
	newName, _ := input["new_name"].(string)
	if line == 0 || col == 0 || newName == "" {
		return &ToolResult{Output: "line, column, and new_name are required for rename", IsError: true}, nil
	}

	if srv.binary == "gopls" {
		return goplsRun(ctx, tc.WorkDir, "rename",
			fmt.Sprintf("%s:%d:%d", file, int(line), int(col)), newName)
	}

	params := lspPosition(file, line, col)
	params["newName"] = newName
	return lspStdioRequest(ctx, tc, srv, file, "textDocument/rename", params)
}

// ─── Generic stdio LSP client ─────────────────────────────────────────────────

var lspReqID int64

func nextLSPID() int {
	return int(atomic.AddInt64(&lspReqID, 1))
}

func fileURI(path string) string {
	if !filepath.IsAbs(path) {
		abs, _ := filepath.Abs(path)
		path = abs
	}
	return "file://" + path
}

func lspPosition(file string, line, col float64) map[string]interface{} {
	return map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": fileURI(file)},
		"position": map[string]interface{}{
			"line":      int(line) - 1, // LSP is 0-indexed
			"character": int(col) - 1,
		},
	}
}

// lspMessage wraps a JSON-RPC 2.0 message
type lspMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int        `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// writeLSPMessage writes a Language Server Protocol message to a writer.
func writeLSPMessage(w io.Writer, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// readLSPMessage reads one Language Server Protocol message from a buffered reader.
func readLSPMessage(r *bufio.Reader) ([]byte, error) {
	contentLen := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // end of headers
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			s := strings.TrimPrefix(line, "Content-Length: ")
			contentLen, _ = strconv.Atoi(strings.TrimSpace(s))
		}
	}
	if contentLen == 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	buf := make([]byte, contentLen)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

// lspStdioRequest starts an LSP server, performs initialize/open handshake,
// sends one request, reads the response, and shuts down the server.
func lspStdioRequest(ctx context.Context, tc *ToolContext, srv *lspServerDef, file, method string, params interface{}) (*ToolResult, error) {
	args := srv.args
	cmd := exec.CommandContext(ctx, srv.binary, args...)
	cmd.Dir = tc.WorkDir
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("stdin pipe: %v", err), IsError: true}, nil
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("stdout pipe: %v", err), IsError: true}, nil
	}

	if err := cmd.Start(); err != nil {
		return &ToolResult{Output: fmt.Sprintf("failed to start %s: %v", srv.binary, err), IsError: true}, nil
	}
	defer func() {
		// Graceful shutdown
		id := nextLSPID()
		_ = writeLSPMessage(stdin, lspMessage{JSONRPC: "2.0", ID: &id, Method: "shutdown"})
		_ = writeLSPMessage(stdin, lspMessage{JSONRPC: "2.0", Method: "exit"})
		stdin.Close()
		cmd.Wait() //nolint:errcheck
	}()

	reader := bufio.NewReaderSize(stdout, 1<<20)

	// 1. initialize
	initID := nextLSPID()
	initReq := lspMessage{
		JSONRPC: "2.0",
		ID:      &initID,
		Method:  "initialize",
		Params: map[string]interface{}{
			"processId":    os.Getpid(),
			"rootUri":      fileURI(tc.WorkDir),
			"capabilities": map[string]interface{}{},
		},
	}
	if err := writeLSPMessage(stdin, initReq); err != nil {
		return &ToolResult{Output: fmt.Sprintf("initialize write: %v", err), IsError: true}, nil
	}
	// Read initialize result
	if _, err := readLSPMessage(reader); err != nil {
		return &ToolResult{Output: fmt.Sprintf("initialize read: %v", err), IsError: true}, nil
	}
	// initialized notification
	_ = writeLSPMessage(stdin, lspMessage{JSONRPC: "2.0", Method: "initialized", Params: map[string]interface{}{}})

	// 2. textDocument/didOpen
	content, err := os.ReadFile(file)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("read file: %v", err), IsError: true}, nil
	}
	lang := inferLanguage(file)
	_ = writeLSPMessage(stdin, lspMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params: map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":        fileURI(file),
				"languageId": lang,
				"version":    1,
				"text":       string(content),
			},
		},
	})

	// 3. Send the actual request
	reqID := nextLSPID()
	req := lspMessage{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  method,
		Params:  params,
	}
	if err := writeLSPMessage(stdin, req); err != nil {
		return &ToolResult{Output: fmt.Sprintf("request write: %v", err), IsError: true}, nil
	}

	// 4. Read responses until we get the one with our ID
	for {
		raw, err := readLSPMessage(reader)
		if err != nil {
			return &ToolResult{Output: fmt.Sprintf("response read: %v", err), IsError: true}, nil
		}

		var resp struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			continue // might be a notification
		}
		if resp.ID == nil || *resp.ID != reqID {
			continue // notification or different response
		}

		if resp.Error != nil {
			return &ToolResult{Output: fmt.Sprintf("LSP error %d: %s", resp.Error.Code, resp.Error.Message), IsError: true}, nil
		}

		// Pretty-print the result
		var pretty interface{}
		json.Unmarshal(resp.Result, &pretty)
		out, _ := json.MarshalIndent(pretty, "", "  ")
		return &ToolResult{Output: string(out)}, nil
	}
}

// lspStdioDiagnostics opens a file and captures publishDiagnostics notifications.
func lspStdioDiagnostics(ctx context.Context, tc *ToolContext, srv *lspServerDef, file string) (*ToolResult, error) {
	args := srv.args
	cmd := exec.CommandContext(ctx, srv.binary, args...)
	cmd.Dir = tc.WorkDir
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("stdin pipe: %v", err), IsError: true}, nil
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("stdout pipe: %v", err), IsError: true}, nil
	}

	if err := cmd.Start(); err != nil {
		return &ToolResult{Output: fmt.Sprintf("failed to start %s: %v", srv.binary, err), IsError: true}, nil
	}
	defer func() {
		id := nextLSPID()
		_ = writeLSPMessage(stdin, lspMessage{JSONRPC: "2.0", ID: &id, Method: "shutdown"})
		_ = writeLSPMessage(stdin, lspMessage{JSONRPC: "2.0", Method: "exit"})
		stdin.Close()
		cmd.Wait() //nolint:errcheck
	}()

	reader := bufio.NewReaderSize(stdout, 1<<20)

	initID := nextLSPID()
	_ = writeLSPMessage(stdin, lspMessage{
		JSONRPC: "2.0",
		ID:      &initID,
		Method:  "initialize",
		Params: map[string]interface{}{
			"processId": os.Getpid(),
			"rootUri":   fileURI(tc.WorkDir),
			"capabilities": map[string]interface{}{
				"textDocument": map[string]interface{}{
					"publishDiagnostics": map[string]interface{}{},
				},
			},
		},
	})
	if _, err := readLSPMessage(reader); err != nil {
		return &ToolResult{Output: fmt.Sprintf("initialize read: %v", err), IsError: true}, nil
	}
	_ = writeLSPMessage(stdin, lspMessage{JSONRPC: "2.0", Method: "initialized", Params: map[string]interface{}{}})

	content, _ := os.ReadFile(file)
	lang := inferLanguage(file)
	_ = writeLSPMessage(stdin, lspMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params: map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":        fileURI(file),
				"languageId": lang,
				"version":    1,
				"text":       string(content),
			},
		},
	})

	// Wait up to 5 seconds for a publishDiagnostics notification for our file
	diagCh := make(chan string, 1)
	go func() {
		for {
			raw, err := readLSPMessage(reader)
			if err != nil {
				diagCh <- fmt.Sprintf("read error: %v", err)
				return
			}
			var notif struct {
				Method string `json:"method"`
				Params struct {
					URI         string `json:"uri"`
					Diagnostics []struct {
						Range struct {
							Start struct {
								Line      int `json:"line"`
								Character int `json:"character"`
							} `json:"start"`
						} `json:"range"`
						Severity int    `json:"severity"`
						Message  string `json:"message"`
					} `json:"diagnostics"`
				} `json:"params"`
			}
			if err := json.Unmarshal(raw, &notif); err != nil {
				continue
			}
			if notif.Method != "textDocument/publishDiagnostics" {
				continue
			}
			if !strings.HasSuffix(notif.Params.URI, filepath.Base(file)) {
				continue
			}
			if len(notif.Params.Diagnostics) == 0 {
				diagCh <- "No diagnostics (file is clean)"
				return
			}
			var sb strings.Builder
			severityName := func(s int) string {
				switch s {
				case 1:
					return "error"
				case 2:
					return "warning"
				case 3:
					return "info"
				case 4:
					return "hint"
				default:
					return "unknown"
				}
			}
			sb.WriteString(fmt.Sprintf("# Diagnostics for %s\n\n", file))
			for _, d := range notif.Params.Diagnostics {
				sb.WriteString(fmt.Sprintf("- Line %d: [%s] %s\n",
					d.Range.Start.Line+1, severityName(d.Severity), d.Message))
			}
			diagCh <- sb.String()
			return
		}
	}()

	select {
	case result := <-diagCh:
		return &ToolResult{Output: result}, nil
	case <-ctx.Done():
		return &ToolResult{Output: "diagnostics timed out", IsError: true}, nil
	}
}

// ─── server_info ─────────────────────────────────────────────────────────────

func getServerInfo(workDir string) (*ToolResult, error) {
	type serverDef struct {
		name    string
		install string
	}
	servers := []serverDef{
		{"gopls", "go install golang.org/x/tools/gopls@latest"},
		{"typescript-language-server", "npm install -g typescript-language-server typescript"},
		{"pylsp", "pip install python-lsp-server"},
		{"rust-analyzer", "rustup component add rust-analyzer"},
		{"clangd", "install clangd from https://clangd.llvm.org/installation"},
		{"jdtls", "install Eclipse JDT LS from https://github.com/eclipse-jdtls/eclipse.jdt.ls"},
		{"solargraph", "gem install solargraph"},
		{"phpactor", "composer global require phpactor/phpactor"},
		{"lua-language-server", "install from https://github.com/LuaLS/lua-language-server"},
	}

	var available, unavailable []string

	for _, srv := range servers {
		if isCommandAvailable(srv.name) {
			cmd := exec.Command(srv.name, "--version")
			out, _ := cmd.CombinedOutput()
			ver := strings.TrimSpace(string(out))
			if ver == "" {
				ver = "installed"
			}
			available = append(available, fmt.Sprintf("✓ %s (%s)", srv.name, ver))
		} else {
			unavailable = append(unavailable, fmt.Sprintf("✗ %s  →  %s", srv.name, srv.install))
		}
	}

	result := "# Language Server Status\n\n## Available\n"
	if len(available) == 0 {
		result += "  (none)\n"
	}
	for _, s := range available {
		result += "  " + s + "\n"
	}
	result += "\n## Not Installed\n"
	for _, s := range unavailable {
		result += "  " + s + "\n"
	}
	return &ToolResult{Output: result}, nil
}

// isCommandAvailable checks if a command exists in PATH.
func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

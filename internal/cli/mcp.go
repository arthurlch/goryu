package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The MCP server speaks JSON-RPC 2.0 over stdio. See https://modelcontextprotocol.io.

const mcpProtocolVersion = "2024-11-05"

func newMCPCommand() *Command {
	return &Command{
		Name:        "mcp",
		Description: "Run an MCP server (stdio) exposing goryu tools to AI agents",
		Usage:       "goryu mcp",
		Action:      cmdMCP,
	}
}

func cmdMCP(_ *Context) error {
	return runMCPServer(os.Stdin, os.Stdout)
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func runMCPServer(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	encoder := json.NewEncoder(out)

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			handleMCPLine(line, encoder)
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func handleMCPLine(line []byte, encoder *json.Encoder) {
	if len(strings.TrimSpace(string(line))) == 0 {
		return
	}

	var req mcpRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return
	}

	// Notifications (no id) get no response.
	isNotification := len(req.ID) == 0
	result, rpcErr := dispatchMCP(req)
	if isNotification {
		return
	}

	resp := mcpResponse{JSONRPC: "2.0", ID: req.ID}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}
	_ = encoder.Encode(resp)
}

func dispatchMCP(req mcpRequest) (interface{}, *mcpError) {
	switch req.Method {
	case "initialize":
		return map[string]interface{}{
			"protocolVersion": mcpProtocolVersion,
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "goryu", "version": "0.1.0"},
		}, nil
	case "ping":
		return map[string]interface{}{}, nil
	case "tools/list":
		return map[string]interface{}{"tools": mcpTools()}, nil
	case "tools/call":
		return callMCPTool(req.Params)
	default:
		return nil, &mcpError{Code: -32601, Message: "method not found: " + req.Method}
	}
}

func mcpTools() []map[string]interface{} {
	obj := func(props map[string]interface{}, required ...string) map[string]interface{} {
		s := map[string]interface{}{"type": "object", "properties": props}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	str := map[string]interface{}{"type": "string"}

	return []map[string]interface{}{
		{
			"name":        "list_routes",
			"description": "List the HTTP routes discovered in the current goryu project.",
			"inputSchema": obj(map[string]interface{}{}),
		},
		{
			"name":        "scaffold_handler",
			"description": "Generate a handler in the current project. Supports type=ai with kind=chat|rag|agent.",
			"inputSchema": obj(map[string]interface{}{
				"name": str,
				"type": map[string]interface{}{"type": "string", "enum": []string{"basic", "crud", "api", "ai"}},
				"kind": map[string]interface{}{"type": "string", "enum": []string{"chat", "rag", "agent"}},
				"path": str,
			}, "name"),
		},
		{
			"name":        "run_dev_server",
			"description": "Start the project's dev server in the background; returns its PID and log file.",
			"inputSchema": obj(map[string]interface{}{"port": str}),
		},
	}
}

func callMCPTool(params json.RawMessage) (interface{}, *mcpError) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &mcpError{Code: -32602, Message: "invalid params"}
	}

	switch call.Name {
	case "list_routes":
		return toolListRoutes()
	case "scaffold_handler":
		return toolScaffoldHandler(call.Arguments)
	case "run_dev_server":
		return toolRunDevServer(call.Arguments)
	default:
		return nil, &mcpError{Code: -32602, Message: "unknown tool: " + call.Name}
	}
}

func toolListRoutes() (interface{}, *mcpError) {
	routes, err := discoverRoutes()
	if err != nil {
		return mcpText("failed to discover routes: "+err.Error(), true), nil
	}
	if len(routes) == 0 {
		return mcpText("no routes found in the current project", false), nil
	}
	data, _ := json.MarshalIndent(routes, "", "  ")
	return mcpText(string(data), false), nil
}

func toolScaffoldHandler(args json.RawMessage) (interface{}, *mcpError) {
	var a struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Kind string `json:"kind"`
		Path string `json:"path"`
	}
	_ = json.Unmarshal(args, &a)
	if a.Name == "" {
		return mcpText("name is required", true), nil
	}
	if strings.Contains(a.Path, "..") || filepath.IsAbs(a.Path) {
		return mcpText("path must be relative and must not contain '..'", true), nil
	}
	if a.Type == "" {
		a.Type = "ai"
	}
	if a.Kind == "" {
		a.Kind = "chat"
	}
	if a.Path == "" {
		a.Path = "internal/handlers"
	}

	cmdArgs := []string{a.Name, "--type=" + a.Type, "--path=" + a.Path}
	if a.Type == "ai" {
		cmdArgs = append(cmdArgs, "--kind="+a.Kind)
	}

	output, err := captureStdout(func() error { return runGenerateHandler(cmdArgs) })
	if err != nil {
		return mcpText(output+"\nerror: "+err.Error(), true), nil
	}
	return mcpText(output, false), nil
}

func toolRunDevServer(args json.RawMessage) (interface{}, *mcpError) {
	var a struct {
		Port string `json:"port"`
	}
	_ = json.Unmarshal(args, &a)
	if a.Port == "" {
		a.Port = "3000"
	}

	exe, err := os.Executable()
	if err != nil {
		return mcpText("cannot locate goryu binary: "+err.Error(), true), nil
	}

	logFile, err := os.Create("goryu-dev.log")
	if err != nil {
		return mcpText("cannot create log file: "+err.Error(), true), nil
	}

	cmd := exec.Command(exe, "dev", "--port="+a.Port, "--hot-reload=false")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	err = cmd.Start()
	_ = logFile.Close() // the child holds its own dup of the fd
	if err != nil {
		return mcpText("failed to start dev server: "+err.Error(), true), nil
	}

	return mcpText(fmt.Sprintf("dev server started (pid %d) on port %s, logs in goryu-dev.log", cmd.Process.Pid, a.Port), false), nil
}

func mcpText(text string, isError bool) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{{"type": "text", "text": text}},
		"isError": isError,
	}
}

// captureStdout redirects os.Stdout while fn runs so generator output (which is
// printed for humans) doesn't corrupt the JSON-RPC stream.
func captureStdout(fn func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", fn()
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fnErr := fn()
	_ = w.Close()
	os.Stdout = old
	captured := <-done
	_ = r.Close()
	return captured, fnErr
}

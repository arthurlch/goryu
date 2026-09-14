package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func runMCP(t *testing.T, requests ...string) []mcpResponse {
	t.Helper()
	in := strings.NewReader(strings.Join(requests, "\n") + "\n")
	var out bytes.Buffer
	if err := runMCPServer(in, &out); err != nil {
		t.Fatalf("runMCPServer: %v", err)
	}

	var responses []mcpResponse
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var r mcpResponse
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("bad response %q: %v", line, err)
		}
		responses = append(responses, r)
	}
	return responses
}

func TestMCPInitialize(t *testing.T) {
	resps := runMCP(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	result := resps[0].Result.(map[string]interface{})
	if result["protocolVersion"] != mcpProtocolVersion {
		t.Fatalf("protocol version wrong: %v", result["protocolVersion"])
	}
}

func TestMCPNotificationHasNoResponse(t *testing.T) {
	resps := runMCP(t,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
	)
	if len(resps) != 1 {
		t.Fatalf("notification must not produce a response; got %d responses", len(resps))
	}
}

func TestMCPToolsList(t *testing.T) {
	resps := runMCP(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	result := resps[0].Result.(map[string]interface{})
	tools := result["tools"].([]interface{})

	names := map[string]bool{}
	for _, tRaw := range tools {
		names[tRaw.(map[string]interface{})["name"].(string)] = true
	}
	for _, want := range []string{"list_routes", "scaffold_handler", "run_dev_server"} {
		if !names[want] {
			t.Fatalf("missing tool %q in %v", want, names)
		}
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	resps := runMCP(t, `{"jsonrpc":"2.0","id":9,"method":"does/not/exist"}`)
	if resps[0].Error == nil || resps[0].Error.Code != -32601 {
		t.Fatalf("expected method-not-found error, got %+v", resps[0].Error)
	}
}

func TestMCPScaffoldToolRequiresName(t *testing.T) {
	resps := runMCP(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scaffold_handler","arguments":{}}}`)
	result := resps[0].Result.(map[string]interface{})
	if result["isError"] != true {
		t.Fatalf("expected isError=true for missing name, got %v", result)
	}
}

func TestMCPScaffoldRejectsPathTraversal(t *testing.T) {
	resps := runMCP(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scaffold_handler","arguments":{"name":"../../evil","type":"basic"}}}`)
	result := resps[0].Result.(map[string]interface{})
	if result["isError"] != true {
		t.Fatalf("path-traversal name must be rejected, got %v", result)
	}
}

func TestMCPScaffoldRejectsPathEscape(t *testing.T) {
	resps := runMCP(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scaffold_handler","arguments":{"name":"ok","path":"../../etc"}}}`)
	result := resps[0].Result.(map[string]interface{})
	if result["isError"] != true {
		t.Fatalf("path escape via --path must be rejected, got %v", result)
	}
}

func TestIsSafeName(t *testing.T) {
	safe := []string{"user", "user_profile", "order-item", "Chat1"}
	unsafe := []string{"", "../evil", "a/b", `a\b`, "..", "a.b", "a b"}
	for _, n := range safe {
		if !isSafeName(n) {
			t.Errorf("%q should be safe", n)
		}
	}
	for _, n := range unsafe {
		if isSafeName(n) {
			t.Errorf("%q should be rejected", n)
		}
	}
}

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func runServer(t *testing.T, input string) []map[string]any {
	t.Helper()
	s := NewServer("test", "0.0.1", io.Discard)
	s.AddTool(Tool{
		Name:        "echo",
		Description: "echoes",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			text := StringArg(args, "text")
			if text == "fail" {
				return "", errors.New("boom")
			}
			return "echo: " + text, nil
		},
	})
	var out strings.Builder
	if err := s.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var responses []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(out.String()))
	for scanner.Scan() {
		var resp map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("response not json: %s", scanner.Text())
		}
		responses = append(responses, resp)
	}
	return responses
}

func TestHandshakeListAndCall(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"claude","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{"text":"fail"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"nope"}}`,
		`{"jsonrpc":"2.0","id":"str","method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":6,"method":"ping"}`,
	}, "\n") + "\n"

	responses := runServer(t, input)
	if len(responses) != 7 {
		t.Fatalf("got %d responses, want 7 (notification must not be answered): %v", len(responses), responses)
	}

	init := responses[0]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion = %v", init["protocolVersion"])
	}
	if _, ok := init["capabilities"].(map[string]any)["tools"]; !ok {
		t.Errorf("tools capability missing: %v", init)
	}

	tools := responses[1]["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["name"] != "echo" {
		t.Errorf("tools/list = %v", tools)
	}

	call := responses[2]["result"].(map[string]any)
	content := call["content"].([]any)[0].(map[string]any)
	if content["text"] != "echo: hi" || call["isError"] != false {
		t.Errorf("tools/call = %v", call)
	}

	failed := responses[3]["result"].(map[string]any)
	if failed["isError"] != true || failed["content"].([]any)[0].(map[string]any)["text"] != "boom" {
		t.Errorf("tool error should be isError result: %v", failed)
	}

	if responses[4]["error"] == nil {
		t.Errorf("unknown tool should be a JSON-RPC error: %v", responses[4])
	}
	if responses[5]["error"].(map[string]any)["code"].(float64) != codeMethodNotFound || responses[5]["id"] != "str" {
		t.Errorf("unknown method: %v", responses[5])
	}
	if _, ok := responses[6]["result"]; !ok {
		t.Errorf("ping: %v", responses[6])
	}
}

func TestUnknownProtocolVersionFallsBackToLatest(t *testing.T) {
	responses := runServer(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`+"\n")
	got := responses[0]["result"].(map[string]any)["protocolVersion"]
	if got != latestProtocolVersion {
		t.Errorf("protocolVersion = %v, want %s", got, latestProtocolVersion)
	}
}

func TestParseError(t *testing.T) {
	responses := runServer(t, "{not json\n")
	if responses[0]["error"].(map[string]any)["code"].(float64) != codeParseError {
		t.Errorf("parse error expected: %v", responses[0])
	}
}

func TestStringSliceArg(t *testing.T) {
	args := map[string]any{"a": "one", "b": []any{"x", "y", 3}, "c": 7}
	if got := StringSliceArg(args, "a"); len(got) != 1 || got[0] != "one" {
		t.Errorf("string: %v", got)
	}
	if got := StringSliceArg(args, "b"); len(got) != 2 {
		t.Errorf("list: %v", got)
	}
	if got := StringSliceArg(args, "c"); got != nil {
		t.Errorf("other: %v", got)
	}
}

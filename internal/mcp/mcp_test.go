package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func serve(t *testing.T, s *Server, requests ...string) map[string]map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := s.Serve(context.Background(), strings.NewReader(strings.Join(requests, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	// Responses come in any order; key them by id.
	byID := map[string]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var resp map[string]any
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("bad response %q: %v", line, err)
		}
		if resp["jsonrpc"] != "2.0" {
			t.Fatalf("response without jsonrpc 2.0: %q", line)
		}
		id, _ := json.Marshal(resp["id"])
		byID[string(id)] = resp
	}
	return byID
}

func echoServer() *Server {
	return &Server{
		Name: "test", Version: "1.0", Instructions: "Echoes.",
		Tools: []Tool{{
			Name:        "echo",
			Description: "Echoes its text.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`),
			Call: func(_ context.Context, args json.RawMessage) Result {
				var in struct{ Text string }
				_ = json.Unmarshal(args, &in)
				return Result{Text: in.Text, Structured: map[string]string{"text": in.Text}, IsError: in.Text == "fail"}
			},
		}},
	}
}

func TestHandshakeAndTools(t *testing.T) {
	got := serve(t, echoServer(),
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":"three","method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{"text":"fail"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"ping"}`,
	)
	if len(got) != 5 {
		t.Fatalf("%d responses, want 5 (the notification gets none): %v", len(got), got)
	}
	init := got["1"]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-03-26" || init["instructions"] != "Echoes." {
		t.Errorf("initialize = %v", init)
	}
	tools := got["2"]["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["name"] != "echo" {
		t.Errorf("tools/list = %v", tools)
	}
	call := got[`"three"`]["result"].(map[string]any)
	text := call["content"].([]any)[0].(map[string]any)["text"]
	if text != "hi" || call["isError"] != false || call["structuredContent"].(map[string]any)["text"] != "hi" {
		t.Errorf("tools/call = %v", call)
	}
	if got["4"]["result"].(map[string]any)["isError"] != true {
		t.Errorf("a failing call is not marked: %v", got["4"])
	}
	if _, ok := got["5"]["result"]; !ok {
		t.Errorf("ping = %v", got["5"])
	}
}

func TestUnknownVersionGetsTheNewest(t *testing.T) {
	got := serve(t, echoServer(), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`)
	if v := got["1"]["result"].(map[string]any)["protocolVersion"]; v != ProtocolVersions[0] {
		t.Errorf("protocolVersion = %v", v)
	}
}

func TestErrors(t *testing.T) {
	got := serve(t, echoServer(),
		`not json`,
		`{"jsonrpc":"2.0","id":1,"method":"bogus"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"nope"}}`,
		`{"jsonrpc":"1.0","id":3,"method":"ping"}`,
	)
	codes := map[string]float64{"null": -32700, "1": -32601, "2": -32602, "3": -32600}
	for id, want := range codes {
		resp, ok := got[id]
		if !ok {
			t.Errorf("no response for id %s", id)
			continue
		}
		if code := resp["error"].(map[string]any)["code"]; code != want {
			t.Errorf("id %s: code %v, want %v", id, code, want)
		}
	}
}

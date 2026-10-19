// Package mcp is a small Model Context Protocol server: JSON-RPC 2.0 over a
// stream, one message per line, as MCP's stdio transport has it. It answers
// initialize, ping, tools/list and tools/call, which is all a server that
// offers tools needs; a request for anything else gets "method not found".
// Requests are handled concurrently, so a long tool call does not hold up a
// ping, and responses are written whole, one at a time.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"sync"
)

// ProtocolVersions are the MCP revisions the server speaks, newest first.
var ProtocolVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// Tool is one tool the server offers.
type Tool struct {
	Name        string
	Title       string
	Description string
	// InputSchema and OutputSchema are JSON Schemas of the arguments and of
	// the result's structured content.
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
	// Call runs the tool with its arguments as sent.
	Call func(ctx context.Context, arguments json.RawMessage) Result
}

// Result is what a tool call returns: text for the model, the same as JSON
// for a program, and whether the call failed.
type Result struct {
	Text       string
	Structured any
	IsError    bool
}

// Server answers MCP requests.
type Server struct {
	Name, Version string
	// Instructions tell the client what the server is for.
	Instructions string
	Tools        []Tool
}

// The JSON-RPC error codes the server uses.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Serve reads requests from r and writes responses to w until r ends or ctx
// is done, then waits for the calls still running.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	send := func(resp response) {
		resp.JSONRPC = "2.0"
		data, err := json.Marshal(resp)
		if err != nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		_, _ = w.Write(append(data, '\n'))
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64<<10), 16<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			send(response{ID: json.RawMessage("null"), Error: &rpcError{codeParse, "parse error: " + err.Error()}})
			continue
		}
		if req.JSONRPC != "2.0" || req.Method == "" {
			if len(req.ID) > 0 {
				send(response{ID: req.ID, Error: &rpcError{codeInvalidRequest, "not a JSON-RPC 2.0 request"}})
			}
			continue
		}
		if len(req.ID) == 0 {
			continue // a notification: nothing to answer
		}
		wg.Go(func() {
			result, rerr := s.handle(ctx, req)
			send(response{ID: req.ID, Result: result, Error: rerr})
		})
	}
	wg.Wait()
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return ctx.Err()
}

// handle answers one request.
func (s *Server) handle(ctx context.Context, req request) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := ProtocolVersions[0]
		if slices.Contains(ProtocolVersions, p.ProtocolVersion) {
			version = p.ProtocolVersion
		}
		result := map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": s.Name, "version": s.Version},
		}
		if s.Instructions != "" {
			result["instructions"] = s.Instructions
		}
		return result, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		tools := make([]map[string]any, 0, len(s.Tools))
		for _, t := range s.Tools {
			entry := map[string]any{"name": t.Name, "description": t.Description, "inputSchema": t.InputSchema}
			if t.Title != "" {
				entry["title"] = t.Title
			}
			if len(t.OutputSchema) > 0 {
				entry["outputSchema"] = t.OutputSchema
			}
			tools = append(tools, entry)
		}
		return map[string]any{"tools": tools}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{codeInvalidParams, "invalid params: " + err.Error()}
		}
		for _, t := range s.Tools {
			if t.Name == p.Name {
				r := t.Call(ctx, p.Arguments)
				result := map[string]any{
					"content": []map[string]any{{"type": "text", "text": r.Text}},
					"isError": r.IsError,
				}
				if r.Structured != nil {
					result["structuredContent"] = r.Structured
				}
				return result, nil
			}
		}
		return nil, &rpcError{codeInvalidParams, "unknown tool: " + p.Name}
	}
	return nil, &rpcError{codeMethodNotFound, "method not found: " + req.Method}
}

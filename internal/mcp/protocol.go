package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var reContentLen = regexp.MustCompile(`(?i)^Content-Length:\s*(\d+)\s*$`)

// ServeStdio handles MCP JSON-RPC frames from stdin/stdout-style readers and writers.
func (s *Server) ServeStdio(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		body, err := readFrame(reader)
		if err != nil {
			return err
		}

		resp, shouldReply := s.ProcessRequestBody(body)
		if shouldReply {
			s.writeResponse(out, resp)
		}
	}
}

// ProcessRequestBody decodes one JSON-RPC request body and returns a response when MCP requires one.
func (s *Server) ProcessRequestBody(body []byte) (response, bool) {
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return response{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32700, Message: "parse error"},
		}, true
	}

	if req.Method == "" {
		if req.ID != nil {
			return response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32600, Message: "invalid request"},
			}, true
		}
		return response{}, false
	}

	resp, ok := s.handleRequest(req)
	if ok && req.ID != nil {
		return resp, true
	}
	return response{}, false
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}
		if match := reContentLen.FindStringSubmatch(trimmed); len(match) == 2 {
			var n int
			fmt.Sscanf(match[1], "%d", &n)
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (s *Server) writeResponse(out io.Writer, resp response) {
	payload, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal response error: %v\n", err)
		return
	}
	s.outMu.Lock()
	defer s.outMu.Unlock()
	fmt.Fprintf(out, "Content-Length: %d\r\n\r\n", len(payload))
	_, _ = out.Write(payload)
}

func (s *Server) handleRequest(req request) (response, bool) {
	resp := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{"subscribe": false, "listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "multicard-docs-mcp-go",
				"version": ServerVersion,
			},
		}
		return resp, true
	case "notifications/initialized":
		return response{}, false
	case "ping":
		resp.Result = map[string]any{}
		return resp, true
	case "tools/list":
		resp.Result = map[string]any{"tools": s.tools()}
		return resp, true
	case "tools/call":
		result, err := s.callTool(req.Params)
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			return resp, true
		}
		resp.Result = result
		return resp, true
	case "resources/list":
		resp.Result = map[string]any{"resources": s.resources()}
		return resp, true
	case "resources/read":
		result, err := s.readResource(req.Params)
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			return resp, true
		}
		resp.Result = result
		return resp, true
	case "prompts/list":
		resp.Result = map[string]any{"prompts": []any{}}
		return resp, true
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found"}
		return resp, true
	}
}

package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func postMCP(t *testing.T, server *Server, body map[string]any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(raw))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	server.handleHTTPMCP(rec, req)
	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) response {
	t.Helper()
	var resp response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body = %s", err, rec.Body.String())
	}
	return resp
}

func TestHandleHTTPMCPLegacyRequestUnaffectedByNewHeaderRules(t *testing.T) {
	server := testServer(t)
	rec := postMCP(t, server, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
}

func TestHandleHTTPMCPModernRequestWithCorrectHeaders(t *testing.T) {
	server := testServer(t)
	rec := postMCP(t, server, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": ModernProtocolVersion},
		},
	}, map[string]string{
		"MCP-Protocol-Version": ModernProtocolVersion,
		"Mcp-Method":           "tools/list",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
}

func TestHandleHTTPMCPModernRequestMissingProtocolVersionHeaderRejected(t *testing.T) {
	server := testServer(t)
	rec := postMCP(t, server, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": ModernProtocolVersion},
		},
	}, map[string]string{
		"Mcp-Method": "tools/list",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != errCodeHeaderMismatch {
		t.Fatalf("expected HeaderMismatch error, got %+v", resp.Error)
	}
}

func TestHandleHTTPMCPModernRequestWrongMcpMethodHeaderRejected(t *testing.T) {
	server := testServer(t)
	rec := postMCP(t, server, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": ModernProtocolVersion},
		},
	}, map[string]string{
		"MCP-Protocol-Version": ModernProtocolVersion,
		"Mcp-Method":           "resources/list",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != errCodeHeaderMismatch {
		t.Fatalf("expected HeaderMismatch error, got %+v", resp.Error)
	}
}

func TestHandleHTTPMCPModernToolsCallRequiresMatchingMcpName(t *testing.T) {
	server := testServer(t)
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "search_multicard_docs",
			"arguments": map[string]any{"query": "invoice"},
			"_meta":     map[string]any{"io.modelcontextprotocol/protocolVersion": ModernProtocolVersion},
		},
	}

	// Correct Mcp-Name: succeeds.
	rec := postMCP(t, server, body, map[string]string{
		"MCP-Protocol-Version": ModernProtocolVersion,
		"Mcp-Method":           "tools/call",
		"Mcp-Name":             "search_multicard_docs",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Wrong Mcp-Name: rejected.
	rec = postMCP(t, server, body, map[string]string{
		"MCP-Protocol-Version": ModernProtocolVersion,
		"Mcp-Method":           "tools/call",
		"Mcp-Name":             "get_multicard_doc",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != errCodeHeaderMismatch {
		t.Fatalf("expected HeaderMismatch error, got %+v", resp.Error)
	}
}

func TestHandleHTTPMCPModernResourcesReadRequiresMatchingMcpName(t *testing.T) {
	server := testServer(t)
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/read",
		"params": map[string]any{
			"uri":   "doc://endpoints/example.md",
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": ModernProtocolVersion},
		},
	}
	rec := postMCP(t, server, body, map[string]string{
		"MCP-Protocol-Version": ModernProtocolVersion,
		"Mcp-Method":           "resources/read",
		"Mcp-Name":             "doc://something-else.md",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != errCodeHeaderMismatch {
		t.Fatalf("expected HeaderMismatch error, got %+v", resp.Error)
	}
}

func TestHandleHTTPMCPGetWithEventStreamAcceptReturns405(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	server.handleHTTPMCP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow header = %q, want %q", got, http.MethodPost)
	}
}

func TestHandleHTTPMCPPlainGetStillReturnsInfoPage(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()
	server.handleHTTPMCP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := out["message"]; !ok {
		t.Fatalf("expected the info message field, got %+v", out)
	}
}

func TestHandleHTTPMCPUnsupportedProtocolVersionReturns400(t *testing.T) {
	server := testServer(t)
	rec := postMCP(t, server, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": "1900-01-01"},
		},
	}, map[string]string{
		"MCP-Protocol-Version": "1900-01-01",
		"Mcp-Method":           "tools/list",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != errCodeUnsupportedProtocolVersion {
		t.Fatalf("expected UnsupportedProtocolVersionError, got %+v", resp.Error)
	}
}

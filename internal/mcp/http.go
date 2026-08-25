package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"multicard-mcp-go/internal/oauth"
)

// ServeHTTP serves MCP-compatible JSON-RPC over HTTP, plus health, readiness,
// and OAuth endpoints. The /mcp endpoint requires a valid bearer token issued
// by auth; every other OAuth route (discovery metadata, registration,
// authorize, token) is public, as required for clients to bootstrap the flow.
func (s *Server) ServeHTTP(listenAddr string, auth *oauth.Server) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.Handle("/mcp", auth.RequireToken(http.HandlerFunc(s.handleHTTPMCP)))
	mux.HandleFunc("/", s.handleRoot)
	auth.RegisterRoutes(mux)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mux.ServeHTTP(w, r)
		fmt.Fprintf(os.Stderr, "%s %s %s\n", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return srv.ListenAndServe()
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":                        "multicard-docs-mcp-go",
		"version":                     ServerVersion,
		"transport":                   "http-json-rpc",
		"mcp_endpoint":                "/mcp",
		"health_endpoint":             "/healthz",
		"readiness_endpoint":          "/readyz",
		"docs_loaded":                 s.corpus.TotalDocs(),
		"authorization":               "OAuth 2.1 (authorization code + PKCE)",
		"protected_resource_metadata": "/.well-known/oauth-protected-resource",
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "docs_loaded": s.corpus.TotalDocs()})
}

func (s *Server) handleHTTPMCP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "failed to read request body"}})
			return
		}

		if id, hdrErr := checkModernHeaders(r, body); hdrErr != nil {
			writeJSON(w, http.StatusBadRequest, response{JSONRPC: "2.0", ID: id, Error: hdrErr})
			return
		}

		resp, shouldReply := s.ProcessRequestBody(body)
		if !shouldReply {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		status := http.StatusOK
		if resp.Error != nil && resp.Error.Code == errCodeUnsupportedProtocolVersion {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, resp)
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "POST JSON-RPC requests to this endpoint",
			"example": map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/list",
				"params":  map[string]any{},
			},
		})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// checkModernHeaders enforces the Streamable HTTP header requirements the
// 2026-07-28 spec adds for modern (per-request-versioned) requests:
// MCP-Protocol-Version, Mcp-Method, and — for tools/call and resources/read —
// Mcp-Name must each mirror the corresponding value in the JSON-RPC body.
// Legacy requests (no params._meta protocol version) are left untouched, so
// existing handshake-based clients are unaffected.
func checkModernHeaders(r *http.Request, body []byte) (id any, rpcErr *rpcError) {
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, nil // malformed JSON is reported by the normal parser downstream
	}

	version := modernVersionFromParams(req.Params)
	if version == "" {
		return req.ID, nil // legacy request: no header requirements apply
	}

	if got := r.Header.Get("MCP-Protocol-Version"); got != version {
		return req.ID, headerMismatch("MCP-Protocol-Version", got, version)
	}
	if got := r.Header.Get("Mcp-Method"); got != req.Method {
		return req.ID, headerMismatch("Mcp-Method", got, req.Method)
	}

	switch req.Method {
	case "tools/call":
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(req.Params, &p)
		if got := r.Header.Get("Mcp-Name"); got != p.Name {
			return req.ID, headerMismatch("Mcp-Name", got, p.Name)
		}
	case "resources/read":
		var p struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(req.Params, &p)
		if got := r.Header.Get("Mcp-Name"); got != p.URI {
			return req.ID, headerMismatch("Mcp-Name", got, p.URI)
		}
	}

	return req.ID, nil
}

func headerMismatch(header, got, want string) *rpcError {
	return &rpcError{
		Code:    errCodeHeaderMismatch,
		Message: fmt.Sprintf("Header mismatch: %s header value %q does not match body value %q", header, got, want),
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

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
		resp, shouldReply := s.ProcessRequestBody(body)
		if !shouldReply {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeJSON(w, http.StatusOK, resp)
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

// Package mcp implements the JSON-RPC methods, tools, resources, and transports for this MCP server.
package mcp

import (
	"encoding/json"
	"sync"

	"multicard-mcp-go/internal/docs"
)

const (
	// LegacyProtocolVersion is the newest handshake-based ("legacy" per the
	// 2026-07-28 spec's terminology) MCP protocol version this server
	// advertises during an initialize request.
	LegacyProtocolVersion = "2025-11-25"
	// ModernProtocolVersion is the newest stateless, per-request-versioned
	// ("modern") MCP protocol version this server accepts.
	ModernProtocolVersion = "2026-07-28"
	// ServerVersion is the application version advertised to MCP clients and HTTP status responses.
	ServerVersion = "0.3.0"

	// metaProtocolVersionKey is the _meta key a modern (2026-07-28+) request
	// uses to declare its protocol version on every call, replacing the
	// once-per-connection initialize handshake used by legacy versions.
	metaProtocolVersionKey = "io.modelcontextprotocol/protocolVersion"

	// errCodeHeaderMismatch and errCodeUnsupportedProtocolVersion are the
	// JSON-RPC error codes the 2026-07-28 spec defines for the two new
	// modern-request validation failures.
	errCodeHeaderMismatch             = -32020
	errCodeUnsupportedProtocolVersion = -32022
)

// Server serves Multicard docs through MCP tools and resources.
type Server struct {
	corpus *docs.Corpus
	outMu  sync.Mutex
}

// NewServer creates an MCP server backed by a loaded docs corpus.
func NewServer(corpus *docs.Corpus) *Server {
	return &Server{corpus: corpus}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Meta      map[string]any `json:"_meta,omitempty"`
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

// modernVersionFromParams reads the modern per-request protocol version
// declaration out of a request's params._meta, returning "" for a legacy
// request that carries no such field. This is transport-agnostic: both the
// stdio and HTTP transports use it to tell modern and legacy requests apart.
func modernVersionFromParams(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var envelope struct {
		Meta map[string]json.RawMessage `json:"_meta"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	versionRaw, ok := envelope.Meta[metaProtocolVersionKey]
	if !ok {
		return ""
	}
	var version string
	if err := json.Unmarshal(versionRaw, &version); err != nil {
		return ""
	}
	return version
}

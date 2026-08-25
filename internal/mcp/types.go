// Package mcp implements the JSON-RPC methods, tools, resources, and transports for this MCP server.
package mcp

import (
	"encoding/json"
	"sync"

	"multicard-mcp-go/internal/docs"
)

const (
	// ProtocolVersion is the MCP protocol version advertised during initialize.
	ProtocolVersion = "2024-11-05"
	// ServerVersion is the application version advertised to MCP clients and HTTP status responses.
	ServerVersion = "0.2.0"
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
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Meta      map[string]any `json:"_meta,omitempty"`
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

package mcp

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"multicard-mcp-go/internal/docs"
)

func TestProcessRequestBodyToolsList(t *testing.T) {
	corp, err := docs.LoadCorpus(filepath.Join("..", "..", "multicard-docs"))
	if err != nil {
		t.Fatalf("LoadCorpus() error = %v", err)
	}
	server := NewServer(corp)
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	resp, ok := server.ProcessRequestBody(body)
	if !ok {
		t.Fatalf("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected rpc error: %+v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatalf("expected result")
	}
}

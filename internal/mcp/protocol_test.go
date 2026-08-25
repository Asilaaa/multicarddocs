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

func testServer(t *testing.T) *Server {
	t.Helper()
	corp, err := docs.LoadCorpus(filepath.Join("..", "..", "multicard-docs"))
	if err != nil {
		t.Fatalf("LoadCorpus() error = %v", err)
	}
	return NewServer(corp)
}

func TestInitializeAdvertisesLatestLegacyProtocolVersion(t *testing.T) {
	server := testServer(t)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2024-11-05"},
	})
	resp, ok := server.ProcessRequestBody(body)
	if !ok || resp.Error != nil {
		t.Fatalf("unexpected response: ok=%v resp=%+v", ok, resp)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}
	if got := result["protocolVersion"]; got != LegacyProtocolVersion {
		t.Fatalf("protocolVersion = %v, want %v", got, LegacyProtocolVersion)
	}
}

func TestServerDiscoverListsSupportedVersions(t *testing.T) {
	server := testServer(t)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "server/discover",
		"params": map[string]any{
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": ModernProtocolVersion},
		},
	})
	resp, ok := server.ProcessRequestBody(body)
	if !ok || resp.Error != nil {
		t.Fatalf("unexpected response: ok=%v resp=%+v", ok, resp)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}
	if result["resultType"] != "complete" {
		t.Fatalf("resultType = %v, want complete", result["resultType"])
	}
	versions, ok := result["supportedVersions"].([]string)
	if !ok {
		t.Fatalf("supportedVersions has unexpected type %T", result["supportedVersions"])
	}
	found := map[string]bool{}
	for _, v := range versions {
		found[v] = true
	}
	if !found[ModernProtocolVersion] || !found[LegacyProtocolVersion] {
		t.Fatalf("supportedVersions = %v, want both %q and %q", versions, ModernProtocolVersion, LegacyProtocolVersion)
	}
}

func TestServerDiscoverWorksWithoutModernMeta(t *testing.T) {
	// A client is free to call server/discover as a plain probe before it
	// knows anything about the server; it shouldn't need to declare a
	// protocol version to get an answer.
	server := testServer(t)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "server/discover",
		"params":  map[string]any{},
	})
	resp, ok := server.ProcessRequestBody(body)
	if !ok || resp.Error != nil {
		t.Fatalf("unexpected response: ok=%v resp=%+v", ok, resp)
	}
}

func TestModernRequestWithUnsupportedVersionRejected(t *testing.T) {
	server := testServer(t)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": "1900-01-01"},
		},
	})
	resp, ok := server.ProcessRequestBody(body)
	if !ok {
		t.Fatalf("expected a response")
	}
	if resp.Error == nil {
		t.Fatalf("expected an error for an unsupported protocol version")
	}
	if resp.Error.Code != errCodeUnsupportedProtocolVersion {
		t.Fatalf("error code = %d, want %d", resp.Error.Code, errCodeUnsupportedProtocolVersion)
	}
	data, ok := resp.Error.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected error.data to be a map, got %T", resp.Error.Data)
	}
	if data["requested"] != "1900-01-01" {
		t.Fatalf("data.requested = %v, want 1900-01-01", data["requested"])
	}
}

func TestModernRequestWithSupportedVersionIsProcessedNormally(t *testing.T) {
	server := testServer(t)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": ModernProtocolVersion},
		},
	})
	resp, ok := server.ProcessRequestBody(body)
	if !ok || resp.Error != nil {
		t.Fatalf("unexpected response: ok=%v resp=%+v", ok, resp)
	}
	if resp.Result == nil {
		t.Fatalf("expected a result")
	}
}

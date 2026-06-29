package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestLoadCorpus(t *testing.T) {
	corp, err := loadCorpus(filepath.Join("multicard-docs"))
	if err != nil {
		t.Fatalf("loadCorpus() error = %v", err)
	}
	if corp.totalDocs == 0 {
		t.Fatalf("expected docs to be loaded")
	}
}

func TestSearchFindsAuthTokenDoc(t *testing.T) {
	corp, err := loadCorpus(filepath.Join("multicard-docs"))
	if err != nil {
		t.Fatalf("loadCorpus() error = %v", err)
	}
	results := corp.search("how to get token", 3)
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}
	if got := results[0].Doc.Path; got != "endpoints/получение-токена-19729295e0.md" {
		t.Fatalf("unexpected top result: %s", got)
	}
}

func TestProcessRequestBodyToolsList(t *testing.T) {
	corp, err := loadCorpus(filepath.Join("multicard-docs"))
	if err != nil {
		t.Fatalf("loadCorpus() error = %v", err)
	}
	server := &mcpServer{corpus: corp}
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	resp, ok := server.processRequestBody(body)
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

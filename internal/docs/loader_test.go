package docs

import (
	"path/filepath"
	"testing"
)

func TestLoadCorpus(t *testing.T) {
	corp, err := LoadCorpus(filepath.Join("..", "..", "multicard-docs"))
	if err != nil {
		t.Fatalf("LoadCorpus() error = %v", err)
	}
	if corp.TotalDocs() == 0 {
		t.Fatalf("expected docs to be loaded")
	}
}

func TestSearchFindsAuthTokenDoc(t *testing.T) {
	corp, err := LoadCorpus(filepath.Join("..", "..", "multicard-docs"))
	if err != nil {
		t.Fatalf("LoadCorpus() error = %v", err)
	}
	results := corp.Search("how to get token", 3)
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}
	if got := results[0].Doc.Path; got != "endpoints/получение-токена-19729295e0.md" {
		t.Fatalf("unexpected top result: %s", got)
	}
}

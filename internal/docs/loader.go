package docs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadCorpus reads every markdown file under root and builds a searchable in-memory corpus.
func LoadCorpus(root string) (*Corpus, error) {
	var docs []Document
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}

		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		doc := parseDocument(rel, string(body))
		docs = append(docs, doc)
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	corp := &Corpus{
		docs:      docs,
		df:        map[string]int{},
		byURI:     map[string]Document{},
		byPath:    map[string]Document{},
		totalDocs: len(docs),
	}

	totalTokens := 0
	for _, doc := range docs {
		corp.byURI[doc.URI] = doc
		corp.byPath[doc.Path] = doc
		totalTokens += doc.TokenCount
		seen := map[string]struct{}{}
		for token := range doc.Tokens {
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			corp.df[token]++
		}
	}
	if corp.totalDocs > 0 {
		corp.avgDocLength = float64(totalTokens) / float64(corp.totalDocs)
	}
	if corp.avgDocLength == 0 {
		corp.avgDocLength = 1
	}
	return corp, nil
}

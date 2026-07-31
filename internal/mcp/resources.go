package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Server) resources() []map[string]any {
	docs := s.corpus.Documents()
	resources := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		descParts := []string{doc.Category}
		if doc.Method != "" && doc.EndpointPath != "" {
			descParts = append(descParts, doc.Method+" "+doc.EndpointPath)
		}
		if doc.Summary != "" {
			descParts = append(descParts, doc.Summary)
		}
		resources = append(resources, map[string]any{
			"uri":         doc.URI,
			"name":        doc.Title,
			"mimeType":    "text/markdown",
			"description": strings.Join(descParts, " | "),
		})
	}
	return resources
}

func (s *Server) readResource(raw json.RawMessage) (map[string]any, error) {
	var params resourceReadParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("invalid resources/read params: %w", err)
	}
	doc, ok := s.corpus.ByURI(params.URI)
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", params.URI)
	}
	return map[string]any{
		"contents": []map[string]any{
			{
				"uri":      doc.URI,
				"mimeType": "text/markdown",
				"text":     doc.Content,
			},
		},
	}, nil
}

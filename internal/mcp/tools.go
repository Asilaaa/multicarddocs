package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"multicard-mcp-go/internal/render"
	"multicard-mcp-go/internal/util"
)

func (s *Server) tools() []map[string]any {
	return []map[string]any{
		{
			"name":        "search_multicard_docs",
			"description": "Search the Multicard markdown docs and return the most relevant pages/snippets for a user question.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "What you want to find in the Multicard docs.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results to return (default 5, max 10).",
						"minimum":     1,
						"maximum":     10,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "get_multicard_doc",
			"description": "Read one specific Multicard markdown page by its relative path, for example endpoints/получение-токена-19729295e0.md.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path inside multicard-docs.",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "answer_multicard_question",
			"description": "Return a concise answer grounded only in the Multicard docs, with steps and source pages.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"question": map[string]any{
						"type":        "string",
						"description": "Question about the Multicard API/docs.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "How many relevant source pages to use (default 4, max 8).",
						"minimum":     1,
						"maximum":     8,
					},
				},
				"required": []string{"question"},
			},
		},
	}
}

func (s *Server) callTool(raw json.RawMessage) (map[string]any, error) {
	var params toolCallParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("invalid tools/call params: %w", err)
	}

	switch params.Name {
	case "search_multicard_docs":
		query := strings.TrimSpace(asString(params.Arguments["query"]))
		if query == "" {
			return toolError("query is required"), nil
		}
		limit := util.Clamp(asInt(params.Arguments["limit"], 5), 1, 10)
		results := s.corpus.Search(query, limit)
		payload := map[string]any{
			"query":   query,
			"results": render.ResultsPayload(results),
		}
		text := render.SearchText(query, results)
		return toolSuccess(text, payload), nil
	case "get_multicard_doc":
		path := strings.TrimSpace(asString(params.Arguments["path"]))
		if path == "" {
			return toolError("path is required"), nil
		}
		path = strings.TrimPrefix(path, "/")
		doc, ok := s.corpus.ByPath(path)
		if !ok {
			return toolError(fmt.Sprintf("document not found: %s", path)), nil
		}
		payload := map[string]any{
			"path":     doc.Path,
			"title":    doc.Title,
			"category": doc.Category,
			"method":   doc.Method,
			"endpoint": doc.EndpointPath,
			"summary":  doc.Summary,
			"content":  doc.Content,
		}
		text := fmt.Sprintf("# %s\n\nPath: %s\nCategory: %s\n", doc.Title, doc.Path, doc.Category)
		if doc.Method != "" && doc.EndpointPath != "" {
			text += fmt.Sprintf("Endpoint: %s %s\n", doc.Method, doc.EndpointPath)
		}
		if doc.Summary != "" {
			text += fmt.Sprintf("Summary: %s\n", doc.Summary)
		}
		text += "\n" + doc.Content
		return toolSuccess(text, payload), nil
	case "answer_multicard_question":
		question := strings.TrimSpace(asString(params.Arguments["question"]))
		if question == "" {
			return toolError("question is required"), nil
		}
		limit := util.Clamp(asInt(params.Arguments["limit"], 4), 1, 8)
		results := s.corpus.Search(question, limit)
		answer, sources := render.Answer(question, results)
		payload := map[string]any{
			"question": question,
			"answer":   answer,
			"sources":  sources,
		}
		return toolSuccess(answer, payload), nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func toolSuccess(text string, payload map[string]any) map[string]any {
	return map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": text,
		}},
		"structuredContent": payload,
		"isError":           false,
	}
}

func toolError(message string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": message,
		}},
		"isError": true,
	}
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	case json.Number:
		return x.String()
	default:
		return ""
	}
}

func asInt(v any, def int) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case json.Number:
		i, err := x.Int64()
		if err == nil {
			return int(i)
		}
	}
	return def
}

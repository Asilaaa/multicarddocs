// Package render converts search results into human-readable MCP/CLI text and structured payloads.
package render

import (
	"fmt"
	"math"
	"strings"

	"multicard-mcp-go/internal/docs"
	"multicard-mcp-go/internal/util"
)

// SearchText renders search results for humans.
func SearchText(query string, results []docs.ScoredDoc) string {
	if len(results) == 0 {
		return fmt.Sprintf("No relevant Multicard docs found for: %s", query)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Top Multicard docs for: %s\n\n", query)
	for i, res := range results {
		fmt.Fprintf(&b, "%d. %s\n", i+1, res.Doc.Title)
		fmt.Fprintf(&b, "   Path: %s\n", res.Doc.Path)
		if res.Doc.Method != "" && res.Doc.EndpointPath != "" {
			fmt.Fprintf(&b, "   Endpoint: %s %s\n", res.Doc.Method, res.Doc.EndpointPath)
		}
		if res.Doc.Summary != "" {
			fmt.Fprintf(&b, "   Summary: %s\n", res.Doc.Summary)
		}
		if len(res.Doc.RequiredFields) > 0 {
			fmt.Fprintf(&b, "   Required fields: %s\n", strings.Join(res.Doc.RequiredFields[:util.Min(len(res.Doc.RequiredFields), 10)], ", "))
		}
		fmt.Fprintf(&b, "   Snippet:\n%s\n\n", Indent(res.Snippet, "     "))
	}
	return strings.TrimSpace(b.String())
}

// Answer writes a concise, grounded answer using the top search results as sources.
func Answer(question string, results []docs.ScoredDoc) (string, []map[string]any) {
	if len(results) == 0 {
		answer := "I could not find a relevant answer in the loaded Multicard docs. Try a more specific question, endpoint name, path, field, or error code."
		return answer, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Based on the Multicard docs, here is the most relevant guidance for: %s\n\n", question)

	primary := results[0].Doc
	fmt.Fprintf(&b, "Main source: %s", primary.Title)
	if primary.Method != "" && primary.EndpointPath != "" {
		fmt.Fprintf(&b, " (%s %s)", primary.Method, primary.EndpointPath)
	}
	fmt.Fprintf(&b, "\n")
	if primary.Summary != "" {
		fmt.Fprintf(&b, "- Summary: %s\n", primary.Summary)
	}
	if len(primary.RequiredFields) > 0 {
		fmt.Fprintf(&b, "- Important fields mentioned in the doc: %s\n", strings.Join(primary.RequiredFields[:util.Min(len(primary.RequiredFields), 12)], ", "))
	}
	if len(primary.ErrorCodes) > 0 {
		fmt.Fprintf(&b, "- Related error codes in this page: %s\n", strings.Join(primary.ErrorCodes[:util.Min(len(primary.ErrorCodes), 8)], ", "))
	}
	fmt.Fprintf(&b, "- Relevant excerpt:\n%s\n", Indent(docs.BestSnippet(results[0].Doc, docs.TokenizeWithSynonyms(question), strings.ToLower(question)), "  "))

	if len(results) > 1 {
		fmt.Fprintf(&b, "\nOther useful pages:\n")
		for _, res := range results[1:util.Min(len(results), 4)] {
			fmt.Fprintf(&b, "- %s", res.Doc.Title)
			if res.Doc.Method != "" && res.Doc.EndpointPath != "" {
				fmt.Fprintf(&b, " (%s %s)", res.Doc.Method, res.Doc.EndpointPath)
			}
			fmt.Fprintf(&b, " — %s\n", res.Doc.Path)
		}
	}

	fmt.Fprintf(&b, "\nSources:\n")
	sources := make([]map[string]any, 0, len(results))
	for _, res := range results {
		fmt.Fprintf(&b, "- %s\n", res.Doc.Path)
		sources = append(sources, map[string]any{
			"path":     res.Doc.Path,
			"title":    res.Doc.Title,
			"method":   res.Doc.Method,
			"endpoint": res.Doc.EndpointPath,
			"summary":  res.Doc.Summary,
			"snippet":  res.Snippet,
			"score":    Round2(res.Score),
		})
	}
	return strings.TrimSpace(b.String()), sources
}

// ResultsPayload transforms search results into JSON-friendly maps.
func ResultsPayload(results []docs.ScoredDoc) []map[string]any {
	out := make([]map[string]any, 0, len(results))
	for _, res := range results {
		out = append(out, map[string]any{
			"path":            res.Doc.Path,
			"title":           res.Doc.Title,
			"category":        res.Doc.Category,
			"method":          res.Doc.Method,
			"endpoint":        res.Doc.EndpointPath,
			"summary":         res.Doc.Summary,
			"required_fields": res.Doc.RequiredFields,
			"error_codes":     res.Doc.ErrorCodes,
			"score":           Round2(res.Score),
			"snippet":         res.Snippet,
		})
	}
	return out
}

// Indent prefixes every line in s.
func Indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

// Round2 rounds a float to two decimal places.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

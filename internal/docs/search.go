package docs

import (
	"math"
	"sort"
	"strings"

	"multicard-mcp-go/internal/util"
)

// Search ranks corpus documents for a query and returns the most relevant results.
func (c *Corpus) Search(query string, limit int) []ScoredDoc {
	if limit <= 0 {
		limit = 5
	}
	rawTokens := Tokenize(query)
	queryTokens := TokenizeWithSynonyms(query)
	if len(queryTokens) == 0 {
		return nil
	}
	queryLower := strings.ToLower(query)
	queryTopics := sliceToSet(DetectTopics(query))
	results := make([]ScoredDoc, 0, len(c.docs))
	for _, doc := range c.docs {
		score := c.scoreDoc(doc, queryTokens, rawTokens, queryLower, queryTopics)
		if score <= 0 {
			continue
		}
		results = append(results, ScoredDoc{
			Doc:     doc,
			Score:   score,
			Snippet: BestSnippet(doc, queryTokens, queryLower),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if math.Abs(results[i].Score-results[j].Score) > 0.0001 {
			return results[i].Score > results[j].Score
		}
		return results[i].Doc.Path < results[j].Doc.Path
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func (c *Corpus) scoreDoc(doc Document, queryTokens map[string]int, rawTokens []string, queryLower string, queryTopics map[string]struct{}) float64 {
	var score float64
	docLen := float64(util.Max(doc.TokenCount, 1))
	for token, qtf := range queryTokens {
		tf := float64(doc.Tokens[token])
		if tf == 0 {
			continue
		}
		df := float64(c.df[token])
		idf := math.Log(1 + (float64(c.totalDocs)-df+0.5)/(df+0.5))
		bm25 := idf * ((tf * 2.2) / (tf + 1.2*(1-0.75+0.75*(docLen/c.avgDocLength))))
		score += bm25 * float64(qtf)
	}

	titleLower := strings.ToLower(doc.Title)
	slugLower := strings.ToLower(doc.Slug)
	summaryLower := strings.ToLower(doc.Summary)
	pathLower := strings.ToLower(doc.Path)
	endpointLower := strings.ToLower(doc.EndpointPath)
	plainLower := strings.ToLower(doc.Plain)

	for token := range queryTokens {
		if strings.Contains(titleLower, token) {
			score += 8
		}
		if strings.Contains(slugLower, token) {
			score += 6
		}
		if strings.Contains(summaryLower, token) {
			score += 5
		}
		if strings.Contains(pathLower, token) {
			score += 4
		}
		if strings.Contains(endpointLower, token) {
			score += 6
		}
	}
	if queryLower != "" {
		if strings.Contains(titleLower, queryLower) {
			score += 12
		}
		if doc.EndpointPath != "" && strings.Contains(strings.ToLower(doc.EndpointPath), queryLower) {
			score += 12
		}
		if strings.Contains(plainLower, queryLower) {
			score += 4
		}
	}
	if doc.Method != "" && strings.Contains(queryLower, strings.ToLower(doc.Method)) {
		score += 3
	}

	if len(rawTokens) > 0 {
		titleHits := 0
		summaryHits := 0
		endpointHits := 0
		for _, token := range rawTokens {
			if strings.Contains(titleLower, token) || strings.Contains(slugLower, token) {
				titleHits++
			}
			if strings.Contains(summaryLower, token) {
				summaryHits++
			}
			if strings.Contains(endpointLower, token) || strings.Contains(pathLower, token) {
				endpointHits++
			}
		}
		score += float64(titleHits * 6)
		score += float64(summaryHits * 4)
		score += float64(endpointHits * 4)
		if titleHits == len(rawTokens) {
			score += 18
		}
		if summaryHits == len(rawTokens) {
			score += 10
		}
		if endpointHits == len(rawTokens) {
			score += 8
		}
	}

	titleTokens := Tokenize(doc.Title + " " + doc.Slug)
	if len(titleTokens) > 0 {
		hits := 0
		for _, t := range titleTokens {
			if _, ok := queryTokens[t]; ok {
				hits++
				continue
			}
			if stem := stemToken(t); stem != t {
				if _, ok := queryTokens[stem]; ok {
					hits++
				}
			}
		}
		if hits > 0 {
			density := float64(hits) / float64(len(titleTokens))
			score += float64(hits*3) + density*20
		}
	}

	if len(queryTopics) > 0 && len(doc.Topics) > 0 {
		for _, topic := range doc.Topics {
			if _, ok := queryTopics[topic]; !ok {
				score -= topicPenalty(topic)
			}
		}
	}

	if _, wantsToken := queryTopics["token"]; wantsToken {
		_, wantsCard := queryTopics["card"]
		if !wantsCard {
			if doc.EndpointPath == "/auth" || strings.Contains(titleLower, "получение токен") {
				score += 30
			}
			for _, topic := range doc.Topics {
				if topic == "card" {
					score -= 20
					break
				}
			}
		}
	}
	return score
}

// BestSnippet extracts the highest-signal markdown section for a result.
func BestSnippet(doc Document, queryTokens map[string]int, queryLower string) string {
	lines := strings.Split(doc.Content, "\n")
	bestIdx := 0
	bestScore := -1
	for i, line := range lines {
		lower := strings.ToLower(line)
		score := 0
		for token := range queryTokens {
			if strings.Contains(lower, token) {
				score += 2
			}
		}
		if queryLower != "" && strings.Contains(lower, queryLower) {
			score += 5
		}
		if strings.HasPrefix(strings.TrimSpace(line), "summary:") || strings.HasPrefix(strings.TrimSpace(line), "description:") {
			score++
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	start := util.Max(bestIdx-3, 0)
	end := util.Min(bestIdx+5, len(lines))
	chunk := strings.Join(lines[start:end], "\n")
	chunk = strings.TrimSpace(chunk)
	if chunk == "" {
		chunk = strings.TrimSpace(doc.Content)
	}
	if len([]rune(chunk)) > 800 {
		r := []rune(chunk)
		chunk = string(r[:800]) + "…"
	}
	return chunk
}

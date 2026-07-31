// Package docs loads, indexes, and searches the bundled Multicard markdown documentation.
package docs

// Document is one markdown page enriched with extracted metadata used for MCP resources and search ranking.
type Document struct {
	Path           string
	URI            string
	Category       string
	Title          string
	Content        string
	Plain          string
	Slug           string
	EndpointPath   string
	Method         string
	Summary        string
	Tags           []string
	ErrorCodes     []string
	RequiredFields []string
	Topics         []string
	Tokens         map[string]int
	TokenCount     int
}

// Corpus is the searchable in-memory index of all loaded documentation pages.
type Corpus struct {
	docs         []Document
	totalDocs    int
	avgDocLength float64
	df           map[string]int
	byURI        map[string]Document
	byPath       map[string]Document
}

// ScoredDoc is a search result with its ranking score and a relevant markdown snippet.
type ScoredDoc struct {
	Doc     Document `json:"doc"`
	Score   float64  `json:"score"`
	Snippet string   `json:"snippet"`
}

// Documents returns all loaded documents in deterministic path order.
func (c *Corpus) Documents() []Document {
	out := make([]Document, len(c.docs))
	copy(out, c.docs)
	return out
}

// TotalDocs returns the number of loaded markdown pages.
func (c *Corpus) TotalDocs() int { return c.totalDocs }

// ByURI returns a document by MCP resource URI.
func (c *Corpus) ByURI(uri string) (Document, bool) {
	doc, ok := c.byURI[uri]
	return doc, ok
}

// ByPath returns a document by relative path inside multicard-docs.
func (c *Corpus) ByPath(path string) (Document, bool) {
	doc, ok := c.byPath[path]
	return doc, ok
}

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	protocolVersion = "2024-11-05"
	serverVersion   = "0.2.0"
)

var (
	reH1         = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	reEndpoint   = regexp.MustCompile(`(?m)^\s{2}(/[^:\n]+):\s*$`)
	reMethod     = regexp.MustCompile(`(?m)^\s{4}(get|post|put|delete|patch|options|head):\s*$`)
	reSummary    = regexp.MustCompile(`(?m)^\s{6}summary:\s*(.+?)\s*$`)
	reTag        = regexp.MustCompile(`(?m)^\s{8}-\s+(.+?)\s*$`)
	reErrorCode  = regexp.MustCompile(`\bERROR_[A-Z0-9_]+\b`)
	reFieldName  = regexp.MustCompile(`(?m)^\s{16}([a-zA-Z0-9_]+):\s*$`)
	reCodeFence  = regexp.MustCompile("```[\\s\\S]*?```")
	reMultiSpace = regexp.MustCompile(`\s+`)
	reContentLen = regexp.MustCompile(`(?i)^Content-Length:\s*(\d+)\s*$`)
)

type document struct {
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

type corpus struct {
	docs         []document
	totalDocs    int
	avgDocLength float64
	df           map[string]int
	byURI        map[string]document
	byPath       map[string]document
}

type scoredDoc struct {
	Doc     document `json:"doc"`
	Score   float64  `json:"score"`
	Snippet string   `json:"snippet"`
}

type mcpServer struct {
	corpus *corpus
	outMu  sync.Mutex
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Meta      map[string]any `json:"_meta,omitempty"`
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

type getDocArgs struct {
	Path string `json:"path"`
}

type searchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type answerArgs struct {
	Question string `json:"question"`
	Limit    int    `json:"limit"`
}

func main() {
	var docsDirFlag string
	var searchFlag string
	var askFlag string
	var getDocFlag string
	var listenAddrFlag string
	var httpFlag bool
	var limitFlag int
	flag.StringVar(&docsDirFlag, "docs-dir", "", "Path to the multicard-docs directory")
	flag.StringVar(&searchFlag, "search", "", "Run a one-off local search instead of starting the MCP server")
	flag.StringVar(&askFlag, "ask", "", "Ask a one-off question from the local docs instead of starting the MCP server")
	flag.StringVar(&getDocFlag, "get-doc", "", "Print one full markdown doc by relative path instead of starting the MCP server")
	flag.BoolVar(&httpFlag, "http", false, "Run as an HTTP JSON-RPC service instead of stdio MCP")
	flag.StringVar(&listenAddrFlag, "listen-addr", envOrDefault("LISTEN_ADDR", "127.0.0.1:8080"), "HTTP listen address for --http mode")
	flag.IntVar(&limitFlag, "limit", 5, "Result limit for --search or --ask")
	flag.Parse()

	docsDir, err := resolveDocsDir(docsDirFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve docs dir: %v\n", err)
		os.Exit(1)
	}

	corp, err := loadCorpus(docsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load docs: %v\n", err)
		os.Exit(1)
	}

	if searchFlag != "" {
		results := corp.search(searchFlag, clamp(limitFlag, 1, 10))
		fmt.Println(renderSearchText(searchFlag, results))
		return
	}
	if askFlag != "" {
		results := corp.search(askFlag, clamp(limitFlag, 1, 8))
		answer, _ := composeAnswer(askFlag, results)
		fmt.Println(answer)
		return
	}
	if getDocFlag != "" {
		path := strings.TrimPrefix(strings.TrimSpace(getDocFlag), "/")
		doc, ok := corp.byPath[path]
		if !ok {
			fmt.Fprintf(os.Stderr, "document not found: %s\n", path)
			os.Exit(1)
		}
		fmt.Println(doc.Content)
		return
	}

	server := &mcpServer{corpus: corp}
	if httpFlag {
		fmt.Fprintf(os.Stderr, "multicard-mcp-go: loaded %d docs from %s\n", corp.totalDocs, docsDir)
		fmt.Fprintf(os.Stderr, "multicard-mcp-go: serving HTTP JSON-RPC on %s\n", listenAddrFlag)
		if err := server.serveHTTP(listenAddrFlag); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "http server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "multicard-mcp-go: loaded %d docs from %s\n", corp.totalDocs, docsDir)
	fmt.Fprintln(os.Stderr, "multicard-mcp-go: waiting for MCP client on stdio")
	if err := server.serve(os.Stdin, os.Stdout); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func resolveDocsDir(flagValue string) (string, error) {
	candidates := make([]string, 0, 6)
	if flagValue != "" {
		candidates = append(candidates, flagValue)
	}
	if env := os.Getenv("MULTICARD_DOCS_DIR"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates, "multicard-docs")

	exe, _ := os.Executable()
	if exe != "" {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "multicard-docs"),
			filepath.Join(exeDir, "..", "multicard-docs"),
		)
	}

	cwd, _ := os.Getwd()
	if cwd != "" {
		candidates = append(candidates, filepath.Join(cwd, "..", "multicard-docs"))
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		info, err := os.Stat(abs)
		if err == nil && info.IsDir() {
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not find multicard-docs directory")
}

func loadCorpus(root string) (*corpus, error) {
	var docs []document
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
	corp := &corpus{
		docs:      docs,
		df:        map[string]int{},
		byURI:     map[string]document{},
		byPath:    map[string]document{},
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

func parseDocument(relPath, content string) document {
	category := filepath.Base(filepath.Dir(relPath))
	title := strings.TrimSpace(filepath.Base(relPath))
	if match := reH1.FindStringSubmatch(content); len(match) > 1 {
		title = strings.TrimSpace(match[1])
	}

	endpointPath := ""
	if match := reEndpoint.FindStringSubmatch(content); len(match) > 1 {
		endpointPath = strings.TrimSpace(match[1])
	}
	method := ""
	if match := reMethod.FindStringSubmatch(content); len(match) > 1 {
		method = strings.ToUpper(strings.TrimSpace(match[1]))
	}
	summary := ""
	if match := reSummary.FindStringSubmatch(content); len(match) > 1 {
		summary = cleanScalarText(match[1])
	}

	tagsMap := map[string]struct{}{}
	for _, match := range reTag.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			tagsMap[strings.TrimSpace(match[1])] = struct{}{}
		}
	}
	var tags []string
	for tag := range tagsMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	errorCodesMap := map[string]struct{}{}
	for _, code := range reErrorCode.FindAllString(content, -1) {
		errorCodesMap[code] = struct{}{}
	}
	var errorCodes []string
	for code := range errorCodesMap {
		errorCodes = append(errorCodes, code)
	}
	sort.Strings(errorCodes)

	requiredFields := extractRequiredFields(content)
	plain := markdownToPlain(content)
	topics := detectTopics(title + " " + relPath + " " + summary + " " + endpointPath)
	slug := buildSlug(relPath)
	metaText := strings.Join([]string{
		title,
		relPath,
		slug,
		category,
		endpointPath,
		method,
		summary,
		strings.Join(tags, " "),
		strings.Join(requiredFields, " "),
		strings.Join(errorCodes, " "),
	}, "\n")
	fullPlain := plain + "\n" + metaText
	tokens := tokenizeWithSynonyms(fullPlain)
	count := 0
	for _, tf := range tokens {
		count += tf
	}

	uri := "doc://multicard/" + relPath
	return document{
		Path:           relPath,
		URI:            uri,
		Category:       category,
		Title:          title,
		Content:        content,
		Plain:          fullPlain,
		Slug:           slug,
		EndpointPath:   endpointPath,
		Method:         method,
		Summary:        summary,
		Tags:           tags,
		ErrorCodes:     errorCodes,
		RequiredFields: requiredFields,
		Topics:         topics,
		Tokens:         tokens,
		TokenCount:     count,
	}
}

func extractRequiredFields(content string) []string {
	lines := strings.Split(content, "\n")
	var result []string
	seen := map[string]struct{}{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) != "required:" {
			continue
		}
		indent := leadingSpaces(line)
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			trimmed := strings.TrimSpace(next)
			if trimmed == "" {
				continue
			}
			nextIndent := leadingSpaces(next)
			if nextIndent <= indent {
				break
			}
			if strings.HasPrefix(strings.TrimSpace(next), "-") {
				field := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(next), "-"))
				if field != "" {
					if _, ok := seen[field]; !ok {
						seen[field] = struct{}{}
						result = append(result, field)
					}
				}
			}
		}
	}
	return result
}

func leadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r == ' ' {
			count++
			continue
		}
		break
	}
	return count
}

func markdownToPlain(s string) string {
	s = strings.ReplaceAll(s, "```yaml", "\n")
	s = strings.ReplaceAll(s, "```json", "\n")
	s = strings.ReplaceAll(s, "```", "\n")
	s = reCodeFence.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "#", " ")
	s = strings.ReplaceAll(s, "`", " ")
	s = strings.ReplaceAll(s, "*", " ")
	s = strings.ReplaceAll(s, ">", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = reMultiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func buildSlug(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	return base
}

func tokenizeWithSynonyms(s string) map[string]int {
	tokens := map[string]int{}
	for _, token := range tokenize(s) {
		addToken(tokens, token)
		for _, alt := range synonyms[token] {
			addToken(tokens, alt)
		}
	}
	return tokens
}

func addToken(tokens map[string]int, token string) {
	if token == "" {
		return
	}
	tokens[token]++
	if stem := stemToken(token); stem != "" && stem != token {
		tokens[stem]++
	}
}

func tokenize(s string) []string {
	parts := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r))
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len([]rune(p)) < 2 && !containsDigit(p) {
			continue
		}
		if _, ok := stopwords[p]; ok {
			continue
		}
		out = append(out, p)
	}
	return out
}

func containsDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func stemToken(token string) string {
	if token == "" || containsDigit(token) {
		return token
	}
	r := []rune(token)
	if len(r) <= 4 {
		return token
	}

	lower := strings.ToLower(token)
	englishSuffixes := []string{"ing", "ed", "es", "s"}
	for _, suf := range englishSuffixes {
		if strings.HasSuffix(lower, suf) && len([]rune(lower)) > len([]rune(suf))+3 {
			return string([]rune(lower)[:len([]rune(lower))-len([]rune(suf))])
		}
	}

	russianSuffixes := []string{
		"иями", "ями", "ами", "ией", "ого", "ему", "ому", "ыми", "ими",
		"иях", "иях", "ах", "ях", "ам", "ям", "ом", "ем", "ов", "ев", "ей",
		"ия", "ья", "ий", "ый", "ой", "ая", "ое", "ые", "ие", "а", "я", "ы", "и", "е", "о", "у", "ю",
	}
	for _, suf := range russianSuffixes {
		if strings.HasSuffix(lower, suf) && len([]rune(lower)) > len([]rune(suf))+3 {
			return string([]rune(lower)[:len([]rune(lower))-len([]rune(suf))])
		}
	}
	return token
}

var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {}, "do": {}, "for": {},
	"from": {}, "how": {}, "i": {}, "if": {}, "in": {}, "is": {}, "it": {}, "me": {}, "my": {}, "of": {},
	"on": {}, "or": {}, "should": {}, "that": {}, "the": {}, "to": {}, "what": {}, "when": {}, "where": {},
	"which": {}, "with": {}, "you": {}, "your": {}, "это": {}, "как": {}, "что": {}, "для": {}, "или": {},
	"и": {}, "в": {}, "на": {}, "по": {}, "из": {}, "у": {}, "о": {}, "об": {},
}

var synonyms = map[string][]string{
	"token":         {"токен", "auth", "authorization", "авторизация"},
	"токен":         {"token", "auth", "авторизация"},
	"auth":          {"token", "токен", "авторизация"},
	"authorization": {"auth", "token", "токен"},
	"get":           {"получение", "получить", "информация", "info"},
	"obtain":        {"получение", "получить", "token", "info"},
	"получение":     {"get", "получить", "info"},
	"получить":      {"get", "получение", "info"},
	"info":          {"информация", "получение"},
	"информация":    {"info", "получение"},
	"create":        {"создание", "создать"},
	"создание":      {"create", "создать"},
	"создать":       {"create", "создание"},
	"check":         {"проверка", "статус"},
	"verify":        {"проверка", "статус"},
	"проверка":      {"check", "verify", "статус"},
	"delete":        {"удаление", "аннулирование", "отмена"},
	"remove":        {"удаление", "аннулирование"},
	"cancel":        {"отмена", "аннулирование"},
	"отмена":        {"cancel", "аннулирование"},
	"удаление":      {"delete", "remove", "аннулирование"},
	"аннулирование": {"delete", "cancel", "удаление"},
	"invoice":       {"инвойс", "checkout", "счет"},
	"инвойс":        {"invoice", "checkout", "счет"},
	"payment":       {"платеж", "оплата", "transaction", "транзакция"},
	"платеж":        {"payment", "оплата", "транзакция"},
	"оплата":        {"payment", "платеж", "checkout"},
	"refund":        {"возврат", "revert"},
	"возврат":       {"refund", "revert"},
	"hold":          {"холд", "холдирование", "блокировка"},
	"холд":          {"hold", "холдирование", "блокировка"},
	"холдирование":  {"hold", "холд", "блокировка"},
	"card":          {"карта", "карты", "pan", "token"},
	"карта":         {"card", "pan", "token"},
	"cards":         {"card", "карта"},
	"bind":          {"привязка", "binding", "token", "link"},
	"binding":       {"привязка", "bind", "token", "link"},
	"link":          {"ссылка", "привязка"},
	"привязка":      {"binding", "bind", "карта", "ссылка"},
	"ссылка":        {"link", "url", "checkout"},
	"callback":      {"webhook", "коллбэк", "callback_url"},
	"webhook":       {"callback", "коллбэк"},
	"коллбэк":       {"callback", "webhook"},
	"status":        {"статус", "state"},
	"статус":        {"status", "state"},
	"merchant":      {"мерчант", "partner", "партнер"},
	"мерчант":       {"merchant", "partner", "партнер"},
	"payout":        {"выплата", "withdrawal"},
	"выплата":       {"payout", "withdrawal"},
	"split":         {"расщепленный", "разделение"},
	"расщепленный":  {"split", "разделение"},
	"checkout":      {"инвойс", "оплата", "payment"},
	"confirm":       {"подтверждение", "otp"},
	"подтверждение": {"confirm", "otp"},
	"app":           {"application", "приложение"},
	"application":   {"app", "приложение"},
	"приложение":    {"application", "app"},
}

var topicTerms = map[string][]string{
	"token":    {"token", "токен", "auth", "authorization", "/auth"},
	"card":     {"card", "карта", "card_pan", "card_token"},
	"invoice":  {"invoice", "инвойс", "checkout", "deeplink"},
	"payment":  {"payment", "платеж", "оплата", "transaction", "транзакция"},
	"refund":   {"refund", "возврат", "revert"},
	"hold":     {"hold", "холд", "холдирование", "блокировка"},
	"callback": {"callback", "webhook", "коллбэк"},
	"payout":   {"payout", "выплата", "recipient", "получателя"},
	"split":    {"split", "расщепленный", "разделение"},
	"app":      {"application", "приложение"},
}

func detectTopics(text string) []string {
	lower := strings.ToLower(text)
	var topics []string
	for topic, terms := range topicTerms {
		for _, term := range terms {
			if strings.Contains(lower, strings.ToLower(term)) {
				topics = append(topics, topic)
				break
			}
		}
	}
	sort.Strings(topics)
	return topics
}

func sliceToSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}

func topicPenalty(topic string) float64 {
	switch topic {
	case "card", "payout", "hold":
		return 8
	case "invoice", "callback", "split", "refund":
		return 6
	case "app":
		return 4
	default:
		return 0
	}
}

func (c *corpus) search(query string, limit int) []scoredDoc {
	if limit <= 0 {
		limit = 5
	}
	rawTokens := tokenize(query)
	queryTokens := tokenizeWithSynonyms(query)
	if len(queryTokens) == 0 {
		return nil
	}
	queryLower := strings.ToLower(query)
	queryTopics := sliceToSet(detectTopics(query))
	results := make([]scoredDoc, 0, len(c.docs))
	for _, doc := range c.docs {
		score := c.scoreDoc(doc, queryTokens, rawTokens, queryLower, queryTopics)
		if score <= 0 {
			continue
		}
		results = append(results, scoredDoc{
			Doc:     doc,
			Score:   score,
			Snippet: bestSnippet(doc, queryTokens, queryLower),
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

func (c *corpus) scoreDoc(doc document, queryTokens map[string]int, rawTokens []string, queryLower string, queryTopics map[string]struct{}) float64 {
	var score float64
	docLen := float64(max(doc.TokenCount, 1))
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

	titleTokens := tokenize(doc.Title + " " + doc.Slug)
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

func bestSnippet(doc document, queryTokens map[string]int, queryLower string) string {
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

	start := max(bestIdx-3, 0)
	end := min(bestIdx+5, len(lines))
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func cleanScalarText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	s = reMultiSpace.ReplaceAllString(s, " ")
	return s
}

func (s *mcpServer) serve(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		body, err := readFrame(reader)
		if err != nil {
			return err
		}

		resp, shouldReply := s.processRequestBody(body)
		if shouldReply {
			s.writeResponse(out, resp)
		}
	}
}

func (s *mcpServer) serveHTTP(listenAddr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/mcp", s.handleHTTPMCP)
	mux.HandleFunc("/", s.handleRoot)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mux.ServeHTTP(w, r)
		fmt.Fprintf(os.Stderr, "%s %s %s\n", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return srv.ListenAndServe()
}

func (s *mcpServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":               "multicard-docs-mcp-go",
		"version":            serverVersion,
		"transport":          "http-json-rpc",
		"mcp_endpoint":       "/mcp",
		"health_endpoint":    "/healthz",
		"readiness_endpoint": "/readyz",
		"docs_loaded":        s.corpus.totalDocs,
	})
}

func (s *mcpServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *mcpServer) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "docs_loaded": s.corpus.totalDocs})
}

func (s *mcpServer) handleHTTPMCP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "failed to read request body"}})
			return
		}
		resp, shouldReply := s.processRequestBody(body)
		if !shouldReply {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "POST JSON-RPC requests to this endpoint",
			"example": map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/list",
				"params":  map[string]any{},
			},
		})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *mcpServer) processRequestBody(body []byte) (response, bool) {
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return response{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32700, Message: "parse error"},
		}, true
	}

	if req.Method == "" {
		if req.ID != nil {
			return response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32600, Message: "invalid request"},
			}, true
		}
		return response{}, false
	}

	resp, ok := s.handleRequest(req)
	if ok && req.ID != nil {
		return resp, true
	}
	return response{}, false
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}
		if match := reContentLen.FindStringSubmatch(trimmed); len(match) == 2 {
			var n int
			fmt.Sscanf(match[1], "%d", &n)
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (s *mcpServer) writeResponse(out io.Writer, resp response) {
	payload, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal response error: %v\n", err)
		return
	}
	s.outMu.Lock()
	defer s.outMu.Unlock()
	fmt.Fprintf(out, "Content-Length: %d\r\n\r\n", len(payload))
	_, _ = out.Write(payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (s *mcpServer) handleRequest(req request) (response, bool) {
	resp := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{"subscribe": false, "listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "multicard-docs-mcp-go",
				"version": serverVersion,
			},
		}
		return resp, true
	case "notifications/initialized":
		return response{}, false
	case "ping":
		resp.Result = map[string]any{}
		return resp, true
	case "tools/list":
		resp.Result = map[string]any{"tools": s.tools()}
		return resp, true
	case "tools/call":
		result, err := s.callTool(req.Params)
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			return resp, true
		}
		resp.Result = result
		return resp, true
	case "resources/list":
		resp.Result = map[string]any{"resources": s.resources()}
		return resp, true
	case "resources/read":
		result, err := s.readResource(req.Params)
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			return resp, true
		}
		resp.Result = result
		return resp, true
	case "prompts/list":
		resp.Result = map[string]any{"prompts": []any{}}
		return resp, true
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found"}
		return resp, true
	}
}

func (s *mcpServer) tools() []map[string]any {
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

func (s *mcpServer) resources() []map[string]any {
	resources := make([]map[string]any, 0, len(s.corpus.docs))
	for _, doc := range s.corpus.docs {
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

func (s *mcpServer) readResource(raw json.RawMessage) (map[string]any, error) {
	var params resourceReadParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("invalid resources/read params: %w", err)
	}
	doc, ok := s.corpus.byURI[params.URI]
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

func (s *mcpServer) callTool(raw json.RawMessage) (map[string]any, error) {
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
		limit := clamp(asInt(params.Arguments["limit"], 5), 1, 10)
		results := s.corpus.search(query, limit)
		payload := map[string]any{
			"query":   query,
			"results": marshalResults(results),
		}
		text := renderSearchText(query, results)
		return toolSuccess(text, payload), nil
	case "get_multicard_doc":
		path := strings.TrimSpace(asString(params.Arguments["path"]))
		if path == "" {
			return toolError("path is required"), nil
		}
		path = strings.TrimPrefix(path, "/")
		doc, ok := s.corpus.byPath[path]
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
		limit := clamp(asInt(params.Arguments["limit"], 4), 1, 8)
		results := s.corpus.search(question, limit)
		answer, sources := composeAnswer(question, results)
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

func composeAnswer(question string, results []scoredDoc) (string, []map[string]any) {
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
		fmt.Fprintf(&b, "- Important fields mentioned in the doc: %s\n", strings.Join(primary.RequiredFields[:min(len(primary.RequiredFields), 12)], ", "))
	}
	if len(primary.ErrorCodes) > 0 {
		fmt.Fprintf(&b, "- Related error codes in this page: %s\n", strings.Join(primary.ErrorCodes[:min(len(primary.ErrorCodes), 8)], ", "))
	}
	fmt.Fprintf(&b, "- Relevant excerpt:\n%s\n", indent(bestSnippet(results[0].Doc, tokenizeWithSynonyms(question), strings.ToLower(question)), "  "))

	if len(results) > 1 {
		fmt.Fprintf(&b, "\nOther useful pages:\n")
		for _, res := range results[1:min(len(results), 4)] {
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
			"score":    round2(res.Score),
		})
	}
	return strings.TrimSpace(b.String()), sources
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func marshalResults(results []scoredDoc) []map[string]any {
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
			"score":           round2(res.Score),
			"snippet":         res.Snippet,
		})
	}
	return out
}

func renderSearchText(query string, results []scoredDoc) string {
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
			fmt.Fprintf(&b, "   Required fields: %s\n", strings.Join(res.Doc.RequiredFields[:min(len(res.Doc.RequiredFields), 10)], ", "))
		}
		fmt.Fprintf(&b, "   Snippet:\n%s\n\n", indent(res.Snippet, "     "))
	}
	return strings.TrimSpace(b.String())
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

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

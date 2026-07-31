package docs

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	reH1         = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	reEndpoint   = regexp.MustCompile(`(?m)^\s{2}(/[^:\n]+):\s*$`)
	reMethod     = regexp.MustCompile(`(?m)^\s{4}(get|post|put|delete|patch|options|head):\s*$`)
	reSummary    = regexp.MustCompile(`(?m)^\s{6}summary:\s*(.+?)\s*$`)
	reTag        = regexp.MustCompile(`(?m)^\s{8}-\s+(.+?)\s*$`)
	reErrorCode  = regexp.MustCompile(`\bERROR_[A-Z0-9_]+\b`)
	reCodeFence  = regexp.MustCompile("```[\\s\\S]*?```")
	reMultiSpace = regexp.MustCompile(`\s+`)
)

func parseDocument(relPath, content string) Document {
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
		summary = CleanScalarText(match[1])
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
	topics := DetectTopics(title + " " + relPath + " " + summary + " " + endpointPath)
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
	tokens := TokenizeWithSynonyms(fullPlain)
	count := 0
	for _, tf := range tokens {
		count += tf
	}

	uri := "doc://multicard/" + relPath
	return Document{
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

// CleanScalarText normalizes scalar values extracted from OpenAPI-style markdown blocks.
func CleanScalarText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	s = reMultiSpace.ReplaceAllString(s, " ")
	return s
}

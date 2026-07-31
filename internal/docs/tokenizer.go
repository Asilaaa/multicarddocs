package docs

import (
	"strings"
	"unicode"
)

// TokenizeWithSynonyms tokenizes text and expands common RU/EN Multicard terms for better search recall.
func TokenizeWithSynonyms(s string) map[string]int {
	tokens := map[string]int{}
	for _, token := range Tokenize(s) {
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

// Tokenize splits natural-language text into normalized search tokens.
func Tokenize(s string) []string {
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

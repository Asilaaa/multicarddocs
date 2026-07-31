package docs

import (
	"sort"
	"strings"
)

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

// DetectTopics maps free text to high-level API domains used by search ranking.
func DetectTopics(text string) []string {
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

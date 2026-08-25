package mcp

import "testing"

func TestModernVersionFromParams(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"empty params", "", ""},
		{"no meta field", `{"query":"invoice"}`, ""},
		{"meta without version key", `{"_meta":{"other":"thing"}}`, ""},
		{"meta with version", `{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}`, "2026-07-28"},
		{"malformed json", `{not json`, ""},
		{"version key wrong type", `{"_meta":{"io.modelcontextprotocol/protocolVersion":123}}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := modernVersionFromParams([]byte(tc.raw))
			if got != tc.want {
				t.Fatalf("modernVersionFromParams(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

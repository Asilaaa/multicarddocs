package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

// VerifyPKCE checks a code_verifier against the code_challenge stored at
// /authorize time, per RFC 7636. The "plain" method exists only so the
// implementation matches the spec; MCP clients are expected to use S256.
func VerifyPKCE(verifier, challenge, method string) bool {
	if verifier == "" || challenge == "" {
		return false
	}
	switch method {
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		expected := base64.RawURLEncoding.EncodeToString(sum[:])
		return subtle.ConstantTimeCompare([]byte(expected), []byte(challenge)) == 1
	case "plain", "":
		return subtle.ConstantTimeCompare([]byte(verifier), []byte(challenge)) == 1
	default:
		return false
	}
}

// randomToken returns a cryptographically random, URL-safe token.
func randomToken(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		panic("oauth: crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

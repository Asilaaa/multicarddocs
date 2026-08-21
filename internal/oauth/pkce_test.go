package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestVerifyPKCES256(t *testing.T) {
	verifier := "test-verifier-value-1234567890"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	if !VerifyPKCE(verifier, challenge, "S256") {
		t.Fatalf("expected matching S256 verifier/challenge pair to verify")
	}
	if VerifyPKCE("wrong-verifier", challenge, "S256") {
		t.Fatalf("expected mismatched verifier to be rejected")
	}
}

func TestVerifyPKCEPlain(t *testing.T) {
	if !VerifyPKCE("abc", "abc", "plain") {
		t.Fatalf("expected equal verifier/challenge to verify under plain method")
	}
	if !VerifyPKCE("abc", "abc", "") {
		t.Fatalf("expected empty method to behave like plain")
	}
	if VerifyPKCE("abc", "xyz", "plain") {
		t.Fatalf("expected mismatched plain verifier to be rejected")
	}
}

func TestVerifyPKCEUnknownMethodRejected(t *testing.T) {
	if VerifyPKCE("abc", "abc", "md5") {
		t.Fatalf("expected unsupported code_challenge_method to be rejected")
	}
}

func TestVerifyPKCEEmptyInputsRejected(t *testing.T) {
	if VerifyPKCE("", "challenge", "S256") {
		t.Fatalf("expected empty verifier to be rejected")
	}
	if VerifyPKCE("verifier", "", "S256") {
		t.Fatalf("expected empty challenge to be rejected")
	}
}

func TestRandomTokenIsUnique(t *testing.T) {
	a := randomToken(16)
	b := randomToken(16)
	if a == "" || b == "" {
		t.Fatalf("expected non-empty tokens")
	}
	if a == b {
		t.Fatalf("expected two random tokens to differ, got the same value twice")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !constantTimeEqual("demo1234", "demo1234") {
		t.Fatalf("expected equal strings to compare equal")
	}
	if constantTimeEqual("demo1234", "wrongpass") {
		t.Fatalf("expected different strings to compare unequal")
	}
}

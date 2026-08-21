package oauth

import (
	"testing"
	"time"
)

func TestStoreClientRoundTrip(t *testing.T) {
	s := NewStore()
	c := &Client{ID: "client-1", Name: "Test Client", RedirectURIs: []string{"http://127.0.0.1/cb"}, CreatedAt: time.Now()}
	s.SaveClient(c)

	got, ok := s.GetClient("client-1")
	if !ok {
		t.Fatalf("expected saved client to be found")
	}
	if got.Name != "Test Client" {
		t.Fatalf("got client name %q, want %q", got.Name, "Test Client")
	}

	if _, ok := s.GetClient("missing"); ok {
		t.Fatalf("expected unknown client_id to be not found")
	}
}

func TestConsumeAuthCodeIsSingleUse(t *testing.T) {
	s := NewStore()
	s.SaveAuthCode(&AuthCode{Code: "code-1", ClientID: "client-1", ExpiresAt: time.Now().Add(time.Minute)})

	got, ok := s.ConsumeAuthCode("code-1")
	if !ok || got.ClientID != "client-1" {
		t.Fatalf("expected first consume to succeed and return the stored code")
	}

	if _, ok := s.ConsumeAuthCode("code-1"); ok {
		t.Fatalf("expected auth code to be consumed after first use")
	}
}

func TestConsumeAuthCodeRejectsExpired(t *testing.T) {
	s := NewStore()
	s.SaveAuthCode(&AuthCode{Code: "expired", ExpiresAt: time.Now().Add(-time.Second)})

	if _, ok := s.ConsumeAuthCode("expired"); ok {
		t.Fatalf("expected expired auth code to be rejected")
	}
	// Even though it was rejected, it should have been removed from the store.
	if _, ok := s.ConsumeAuthCode("expired"); ok {
		t.Fatalf("expected expired auth code to remain consumed on second lookup")
	}
}

func TestConsumeAuthCodeRejectsUnknown(t *testing.T) {
	s := NewStore()
	if _, ok := s.ConsumeAuthCode("does-not-exist"); ok {
		t.Fatalf("expected unknown auth code to be rejected")
	}
}

func TestTokenAccessAndRefreshLookup(t *testing.T) {
	s := NewStore()
	tok := &Token{
		AccessToken:      "at-1",
		RefreshToken:     "rt-1",
		ClientID:         "client-1",
		AccessExpiresAt:  time.Now().Add(time.Minute),
		RefreshExpiresAt: time.Now().Add(time.Hour),
	}
	s.SaveToken(tok)

	if _, ok := s.GetByAccessToken("at-1"); !ok {
		t.Fatalf("expected access token to be found")
	}
	if _, ok := s.GetByRefreshToken("rt-1"); !ok {
		t.Fatalf("expected refresh token to be found")
	}
	if _, ok := s.GetByAccessToken("unknown"); ok {
		t.Fatalf("expected unknown access token to be not found")
	}
}

func TestTokenExpiryIsEnforced(t *testing.T) {
	s := NewStore()
	tok := &Token{
		AccessToken:      "at-expired",
		RefreshToken:     "rt-expired",
		AccessExpiresAt:  time.Now().Add(-time.Minute),
		RefreshExpiresAt: time.Now().Add(-time.Minute),
	}
	s.SaveToken(tok)

	if _, ok := s.GetByAccessToken("at-expired"); ok {
		t.Fatalf("expected expired access token to be rejected")
	}
	if _, ok := s.GetByRefreshToken("rt-expired"); ok {
		t.Fatalf("expected expired refresh token to be rejected")
	}
}

func TestRevokeTokenRemovesBothSides(t *testing.T) {
	s := NewStore()
	tok := &Token{
		AccessToken:      "at-2",
		RefreshToken:     "rt-2",
		AccessExpiresAt:  time.Now().Add(time.Minute),
		RefreshExpiresAt: time.Now().Add(time.Minute),
	}
	s.SaveToken(tok)
	s.RevokeToken(tok)

	if _, ok := s.GetByAccessToken("at-2"); ok {
		t.Fatalf("expected access token to be revoked")
	}
	if _, ok := s.GetByRefreshToken("rt-2"); ok {
		t.Fatalf("expected refresh token to be revoked")
	}
}

package oauth

import (
	"sync"
	"time"
)

// Store is a simple in-memory OAuth data store. It intentionally is not
// backed by a database: this server issues short-lived demo tokens and has
// no need to survive a restart.
type Store struct {
	mu        sync.Mutex
	clients   map[string]*Client
	authCodes map[string]*AuthCode
	byAccess  map[string]*Token
	byRefresh map[string]*Token
}

// NewStore creates an empty in-memory store.
func NewStore() *Store {
	return &Store{
		clients:   make(map[string]*Client),
		authCodes: make(map[string]*AuthCode),
		byAccess:  make(map[string]*Token),
		byRefresh: make(map[string]*Token),
	}
}

func (s *Store) SaveClient(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
}

func (s *Store) GetClient(id string) (*Client, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	return c, ok
}

func (s *Store) SaveAuthCode(c *AuthCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authCodes[c.Code] = c
}

// ConsumeAuthCode returns and deletes an authorization code so it can only
// ever be exchanged once, then rejects it if it had already expired.
func (s *Store) ConsumeAuthCode(code string) (*AuthCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.authCodes[code]
	if !ok {
		return nil, false
	}
	delete(s.authCodes, code)
	if time.Now().After(c.ExpiresAt) {
		return nil, false
	}
	return c, true
}

func (s *Store) SaveToken(t *Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byAccess[t.AccessToken] = t
	s.byRefresh[t.RefreshToken] = t
}

func (s *Store) GetByAccessToken(token string) (*Token, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.byAccess[token]
	if !ok || time.Now().After(t.AccessExpiresAt) {
		return nil, false
	}
	return t, true
}

func (s *Store) GetByRefreshToken(token string) (*Token, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.byRefresh[token]
	if !ok || time.Now().After(t.RefreshExpiresAt) {
		return nil, false
	}
	return t, true
}

// RevokeToken removes a token pair, e.g. after it has been rotated by a
// refresh_token grant.
func (s *Store) RevokeToken(t *Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byAccess, t.AccessToken)
	delete(s.byRefresh, t.RefreshToken)
}

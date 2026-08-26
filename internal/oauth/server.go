package oauth

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authCodeTTL     = 60 * time.Second
	accessTokenTTL  = 1 * time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)

// Options configures the demo authorization server.
type Options struct {
	// PublicBaseURL overrides the issuer/base URL used in discovery metadata
	// and redirects, e.g. "https://multicardocs.sukoon.uz". When empty it is
	// inferred per-request from the Host and X-Forwarded-Proto headers, which
	// is convenient for local testing.
	PublicBaseURL string

	// DemoUsername/DemoPassword are the fixed credentials shown on the
	// login screen when Google sign-in isn't configured. This server has no
	// real user accounts in that mode: the login step exists to demonstrate
	// the "user authenticates and approves access" half of the OAuth flow,
	// not to gate real data.
	DemoUsername string
	DemoPassword string

	// GoogleClientID/GoogleClientSecret enable federating authentication to
	// Google (real OpenID Connect) instead of the demo login. Both must be
	// set together; when empty, /authorize falls back to the demo login so
	// local development doesn't require real Google credentials.
	GoogleClientID     string
	GoogleClientSecret string
}

// Server is a minimal OAuth 2.1 (draft-ietf-oauth-v2-1) authorization server,
// combined with the bearer-token guard that protects the MCP resource
// endpoint. OAuth 2.1 does not replace the extension RFCs below — it folds
// them in as hard requirements instead of options — so this implementation
// only ever exercises the 2.1-compliant subset of OAuth 2.0's grants:
// authorization code + PKCE, no implicit or password grants, exact
// redirect_uri matching, and refresh token rotation. It implements just
// enough of RFC 7636 (PKCE), RFC 7591 (dynamic client registration),
// RFC 8414 (authorization server metadata), and RFC 9728 (protected
// resource metadata) for a real MCP client to complete the flow.
type Server struct {
	store *Store
	opts  Options
}

// NewServer creates an authorization server with the given options.
func NewServer(opts Options) *Server {
	if opts.DemoUsername == "" {
		opts.DemoUsername = "demo"
	}
	if opts.DemoPassword == "" {
		opts.DemoPassword = "demo1234"
	}
	return &Server{store: NewStore(), opts: opts}
}

// RegisterRoutes mounts every OAuth endpoint on mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handleAuthServerMetadata)
	mux.HandleFunc("/.well-known/oauth-protected-resource", s.handleProtectedResourceMetadata)
	mux.HandleFunc("/register", s.handleRegister)
	mux.HandleFunc("/authorize", s.handleAuthorize)
	mux.HandleFunc("/token", s.handleToken)
	mux.HandleFunc("/auth/google/callback", s.handleGoogleCallback)
	mux.HandleFunc("/authorize/google/confirm", s.handleGoogleConfirm)
}

// googleEnabled reports whether Google sign-in is configured. Both the
// client ID and secret must be set; a partially-configured pair is treated
// as disabled rather than guessed at.
func (s *Server) googleEnabled() bool {
	return s.opts.GoogleClientID != "" && s.opts.GoogleClientSecret != ""
}

// RequireToken protects the MCP endpoint. A request without a valid bearer
// access token gets a 401 whose WWW-Authenticate header points MCP clients
// at the protected-resource metadata, so they can discover this
// authorization server and start the flow themselves.
func (s *Server) RequireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadataURL := s.baseURL(r) + "/.well-known/oauth-protected-resource"
		const prefix = "Bearer "

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, prefix) {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s"`, metadataURL))
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, prefix)
		if _, ok := s.store.GetByAccessToken(token); !ok {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer error="invalid_token", resource_metadata="%s"`, metadataURL))
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) baseURL(r *http.Request) string {
	if s.opts.PublicBaseURL != "" {
		return strings.TrimRight(s.opts.PublicBaseURL, "/")
	}
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// --- Discovery metadata -----------------------------------------------

func (s *Server) handleAuthServerMetadata(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/authorize",
		"token_endpoint":                        base + "/token",
		"registration_endpoint":                 base + "/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{"mcp"},
	})
}

func (s *Server) handleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 base + "/mcp",
		"authorization_servers":    []string{base},
		"bearer_methods_supported": []string{"header"},
	})
}

// --- Dynamic client registration (RFC 7591) ----------------------------

type registerRequest struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_client_metadata"})
		return
	}
	if len(req.RedirectURIs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid_redirect_uri", "error_description": "redirect_uris is required",
		})
		return
	}
	for _, u := range req.RedirectURIs {
		if _, err := url.ParseRequestURI(u); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_redirect_uri"})
			return
		}
	}

	client := &Client{
		ID:           "mcp-" + randomToken(12),
		Name:         req.ClientName,
		RedirectURIs: req.RedirectURIs,
		CreatedAt:    time.Now(),
	}
	s.store.SaveClient(client)

	// Public client: no client_secret. PKCE (required at /authorize) is
	// what protects the authorization code instead.
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ID,
		"client_id_issued_at":        client.CreatedAt.Unix(),
		"client_name":                client.Name,
		"redirect_uris":              client.RedirectURIs,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	})
}

// --- Authorization endpoint: login + consent ----------------------------

type authorizeView struct {
	ClientID            string
	ClientName          string
	RedirectURI         string
	State               string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
	Error               string
}

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.renderAuthorize(w, r, "")
	case http.MethodPost:
		s.decideAuthorize(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// loadAuthRequest validates the client_id/redirect_uri/PKCE parameters that
// must be present on both the initial GET and the consent-form POST.
func (s *Server) loadAuthRequest(form url.Values) (*Client, authorizeView, error) {
	view := authorizeView{
		ClientID:            form.Get("client_id"),
		RedirectURI:         form.Get("redirect_uri"),
		State:               form.Get("state"),
		Scope:               defaultString(form.Get("scope"), "mcp"),
		CodeChallenge:       form.Get("code_challenge"),
		CodeChallengeMethod: defaultString(form.Get("code_challenge_method"), "S256"),
	}

	client, ok := s.store.GetClient(view.ClientID)
	if !ok {
		return nil, view, fmt.Errorf("unknown client_id: register via POST /register first")
	}
	if !containsString(client.RedirectURIs, view.RedirectURI) {
		return nil, view, fmt.Errorf("redirect_uri does not match a URI registered for this client")
	}
	if view.CodeChallenge == "" {
		return nil, view, fmt.Errorf("code_challenge is required (PKCE)")
	}

	view.ClientName = client.Name
	if view.ClientName == "" {
		view.ClientName = client.ID
	}
	return client, view, nil
}

func (s *Server) renderAuthorize(w http.ResponseWriter, r *http.Request, errMsg string) {
	if rt := r.Form.Get("response_type"); rt != "" && rt != "code" {
		http.Error(w, "unsupported_response_type", http.StatusBadRequest)
		return
	}
	_, view, err := s.loadAuthRequest(r.Form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.googleEnabled() {
		s.startGoogleSignIn(w, r, view)
		return
	}

	view.Error = errMsg
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = authorizeTemplate.Execute(w, view)
}

func (s *Server) decideAuthorize(w http.ResponseWriter, r *http.Request) {
	client, view, err := s.loadAuthRequest(r.Form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	redirect, err := url.Parse(view.RedirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	q := redirect.Query()

	if r.Form.Get("decision") != "approve" {
		q.Set("error", "access_denied")
		if view.State != "" {
			q.Set("state", view.State)
		}
		redirect.RawQuery = q.Encode()
		http.Redirect(w, r, redirect.String(), http.StatusFound)
		return
	}

	username := r.Form.Get("username")
	password := r.Form.Get("password")
	if !constantTimeEqual(username, s.opts.DemoUsername) || !constantTimeEqual(password, s.opts.DemoPassword) {
		s.renderAuthorize(w, r, "Invalid username or password.")
		return
	}

	code := &AuthCode{
		Code:                randomToken(24),
		ClientID:            client.ID,
		RedirectURI:         view.RedirectURI,
		Scope:               view.Scope,
		CodeChallenge:       view.CodeChallenge,
		CodeChallengeMethod: view.CodeChallengeMethod,
		Subject:             username,
		ExpiresAt:           time.Now().Add(authCodeTTL),
	}
	s.store.SaveAuthCode(code)

	q.Set("code", code.Code)
	if view.State != "" {
		q.Set("state", view.State)
	}
	redirect.RawQuery = q.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

// --- Token endpoint ------------------------------------------------------

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_request"})
		return
	}

	w.Header().Set("Cache-Control", "no-store")

	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		s.exchangeAuthCode(w, r)
	case "refresh_token":
		s.exchangeRefreshToken(w, r)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unsupported_grant_type"})
	}
}

func (s *Server) exchangeAuthCode(w http.ResponseWriter, r *http.Request) {
	code := r.PostForm.Get("code")
	verifier := r.PostForm.Get("code_verifier")
	clientID := r.PostForm.Get("client_id")
	redirectURI := r.PostForm.Get("redirect_uri")

	ac, ok := s.store.ConsumeAuthCode(code)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
		return
	}
	if ac.ClientID != clientID || ac.RedirectURI != redirectURI {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid_grant", "error_description": "client_id or redirect_uri mismatch",
		})
		return
	}
	if !VerifyPKCE(verifier, ac.CodeChallenge, ac.CodeChallengeMethod) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid_grant", "error_description": "PKCE verification failed",
		})
		return
	}

	tok := s.issueToken(ac.ClientID, ac.Subject, ac.Scope)
	writeJSON(w, http.StatusOK, tokenResponse(tok))
}

func (s *Server) exchangeRefreshToken(w http.ResponseWriter, r *http.Request) {
	refresh := r.PostForm.Get("refresh_token")
	clientID := r.PostForm.Get("client_id")

	old, ok := s.store.GetByRefreshToken(refresh)
	if !ok || old.ClientID != clientID {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
		return
	}

	// Rotate: the old refresh token is single-use.
	s.store.RevokeToken(old)
	tok := s.issueToken(old.ClientID, old.Subject, old.Scope)
	writeJSON(w, http.StatusOK, tokenResponse(tok))
}

func (s *Server) issueToken(clientID, subject, scope string) *Token {
	now := time.Now()
	tok := &Token{
		AccessToken:      randomToken(32),
		RefreshToken:     randomToken(32),
		ClientID:         clientID,
		Subject:          subject,
		Scope:            scope,
		AccessExpiresAt:  now.Add(accessTokenTTL),
		RefreshExpiresAt: now.Add(refreshTokenTTL),
	}
	s.store.SaveToken(tok)
	return tok
}

func tokenResponse(t *Token) map[string]any {
	return map[string]any{
		"access_token":  t.AccessToken,
		"refresh_token": t.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(accessTokenTTL.Seconds()),
		"scope":         t.Scope,
	}
}

// --- small helpers ---------------------------------------------------------

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

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

var authorizeTemplate = template.Must(template.New("authorize").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>Authorize MulticardDocs MCP</title>
<style>
 body{font-family:system-ui,-apple-system,sans-serif;max-width:420px;margin:64px auto;color:#1a1a1a;padding:0 16px}
 .card{border:1px solid #ddd;border-radius:12px;padding:28px}
 h1{font-size:18px;margin:0 0 4px}
 p.sub{color:#666;font-size:14px;margin-top:0}
 label{display:block;font-size:13px;margin:14px 0 4px;font-weight:600}
 input{width:100%;padding:9px 10px;border:1px solid #ccc;border-radius:6px;box-sizing:border-box;font-size:14px}
 .actions{display:flex;gap:8px;margin-top:22px}
 button{flex:1;padding:10px;border-radius:6px;border:1px solid #ccc;font-size:14px;cursor:pointer;background:#fff}
 button.approve{background:#111;color:#fff;border-color:#111}
 .scope{background:#f5f5f5;border-radius:8px;padding:10px;font-size:13px;margin:14px 0}
 .err{color:#b00020;font-size:13px;margin:6px 0 0}
</style></head>
<body>
<div class="card">
 <h1>{{.ClientName}}</h1>
 <p class="sub">wants to access MulticardDocs MCP on your behalf</p>
 <div class="scope">Requested scope: <code>{{.Scope}}</code></div>
 {{if .Error}}<p class="err">{{.Error}}</p>{{end}}
 <form method="POST" action="/authorize">
  <input type="hidden" name="client_id" value="{{.ClientID}}">
  <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
  <input type="hidden" name="state" value="{{.State}}">
  <input type="hidden" name="scope" value="{{.Scope}}">
  <input type="hidden" name="code_challenge" value="{{.CodeChallenge}}">
  <input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}">
  <label for="username">Username</label>
  <input id="username" name="username" autocomplete="username" required>
  <label for="password">Password</label>
  <input id="password" name="password" type="password" autocomplete="current-password" required>
  <div class="actions">
   <button type="submit" name="decision" value="deny">Deny</button>
   <button type="submit" name="decision" value="approve" class="approve">Approve</button>
  </div>
 </form>
</div>
</body></html>`))

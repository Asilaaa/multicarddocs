package oauth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	auth := NewServer(Options{})
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return auth, ts
}

func pkcePair() (verifier, challenge string) {
	verifier = randomToken(32)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

func registerClient(t *testing.T, baseURL, redirectURI string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"client_name":   "Test Client",
		"redirect_uris": []string{redirectURI},
	})
	if err != nil {
		t.Fatalf("marshal register request: %v", err)
	}
	resp, err := http.Post(baseURL+"/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /register status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	var out struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if out.ClientID == "" {
		t.Fatalf("expected non-empty client_id")
	}
	return out.ClientID
}

// noFollowClient never follows redirects, so callers can inspect the
// Location header of the /authorize response directly.
func noFollowClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func approveAuthorize(t *testing.T, baseURL, clientID, redirectURI, challenge, state string) *http.Response {
	t.Helper()
	form := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"scope":                 {"mcp"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"username":              {"demo"},
		"password":              {"demo1234"},
		"decision":              {"approve"},
	}
	resp, err := noFollowClient().PostForm(baseURL+"/authorize", form)
	if err != nil {
		t.Fatalf("POST /authorize: %v", err)
	}
	return resp
}

func exchangeCode(t *testing.T, baseURL, code, clientID, redirectURI, verifier string) (map[string]any, int) {
	t.Helper()
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"code_verifier": {verifier},
	}
	resp, err := http.PostForm(baseURL+"/token", form)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	return out, resp.StatusCode
}

// --- registration ---------------------------------------------------------

func TestRegisterRequiresRedirectURIs(t *testing.T) {
	_, ts := newTestServer(t)
	body, _ := json.Marshal(map[string]any{"client_name": "No Redirects"})
	resp, err := http.Post(ts.URL+"/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestRegisterRejectsInvalidRedirectURI(t *testing.T) {
	_, ts := newTestServer(t)
	body, _ := json.Marshal(map[string]any{
		"client_name":   "Bad Redirect",
		"redirect_uris": []string{"://not-a-valid-uri"},
	})
	resp, err := http.Post(ts.URL+"/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestRegisterIssuesPublicClient(t *testing.T) {
	_, ts := newTestServer(t)
	body, _ := json.Marshal(map[string]any{
		"client_name":   "Demo Client",
		"redirect_uris": []string{"http://127.0.0.1:9999/callback"},
	})
	resp, err := http.Post(ts.URL+"/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["token_endpoint_auth_method"] != "none" {
		t.Fatalf("expected public client (token_endpoint_auth_method=none), got %v", out["token_endpoint_auth_method"])
	}
	if _, hasSecret := out["client_secret"]; hasSecret {
		t.Fatalf("expected no client_secret to be issued for a public client")
	}
}

// --- authorize: validation -------------------------------------------------

func TestAuthorizeUnknownClientRejected(t *testing.T) {
	_, ts := newTestServer(t)
	_, challenge := pkcePair()
	resp, err := http.Get(ts.URL + "/authorize?" + url.Values{
		"client_id":      {"does-not-exist"},
		"redirect_uri":   {"http://127.0.0.1:9999/callback"},
		"code_challenge": {challenge},
	}.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestAuthorizeRedirectURIMismatchRejected(t *testing.T) {
	_, ts := newTestServer(t)
	clientID := registerClient(t, ts.URL, "http://127.0.0.1:9999/callback")
	_, challenge := pkcePair()

	resp, err := http.Get(ts.URL + "/authorize?" + url.Values{
		"client_id":      {clientID},
		"redirect_uri":   {"http://evil.example.com/callback"},
		"code_challenge": {challenge},
	}.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestAuthorizeMissingCodeChallengeRejected(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)

	resp, err := http.Get(ts.URL + "/authorize?" + url.Values{
		"client_id":    {clientID},
		"redirect_uri": {redirectURI},
	}.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestAuthorizeRendersConsentScreenWithClientName(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	_, challenge := pkcePair()

	resp, err := http.Get(ts.URL + "/authorize?" + url.Values{
		"client_id":      {clientID},
		"redirect_uri":   {redirectURI},
		"code_challenge": {challenge},
	}.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Test Client") {
		t.Fatalf("expected consent page to include the registered client name")
	}
}

// --- authorize: decision -----------------------------------------------

func TestAuthorizeDenyRedirectsWithError(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	_, challenge := pkcePair()

	form := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"state":                 {"xyz"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"decision":              {"deny"},
	}
	resp, err := noFollowClient().PostForm(ts.URL+"/authorize", form)
	if err != nil {
		t.Fatalf("POST /authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if got := loc.Query().Get("error"); got != "access_denied" {
		t.Fatalf("error = %q, want %q", got, "access_denied")
	}
	if got := loc.Query().Get("state"); got != "xyz" {
		t.Fatalf("state = %q, want %q", got, "xyz")
	}
}

func TestAuthorizeWrongCredentialsReRendersForm(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	_, challenge := pkcePair()

	form := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"username":              {"demo"},
		"password":              {"wrong-password"},
		"decision":              {"approve"},
	}
	resp, err := noFollowClient().PostForm(ts.URL+"/authorize", form)
	if err != nil {
		t.Fatalf("POST /authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (form re-render, not a redirect)", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Invalid username or password") {
		t.Fatalf("expected re-rendered form to show a login error")
	}
}

func TestAuthorizeApproveRedirectsWithCode(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	_, challenge := pkcePair()

	resp := approveAuthorize(t, ts.URL, clientID, redirectURI, challenge, "state-abc")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if loc.Query().Get("code") == "" {
		t.Fatalf("expected a code param in the redirect")
	}
	if got := loc.Query().Get("state"); got != "state-abc" {
		t.Fatalf("state = %q, want %q", got, "state-abc")
	}
}

// --- token exchange -----------------------------------------------------

func TestTokenExchangeHappyPath(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	verifier, challenge := pkcePair()

	resp := approveAuthorize(t, ts.URL, clientID, redirectURI, challenge, "s")
	loc, _ := url.Parse(resp.Header.Get("Location"))
	resp.Body.Close()
	code := loc.Query().Get("code")

	out, status := exchangeCode(t, ts.URL, code, clientID, redirectURI, verifier)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %+v", status, http.StatusOK, out)
	}
	if out["access_token"] == "" || out["access_token"] == nil {
		t.Fatalf("expected non-empty access_token, got %+v", out)
	}
	if out["refresh_token"] == "" || out["refresh_token"] == nil {
		t.Fatalf("expected non-empty refresh_token, got %+v", out)
	}
	if out["token_type"] != "Bearer" {
		t.Fatalf("token_type = %v, want Bearer", out["token_type"])
	}
}

func TestTokenExchangeWrongVerifierRejected(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	_, challenge := pkcePair()

	resp := approveAuthorize(t, ts.URL, clientID, redirectURI, challenge, "s")
	loc, _ := url.Parse(resp.Header.Get("Location"))
	resp.Body.Close()
	code := loc.Query().Get("code")

	out, status := exchangeCode(t, ts.URL, code, clientID, redirectURI, "not-the-real-verifier")
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %+v", status, http.StatusBadRequest, out)
	}
	if out["error"] != "invalid_grant" {
		t.Fatalf("error = %v, want invalid_grant", out["error"])
	}
}

func TestTokenExchangeRedirectURIMismatchRejected(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	verifier, challenge := pkcePair()

	resp := approveAuthorize(t, ts.URL, clientID, redirectURI, challenge, "s")
	loc, _ := url.Parse(resp.Header.Get("Location"))
	resp.Body.Close()
	code := loc.Query().Get("code")

	out, status := exchangeCode(t, ts.URL, code, clientID, "http://different.example.com/callback", verifier)
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %+v", status, http.StatusBadRequest, out)
	}
}

func TestTokenExchangeCodeIsSingleUse(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	verifier, challenge := pkcePair()

	resp := approveAuthorize(t, ts.URL, clientID, redirectURI, challenge, "s")
	loc, _ := url.Parse(resp.Header.Get("Location"))
	resp.Body.Close()
	code := loc.Query().Get("code")

	if _, status := exchangeCode(t, ts.URL, code, clientID, redirectURI, verifier); status != http.StatusOK {
		t.Fatalf("first exchange status = %d, want %d", status, http.StatusOK)
	}
	out, status := exchangeCode(t, ts.URL, code, clientID, redirectURI, verifier)
	if status != http.StatusBadRequest {
		t.Fatalf("second exchange status = %d, want %d, body = %+v", status, http.StatusBadRequest, out)
	}
}

func TestTokenUnsupportedGrantTypeRejected(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := http.PostForm(ts.URL+"/token", url.Values{"grant_type": {"password"}})
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	_, ts := newTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	verifier, challenge := pkcePair()

	resp := approveAuthorize(t, ts.URL, clientID, redirectURI, challenge, "s")
	loc, _ := url.Parse(resp.Header.Get("Location"))
	resp.Body.Close()
	code := loc.Query().Get("code")

	first, status := exchangeCode(t, ts.URL, code, clientID, redirectURI, verifier)
	if status != http.StatusOK {
		t.Fatalf("initial exchange status = %d, want %d", status, http.StatusOK)
	}
	oldRefresh := first["refresh_token"].(string)
	oldAccess := first["access_token"].(string)

	refreshResp, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {oldRefresh},
		"client_id":     {clientID},
	})
	if err != nil {
		t.Fatalf("POST /token (refresh): %v", err)
	}
	defer refreshResp.Body.Close()
	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("refresh status = %d, want %d", refreshResp.StatusCode, http.StatusOK)
	}
	var second map[string]any
	if err := json.NewDecoder(refreshResp.Body).Decode(&second); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if second["access_token"] == oldAccess {
		t.Fatalf("expected a new access token after refresh")
	}
	if second["refresh_token"] == oldRefresh {
		t.Fatalf("expected the refresh token to rotate")
	}

	// The old refresh token must now be dead (single use / rotated away).
	reuseResp, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {oldRefresh},
		"client_id":     {clientID},
	})
	if err != nil {
		t.Fatalf("POST /token (reuse old refresh token): %v", err)
	}
	defer reuseResp.Body.Close()
	if reuseResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("reused refresh token status = %d, want %d", reuseResp.StatusCode, http.StatusBadRequest)
	}
}

// --- discovery metadata ---------------------------------------------------

func TestDiscoveryMetadataUsesPublicBaseURLOverride(t *testing.T) {
	auth := NewServer(Options{PublicBaseURL: "https://multicardocs.sukoon.uz/"})
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["issuer"] != "https://multicardocs.sukoon.uz" {
		t.Fatalf("issuer = %v, want trailing slash trimmed", out["issuer"])
	}
}

func TestBaseURLInfersFromRequestWhenNoOverride(t *testing.T) {
	auth := NewServer(Options{})
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Host = "example.internal"
	req.Header.Set("X-Forwarded-Proto", "https")

	if got := auth.baseURL(req); got != "https://example.internal" {
		t.Fatalf("baseURL() = %q, want %q", got, "https://example.internal")
	}
}

func TestNewServerDefaultsDemoCredentials(t *testing.T) {
	auth := NewServer(Options{})
	if auth.opts.DemoUsername != "demo" || auth.opts.DemoPassword != "demo1234" {
		t.Fatalf("expected default demo credentials, got %q/%q", auth.opts.DemoUsername, auth.opts.DemoPassword)
	}
}

// --- RequireToken middleware ----------------------------------------------

func TestRequireTokenRejectsMissingBearer(t *testing.T) {
	auth := NewServer(Options{PublicBaseURL: "https://example.com"})
	called := false
	h := auth.RequireToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatalf("expected the protected handler not to be called")
	}
	want := `Bearer resource_metadata="https://example.com/.well-known/oauth-protected-resource"`
	if got := rec.Header().Get("WWW-Authenticate"); got != want {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, want)
	}
}

func TestRequireTokenRejectsInvalidToken(t *testing.T) {
	auth := NewServer(Options{PublicBaseURL: "https://example.com"})
	called := false
	h := auth.RequireToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer garbage-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatalf("expected the protected handler not to be called")
	}
	if !strings.Contains(rec.Header().Get("WWW-Authenticate"), `error="invalid_token"`) {
		t.Fatalf("WWW-Authenticate = %q, expected it to mention invalid_token", rec.Header().Get("WWW-Authenticate"))
	}
}

func TestRequireTokenAllowsValidToken(t *testing.T) {
	auth := NewServer(Options{})
	tok := auth.issueToken("client-1", "demo", "mcp")

	called := false
	h := auth.RequireToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected the protected handler to be called with a valid token")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

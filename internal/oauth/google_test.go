package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fakeGoogle stands in for Google's token + userinfo endpoints, and rewrites
// the package-level endpoint vars for the duration of the test.
type fakeGoogle struct {
	ts             *httptest.Server
	email          string
	sub            string
	emailVerified  bool
	tokenStatus    int
	userinfoStatus int
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	t.Helper()
	fg := &fakeGoogle{email: "person@example.com", sub: "1234567890", emailVerified: true, tokenStatus: http.StatusOK, userinfoStatus: http.StatusOK}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if fg.tokenStatus != http.StatusOK {
			w.WriteHeader(fg.tokenStatus)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "fake-google-access-token", "token_type": "Bearer"})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if fg.userinfoStatus != http.StatusOK {
			w.WriteHeader(fg.userinfoStatus)
			_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            fg.sub,
			"email":          fg.email,
			"email_verified": fg.emailVerified,
		})
	})
	fg.ts = httptest.NewServer(mux)
	t.Cleanup(fg.ts.Close)

	origToken, origUserinfo := googleTokenEndpoint, googleUserinfoEndpoint
	googleTokenEndpoint = fg.ts.URL + "/token"
	googleUserinfoEndpoint = fg.ts.URL + "/userinfo"
	t.Cleanup(func() {
		googleTokenEndpoint = origToken
		googleUserinfoEndpoint = origUserinfo
	})
	return fg
}

func newGoogleTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	auth := NewServer(Options{GoogleClientID: "test-google-client-id", GoogleClientSecret: "test-google-client-secret"})
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return auth, ts
}

func TestGoogleEnabledRequiresBothClientIDAndSecret(t *testing.T) {
	cases := []struct {
		name   string
		id     string
		secret string
		want   bool
	}{
		{"neither", "", "", false},
		{"id only", "id", "", false},
		{"secret only", "", "secret", false},
		{"both", "id", "secret", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewServer(Options{GoogleClientID: tc.id, GoogleClientSecret: tc.secret})
			if got := s.googleEnabled(); got != tc.want {
				t.Fatalf("googleEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAuthorizeRedirectsToGoogleWhenEnabled(t *testing.T) {
	auth, ts := newGoogleTestServer(t)
	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	_, challenge := pkcePair()

	resp, err := noFollowClient().Get(ts.URL + "/authorize?" + url.Values{
		"response_type":  {"code"},
		"client_id":      {clientID},
		"redirect_uri":   {redirectURI},
		"code_challenge": {challenge},
		"state":          {"client-state"},
	}.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, googleAuthEndpoint) {
		t.Fatalf("Location = %q, want it to start with the Google auth endpoint %q", loc, googleAuthEndpoint)
	}
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	q := parsed.Query()
	if q.Get("client_id") != "test-google-client-id" {
		t.Fatalf("client_id = %q, want our Google client id", q.Get("client_id"))
	}
	if q.Get("scope") != "openid email profile" {
		t.Fatalf("scope = %q, want openid email profile", q.Get("scope"))
	}
	pendingID := q.Get("state")
	if pendingID == "" {
		t.Fatalf("expected a non-empty state (pending authorization id)")
	}
	if _, ok := auth.store.GetPendingAuthorization(pendingID); !ok {
		t.Fatalf("expected a PendingAuthorization to be stored for id %q", pendingID)
	}
}

func TestGoogleCallbackUnknownStateRejected(t *testing.T) {
	newFakeGoogle(t)
	_, ts := newGoogleTestServer(t)
	resp, err := http.Get(ts.URL + "/auth/google/callback?code=abc&state=does-not-exist")
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGoogleCallbackPropagatesGoogleError(t *testing.T) {
	newFakeGoogle(t)
	_, ts := newGoogleTestServer(t)
	resp, err := http.Get(ts.URL + "/auth/google/callback?error=access_denied&state=whatever")
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// beginGoogleFlow drives register -> /authorize (capturing the pending id
// Google would echo back) so tests can jump straight to exercising the
// callback and confirm steps.
func beginGoogleFlow(t *testing.T, ts *httptest.Server) (clientID, redirectURI, pendingID string) {
	t.Helper()
	redirectURI = "http://127.0.0.1:9999/callback"
	clientID = registerClient(t, ts.URL, redirectURI)
	_, challenge := pkcePair()

	resp, err := noFollowClient().Get(ts.URL + "/authorize?" + url.Values{
		"response_type":  {"code"},
		"client_id":      {clientID},
		"redirect_uri":   {redirectURI},
		"code_challenge": {challenge},
		"state":          {"client-state"},
	}.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	defer resp.Body.Close()
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	pendingID = loc.Query().Get("state")
	return clientID, redirectURI, pendingID
}

func TestGoogleCallbackShowsConsentWithEmail(t *testing.T) {
	fg := newFakeGoogle(t)
	fg.email = "presenter@example.com"
	_, ts := newGoogleTestServer(t)
	_, _, pendingID := beginGoogleFlow(t, ts)

	resp, err := http.Get(ts.URL + "/auth/google/callback?code=fake-code&state=" + pendingID)
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body := readAll(t, resp)
	if !strings.Contains(body, "presenter@example.com") {
		t.Fatalf("expected consent page to show the verified email, got: %s", body)
	}
}

func TestGoogleCallbackRejectsUnverifiedEmail(t *testing.T) {
	fg := newFakeGoogle(t)
	fg.emailVerified = false
	_, ts := newGoogleTestServer(t)
	_, _, pendingID := beginGoogleFlow(t, ts)

	resp, err := http.Get(ts.URL + "/auth/google/callback?code=fake-code&state=" + pendingID)
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
}

func TestGoogleCallbackHandlesTokenEndpointFailure(t *testing.T) {
	fg := newFakeGoogle(t)
	fg.tokenStatus = http.StatusUnauthorized
	_, ts := newGoogleTestServer(t)
	_, _, pendingID := beginGoogleFlow(t, ts)

	resp, err := http.Get(ts.URL + "/auth/google/callback?code=fake-code&state=" + pendingID)
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
}

func TestGoogleConfirmBeforeCallbackRejected(t *testing.T) {
	newFakeGoogle(t)
	_, ts := newGoogleTestServer(t)
	_, _, pendingID := beginGoogleFlow(t, ts)

	// Skip the callback step entirely: pending exists but has no verified email yet.
	resp, err := noFollowClient().PostForm(ts.URL+"/authorize/google/confirm", url.Values{
		"pending_id": {pendingID},
		"decision":   {"approve"},
	})
	if err != nil {
		t.Fatalf("POST confirm: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGoogleConfirmDenyRedirectsWithError(t *testing.T) {
	fg := newFakeGoogle(t)
	fg.email = "denier@example.com"
	_, ts := newGoogleTestServer(t)
	_, redirectURI, pendingID := beginGoogleFlow(t, ts)

	cbResp, err := http.Get(ts.URL + "/auth/google/callback?code=fake-code&state=" + pendingID)
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	cbResp.Body.Close()

	resp, err := noFollowClient().PostForm(ts.URL+"/authorize/google/confirm", url.Values{
		"pending_id": {pendingID},
		"decision":   {"deny"},
	})
	if err != nil {
		t.Fatalf("POST confirm: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	loc, _ := url.Parse(resp.Header.Get("Location"))
	if !strings.HasPrefix(loc.String(), redirectURI) {
		t.Fatalf("Location = %q, want prefix %q", loc.String(), redirectURI)
	}
	if got := loc.Query().Get("error"); got != "access_denied" {
		t.Fatalf("error = %q, want access_denied", got)
	}
}

func TestGoogleFullFlowIssuesTokenWithGoogleEmailAsSubject(t *testing.T) {
	fg := newFakeGoogle(t)
	fg.email = "full-flow@example.com"
	fg.sub = "google-sub-42"
	authServer, ts := newGoogleTestServer(t)

	redirectURI := "http://127.0.0.1:9999/callback"
	clientID := registerClient(t, ts.URL, redirectURI)
	verifier, challenge := pkcePair()

	authResp, err := noFollowClient().Get(ts.URL + "/authorize?" + url.Values{
		"response_type":  {"code"},
		"client_id":      {clientID},
		"redirect_uri":   {redirectURI},
		"code_challenge": {challenge},
		"state":          {"client-state"},
	}.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	authLoc, _ := url.Parse(authResp.Header.Get("Location"))
	authResp.Body.Close()
	pendingID := authLoc.Query().Get("state")

	cbResp, err := http.Get(ts.URL + "/auth/google/callback?code=fake-code&state=" + pendingID)
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	cbResp.Body.Close()

	confirmResp, err := noFollowClient().PostForm(ts.URL+"/authorize/google/confirm", url.Values{
		"pending_id": {pendingID},
		"decision":   {"approve"},
	})
	if err != nil {
		t.Fatalf("POST confirm: %v", err)
	}
	confirmLoc, _ := url.Parse(confirmResp.Header.Get("Location"))
	confirmResp.Body.Close()
	if confirmLoc.Query().Get("state") != "client-state" {
		t.Fatalf("expected the original MCP client's state to survive the round trip, got %q", confirmLoc.Query().Get("state"))
	}
	authCode := confirmLoc.Query().Get("code")
	if authCode == "" {
		t.Fatalf("expected an authorization code in the redirect")
	}

	tokOut, status := exchangeCode(t, ts.URL, authCode, clientID, redirectURI, verifier)
	if status != http.StatusOK {
		t.Fatalf("token exchange status = %d, want %d, body = %+v", status, http.StatusOK, tokOut)
	}
	accessToken, _ := tokOut["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("expected an access_token")
	}
	tok, ok := authServer.store.GetByAccessToken(accessToken)
	if !ok {
		t.Fatalf("issued access token not found in store")
	}
	if tok.Subject != "full-flow@example.com" {
		t.Fatalf("token Subject = %q, want the verified Google email", tok.Subject)
	}
}

func readAll(t *testing.T, resp *http.Response) string {
	t.Helper()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return sb.String()
}

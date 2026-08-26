package oauth

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Google's OIDC endpoints. Declared as vars, not consts, so tests can point
// them at an httptest.Server instead of the real Google infrastructure.
var (
	googleAuthEndpoint     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint    = "https://oauth2.googleapis.com/token"
	googleUserinfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"
)

const pendingAuthTTL = 10 * time.Minute

// startGoogleSignIn stashes the MCP client's authorization request and
// redirects the browser to Google's real login instead of rendering the
// local demo form.
func (s *Server) startGoogleSignIn(w http.ResponseWriter, r *http.Request, view authorizeView) {
	pending := &PendingAuthorization{
		ID:                  randomToken(24),
		ClientID:            view.ClientID,
		RedirectURI:         view.RedirectURI,
		State:               view.State,
		Scope:               view.Scope,
		CodeChallenge:       view.CodeChallenge,
		CodeChallengeMethod: view.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(pendingAuthTTL),
	}
	s.store.SavePendingAuthorization(pending)

	q := url.Values{
		"client_id":     {s.opts.GoogleClientID},
		"redirect_uri":  {s.googleRedirectURI(r)},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {pending.ID},
	}
	http.Redirect(w, r, googleAuthEndpoint+"?"+q.Encode(), http.StatusFound)
}

func (s *Server) googleRedirectURI(r *http.Request) string {
	return s.baseURL(r) + "/auth/google/callback"
}

// handleGoogleCallback receives Google's redirect after the user signs in
// there, exchanges the code for their verified identity, and shows this
// server's own consent screen — Google's screen only covers "share your
// profile with this app," not "let this specific MCP client use these
// tools," so that decision still has to happen here.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errParam := q.Get("error"); errParam != "" {
		http.Error(w, "google sign-in failed: "+errParam, http.StatusBadRequest)
		return
	}
	code := q.Get("code")
	pendingID := q.Get("state")
	if code == "" || pendingID == "" {
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	pending, ok := s.store.GetPendingAuthorization(pendingID)
	if !ok {
		http.Error(w, "unknown or expired sign-in attempt, please retry", http.StatusBadRequest)
		return
	}

	email, subject, err := s.exchangeGoogleCode(code, s.googleRedirectURI(r))
	if err != nil {
		http.Error(w, "google sign-in failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	// The verified identity is stored server-side, keyed by the opaque
	// pending ID — never trust an email echoed back from a client-supplied
	// form field on the confirm step below.
	pending.GoogleEmail = email
	pending.GoogleSubject = subject
	s.store.SavePendingAuthorization(pending)

	clientName := pending.ClientID
	if client, ok := s.store.GetClient(pending.ClientID); ok && client.Name != "" {
		clientName = client.Name
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = googleConsentTemplate.Execute(w, googleConsentView{
		PendingID:  pending.ID,
		Email:      email,
		ClientName: clientName,
		Scope:      pending.Scope,
	})
}

// exchangeGoogleCode swaps a Google authorization code for the signed-in
// user's verified email, via Google's token endpoint and then its userinfo
// endpoint. Using userinfo (a direct, TLS-protected call to Google) rather
// than locally verifying the ID token's JWT signature keeps this package
// dependency-free; a hardened production version would verify the ID token
// against Google's JWKS instead.
func (s *Server) exchangeGoogleCode(code, redirectURI string) (email, subject string, err error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {s.opts.GoogleClientID},
		"client_secret": {s.opts.GoogleClientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}
	resp, err := http.PostForm(googleTokenEndpoint, form)
	if err != nil {
		return "", "", fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", "", fmt.Errorf("token exchange returned %d: %s", resp.StatusCode, string(body))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", "", fmt.Errorf("decode token response: %w", err)
	}
	if tok.AccessToken == "" {
		return "", "", fmt.Errorf("token response had no access_token")
	}

	req, err := http.NewRequest(http.MethodGet, googleUserinfoEndpoint, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	userResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("userinfo request failed: %w", err)
	}
	defer userResp.Body.Close()
	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(userResp.Body, 4096))
		return "", "", fmt.Errorf("userinfo returned %d: %s", userResp.StatusCode, string(body))
	}
	var profile struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&profile); err != nil {
		return "", "", fmt.Errorf("decode userinfo response: %w", err)
	}
	if !profile.EmailVerified {
		return "", "", fmt.Errorf("google account email is not verified")
	}
	if profile.Email == "" || profile.Sub == "" {
		return "", "", fmt.Errorf("userinfo response missing email or sub")
	}
	return profile.Email, profile.Sub, nil
}

// handleGoogleConfirm is the final approve/deny step, shown after Google
// sign-in succeeds. It mints the authorization code using the server-side
// verified identity, never anything from the submitted form.
func (s *Server) handleGoogleConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	pending, ok := s.store.ConsumePendingAuthorization(r.Form.Get("pending_id"))
	if !ok {
		http.Error(w, "unknown or expired sign-in attempt, please retry", http.StatusBadRequest)
		return
	}
	if pending.GoogleEmail == "" {
		http.Error(w, "sign-in was not completed", http.StatusBadRequest)
		return
	}

	redirect, err := url.Parse(pending.RedirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	q := redirect.Query()

	if r.Form.Get("decision") != "approve" {
		q.Set("error", "access_denied")
		if pending.State != "" {
			q.Set("state", pending.State)
		}
		redirect.RawQuery = q.Encode()
		http.Redirect(w, r, redirect.String(), http.StatusFound)
		return
	}

	code := &AuthCode{
		Code:                randomToken(24),
		ClientID:            pending.ClientID,
		RedirectURI:         pending.RedirectURI,
		Scope:               pending.Scope,
		CodeChallenge:       pending.CodeChallenge,
		CodeChallengeMethod: pending.CodeChallengeMethod,
		Subject:             pending.GoogleEmail,
		ExpiresAt:           time.Now().Add(authCodeTTL),
	}
	s.store.SaveAuthCode(code)

	q.Set("code", code.Code)
	if pending.State != "" {
		q.Set("state", pending.State)
	}
	redirect.RawQuery = q.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

type googleConsentView struct {
	PendingID  string
	Email      string
	ClientName string
	Scope      string
}

var googleConsentTemplate = template.Must(template.New("google-consent").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>Authorize MulticardDocs MCP</title>
<style>
 body{font-family:system-ui,-apple-system,sans-serif;max-width:420px;margin:64px auto;color:#1a1a1a;padding:0 16px}
 .card{border:1px solid #ddd;border-radius:12px;padding:28px}
 h1{font-size:18px;margin:0 0 4px}
 p.sub{color:#666;font-size:14px;margin-top:0}
 .identity{display:flex;align-items:center;gap:10px;background:#f5f5f5;border-radius:8px;padding:12px;font-size:14px;margin:14px 0}
 .identity .dot{width:8px;height:8px;border-radius:50%;background:#1a73e8;flex:none}
 .scope{color:#666;font-size:13px;margin:0 0 14px}
 .actions{display:flex;gap:8px;margin-top:8px}
 button{flex:1;padding:10px;border-radius:6px;border:1px solid #ccc;font-size:14px;cursor:pointer;background:#fff}
 button.approve{background:#111;color:#fff;border-color:#111}
</style></head>
<body>
<div class="card">
 <h1>{{.ClientName}}</h1>
 <p class="sub">wants to access MulticardDocs MCP as</p>
 <div class="identity"><span class="dot"></span><strong>{{.Email}}</strong></div>
 <p class="scope">Requested scope: <code>{{.Scope}}</code></p>
 <form method="POST" action="/authorize/google/confirm">
  <input type="hidden" name="pending_id" value="{{.PendingID}}">
  <div class="actions">
   <button type="submit" name="decision" value="deny">Deny</button>
   <button type="submit" name="decision" value="approve" class="approve">Approve</button>
  </div>
 </form>
</div>
</body></html>`))

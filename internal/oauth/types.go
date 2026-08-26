// Package oauth implements a minimal, self-contained OAuth 2.1 authorization
// server plus resource-server guard for the MCP HTTP transport.
//
// It exists to demonstrate the flow MCP clients use against a remote,
// HTTP-based server: dynamic client registration (RFC 7591), authorization
// code + PKCE (RFC 7636), discovery metadata (RFC 8414 / RFC 9728), and
// bearer-token protection of the /mcp endpoint. There is no real user
// database — a single fixed demo login stands in for "the user signs in and
// approves access".
package oauth

import "time"

// Client is an OAuth client registered through the dynamic registration
// endpoint. MCP clients are treated as public clients: no client_secret is
// issued, and PKCE is required on every authorization request instead.
type Client struct {
	ID           string
	Name         string
	RedirectURIs []string
	CreatedAt    time.Time
}

// AuthCode is a short-lived, single-use authorization code minted after the
// user approves the consent screen.
type AuthCode struct {
	Code                string
	ClientID            string
	RedirectURI         string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
	Subject             string
	ExpiresAt           time.Time
}

// Token is an issued access/refresh token pair.
type Token struct {
	AccessToken      string
	RefreshToken     string
	ClientID         string
	Subject          string
	Scope            string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// PendingAuthorization is an MCP client's /authorize request, stashed
// server-side while the user completes sign-in with Google. GoogleEmail and
// GoogleSubject are populated only after the Google callback verifies them —
// they are never taken from client-supplied form data, since a consent-page
// form field could be tampered with by anyone who can reach the browser.
type PendingAuthorization struct {
	ID                  string
	ClientID            string
	RedirectURI         string
	State               string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
	GoogleEmail         string
	GoogleSubject       string
	ExpiresAt           time.Time
}

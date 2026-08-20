# Architecture

This project is organized like a production Go service: the binary entrypoint is small, and the MCP server logic is split into focused internal packages.

## Runtime flow

1. `cmd/multicard-mcp-go/main.go` parses configuration through `internal/config`.
2. `internal/docs` loads every markdown file from `multicard-docs`, extracts metadata, tokenizes content, and builds a search index.
3. The binary creates `internal/mcp.Server` with the loaded docs corpus.
4. Depending on flags, the server runs either:
   - stdio MCP transport for local MCP clients, or
   - HTTP JSON-RPC transport with `/mcp`, `/healthz`, and `/readyz` endpoints, where `/mcp` is
     protected by an `internal/oauth.Server` (OAuth 2.1 authorization code + PKCE).
5. MCP tools call into the docs search/index layer and use `internal/render` to return text plus structured JSON payloads.

## File and package guide

### `cmd/multicard-mcp-go/main.go`
Application entrypoint. It wires packages together, handles one-off CLI modes (`--search`, `--ask`, `--get-doc`), and starts stdio or HTTP mode.

### `internal/config/config.go`
Owns flags, environment defaults, and locating `multicard-docs`. This keeps startup/configuration concerns out of MCP protocol code.

### `internal/docs/types.go`
Defines the domain model: `Document`, `Corpus`, and `ScoredDoc`. It also exposes safe accessor methods used by MCP handlers.

### `internal/docs/loader.go`
Walks the docs directory, reads markdown files, parses each page, and builds lookup maps plus document-frequency data for search ranking.

### `internal/docs/parser.go`
Extracts metadata from markdown/OpenAPI-style content: title, endpoint path, HTTP method, summary, tags, error codes, required fields, topics, and plain text.

### `internal/docs/tokenizer.go`
Normalizes natural-language text into search tokens and expands common English/Russian Multicard synonyms.

### `internal/docs/topics.go`
Detects high-level API topics such as token, card, invoice, payment, refund, callback, payout, and split. Search uses these topics to reduce false positives.

### `internal/docs/search.go`
Implements the local BM25-style heuristic search, score boosts, topic penalties, and snippet extraction.

### `internal/mcp/types.go`
Defines MCP server state, server version constants, and JSON-RPC request/response DTOs.

### `internal/mcp/protocol.go`
Handles stdio MCP framing (`Content-Length`) and routes JSON-RPC methods like `initialize`, `tools/list`, `tools/call`, `resources/list`, and `resources/read`.

### `internal/mcp/http.go`
Provides HTTP JSON-RPC transport and operational endpoints (`/healthz`, `/readyz`, `/mcp`). This is the deployment-friendly transport. `/mcp` is wrapped in `oauth.Server.RequireToken`, so every request needs a valid bearer access token.

### `internal/oauth/`
Self-contained OAuth 2.1 authorization server and resource-server guard, stdlib-only:

- `types.go` / `store.go` — in-memory `Client`, `AuthCode`, `Token` models and thread-safe storage (no database; tokens don't need to survive a restart).
- `pkce.go` — PKCE (`S256`) verification and constant-time comparisons.
- `server.go` — HTTP handlers for dynamic client registration (`/register`, RFC 7591), the login/consent screen (`/authorize`), the token endpoint (`/token`, authorization_code + refresh_token grants), discovery metadata (`/.well-known/oauth-authorization-server`, `/.well-known/oauth-protected-resource`), and the `RequireToken` middleware that guards `/mcp`.

There is no real user database: `/authorize` shows one fixed demo login (configurable via `OAUTH_DEMO_USER`/`OAUTH_DEMO_PASSWORD`), so the human-approval step of the flow is still visible without needing an identity provider.

### `internal/mcp/tools.go`
Defines MCP tool schemas and implementations:

- `search_multicard_docs`
- `get_multicard_doc`
- `answer_multicard_question`

### `internal/mcp/resources.go`
Exposes every markdown document as an MCP resource and supports `resources/read` by document URI.

### `internal/render/render.go`
Converts search results into human-readable answers and JSON-friendly structured payloads returned by MCP tools.

### `internal/util/util.go`
Small reusable helpers (`Clamp`, `Min`, `Max`) shared across packages.

## CI/CD notes

- CI runs `gofmt`, `go test ./...`, and `go build ./...`.
- Release and deploy workflows build the command from `./cmd/multicard-mcp-go`.
- Deployment bundles include the compiled binary, `multicard-docs`, deployment files, scripts, and documentation.

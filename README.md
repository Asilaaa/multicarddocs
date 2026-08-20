# multicard-mcp-go

Go-based MCP server built from the bundled `multicard-docs` markdown pages.

It can run in two modes:

- **stdio MCP mode** for local MCP clients
- **HTTP JSON-RPC mode** for server deployment behind systemd/nginx/caddy

The repository is now self-contained: the markdown docs are included in `./multicard-docs`.

## Features

- loads local Multicard markdown docs
- searches them with lightweight local ranking
- returns grounded answers with source pages
- exposes MCP tools and MCP resources
- can be deployed as a long-running HTTP service

## MCP tools

### `search_multicard_docs`
Searches all markdown pages and returns the most relevant results with snippets.

Example input:

```json
{
  "query": "how to create invoice",
  "limit": 5
}
```

### `get_multicard_doc`
Returns one full markdown page by relative path.

Example input:

```json
{
  "path": "endpoints/создание-инвойса-19729296e0.md"
}
```

### `answer_multicard_question`
Returns a concise answer grounded in the docs with source pages.

Example input:

```json
{
  "question": "How do I get an auth token?",
  "limit": 4
}
```

## Project layout

```text
multicard-mcp-go/
├── cmd/
│   └── multicard-mcp-go/
│       └── main.go              # CLI entrypoint; parses mode and starts the app
├── internal/
│   ├── config/                  # flags, env vars, docs-directory resolution
│   ├── docs/                    # markdown loading, metadata extraction, search index
│   ├── mcp/                     # MCP JSON-RPC protocol, tools, resources, transports
│   ├── render/                  # human-readable answers and structured payloads
│   └── util/                    # small shared helpers
├── multicard-docs/              # bundled source markdown docs served by the MCP server
├── deploy/                      # systemd/nginx deployment examples
├── scripts/                     # install/deployment scripts
├── .github/workflows/           # CI, release, and deployment automation
├── Makefile                     # local build/test/package commands
├── go.mod
└── README.md
```

For a detailed file-by-file explanation, see [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Build

```bash
cd multicard-mcp-go
go build -o bin/multicard-mcp-go ./cmd/multicard-mcp-go
```

Or:

```bash
make build
```

## Local usage

### 1. stdio MCP mode

```bash
./bin/multicard-mcp-go
```

This starts the MCP server and waits for a client on stdin/stdout.

### 2. HTTP mode

```bash
./bin/multicard-mcp-go --http --listen-addr 127.0.0.1:8080
```

Available endpoints:

- `GET /healthz`
- `GET /readyz`
- `POST /mcp` — OAuth 2.1 protected, requires `Authorization: Bearer <token>`
- `GET /.well-known/oauth-protected-resource` — RFC 9728 resource metadata
- `GET /.well-known/oauth-authorization-server` — RFC 8414 authorization server metadata
- `POST /register` — RFC 7591 dynamic client registration
- `GET/POST /authorize` — login + consent screen
- `POST /token` — authorization code / refresh token exchange

Example:

```bash
curl -s http://127.0.0.1:8080/healthz
```

Calling `/mcp` without a token is rejected with a `401` that points at the discovery metadata:

```bash
curl -si http://127.0.0.1:8080/mcp -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
# HTTP/1.1 401 Unauthorized
# Www-Authenticate: Bearer resource_metadata="http://127.0.0.1:8080/.well-known/oauth-protected-resource"
```

### 3. OAuth 2.1 flow (authorization code + PKCE)

This server is both the OAuth authorization server and the protected resource — matching how a
standalone remote MCP server is expected to behave per the MCP Authorization spec. There is no
real user database: `/authorize` shows a fixed demo login (`demo` / `demo1234` by default,
overridable via `OAUTH_DEMO_USER` / `OAUTH_DEMO_PASSWORD`) so the human-approval step in the flow
is still visible end to end.

1. **Register a client** (`internal/oauth/server.go` → `handleRegister`):

   ```bash
   curl -s -X POST http://127.0.0.1:8080/register \
     -H 'Content-Type: application/json' \
     -d '{"client_name":"Demo Client","redirect_uris":["http://127.0.0.1:9999/callback"]}'
   ```

2. **Send the user to `/authorize`** with a PKCE `code_challenge` (`S256`). The server renders a
   login + consent page (`handleAuthorize` / `authorizeTemplate`); approving redirects back to
   `redirect_uri` with `?code=...`.

3. **Exchange the code for a token** (`handleToken` → `exchangeAuthCode`), presenting the original
   PKCE `code_verifier`:

   ```bash
   curl -s -X POST http://127.0.0.1:8080/token \
     -d grant_type=authorization_code -d code=<code> \
     -d redirect_uri=http://127.0.0.1:9999/callback \
     -d client_id=<client_id> -d code_verifier=<verifier>
   ```

4. **Call `/mcp`** with the returned `access_token` as a bearer token.

Access tokens last 1 hour; refresh tokens (`grant_type=refresh_token`) last 30 days and rotate on
each use. See `internal/oauth/` for the full implementation — it's deliberately small and
dependency-free (stdlib only) so it reads well end to end.

### 4. Quick CLI testing without an MCP client

Search:

```bash
./bin/multicard-mcp-go --search "create invoice"
```

Ask:

```bash
./bin/multicard-mcp-go --ask "How do I get an auth token?"
```

Read one page:

```bash
./bin/multicard-mcp-go --get-doc "endpoints/получение-токена-19729295e0.md"
```

## Example MCP client config

For a local stdio MCP client:

```json
{
  "mcpServers": {
    "multicard-docs": {
      "command": "/home/asila/Documents/multicard-mcp-go/bin/multicard-mcp-go"
    }
  }
}
```

## systemd deployment

Files:

- `deploy/systemd/multicard-mcp.service`
- `deploy/systemd/multicard-mcp.env.example`
- `scripts/install.sh`

Install on a Linux server:

```bash
cd /opt/multicard-mcp
chmod +x scripts/install.sh
sudo INSTALL_DIR=/opt/multicard-mcp bash scripts/install.sh
sudo systemctl enable --now multicard-mcp.service
```

Default HTTP bind address in the env file:

```bash
LISTEN_ADDR=127.0.0.1:8080
```

Set `PUBLIC_BASE_URL` to the externally reachable URL (e.g. `https://multicardocs.sukoon.uz`) so
OAuth discovery metadata and redirects are built with the right host instead of being inferred
from the proxied request. `OAUTH_DEMO_USER` / `OAUTH_DEMO_PASSWORD` control the login shown on the
`/authorize` consent screen — change them from the defaults before deploying somewhere public.

Env file location:

```bash
/etc/multicard-mcp/multicard-mcp.env
```

Check status:

```bash
systemctl status multicard-mcp.service
journalctl -u multicard-mcp.service -f
```

## Reverse proxy

Optional nginx example:

- `deploy/nginx/multicard-mcp.conf.example`

## GitHub Actions

### CI

Workflow:

- `.github/workflows/ci.yml`

It runs:

- `gofmt` check
- `go test ./...`
- `go build ./...`

The executable entrypoint is in `cmd/multicard-mcp-go`; reusable server logic lives under `internal/`.

### Release

Workflow:

- `.github/workflows/release.yml`

On tags like `v0.2.0`, it builds release tarballs for:

- linux/amd64
- linux/arm64

### Deploy

Workflow:

- `.github/workflows/deploy.yml`

This is a simple SSH-based deployment workflow.

Required GitHub secrets:

- `DEPLOY_HOST`
- `DEPLOY_USER`
- `DEPLOY_PATH`
- `DEPLOY_SSH_KEY`

The workflow:

1. builds the Linux binary
2. uploads a deployment bundle
3. runs `scripts/install.sh` on the server
4. restarts `multicard-mcp.service`

## Make targets

```bash
make fmt
make test
make build
make run-http
make package-linux-amd64
make clean
```

## Notes

- No external API calls are required.
- The server reads only local markdown files.
- Search is heuristic/BM25-style with RU/EN keyword handling for common Multicard terms.
- HTTP mode is intended for deployment convenience; stdio mode is still the default MCP mode.

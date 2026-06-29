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
├── multicard-docs/
├── deploy/
│   ├── nginx/
│   └── systemd/
├── scripts/
├── .github/workflows/
├── main.go
├── main_test.go
├── go.mod
└── README.md
```

## Build

```bash
cd multicard-mcp-go
go build -o bin/multicard-mcp-go .
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
- `POST /mcp`

Example:

```bash
curl -s http://127.0.0.1:8080/healthz
```

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

### 3. Quick CLI testing without an MCP client

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

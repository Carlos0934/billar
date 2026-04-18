# Billar

Billar is a Go application with:

- a CLI entrypoint in `cmd/cli`
- an HTTP MCP server in `cmd/mcp-http`
- SQLite-backed storage

## Current MCP setup

MCP is served over HTTP only.

- Endpoint: `http://127.0.0.1:8080/v1/mcp`
- Auth: `Authorization: Bearer <api-key>`
- Config: `MCP_API_KEYS` (required)

Example `opencode.json` snippet:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "billar": {
      "type": "remote",
      "url": "http://127.0.0.1:8080/v1/mcp",
      "headers": {
        "Authorization": "Bearer <your-api-key>"
      }
    }
  }
}
```

Generate a key with:

```bash
openssl rand -hex 32
```

## Environment

Copy values from `.env.example`:

```env
MCP_API_KEYS=your-secret-api-key-here
MCP_HTTP_LISTEN_ADDR=127.0.0.1:8080
```

`BILLAR_DB_PATH` is also supported if you want to point Billar at a specific SQLite database.

## Commands

Use the `Makefile` targets:

```bash
make test
make build
make fmt
make run-health
make run-customer-list
make run-mcp-http
```

## Architecture

- `internal/core` — domain types
- `internal/app` — services and DTOs
- `internal/connectors` — CLI and MCP transport layer
- `internal/infra` — config, logging, SQLite

## Notes

- `cmd/mcp-http` is the only MCP entrypoint.
- Legacy OAuth/OIDC and stdio MCP flows have been removed.
- `session.status` keeps an internal fixed identity for compatibility, but hides synthetic identity fields from text output.

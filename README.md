# Billar

Billar is a Go billing application with:

- a CLI entrypoint in `cmd/cli`
- an HTTP MCP server in `cmd/mcp-http`
- SQLite-backed storage

## Prerequisites

- Go 1.25.8

## Setup

Copy `.env.example` to `.env` for local runs, then adjust values as needed:

```env
LOG_LEVEL=info
MCP_API_KEYS=your-secret-api-key-here
MCP_HTTP_LISTEN_ADDR=127.0.0.1:8080
BILLAR_EXPORT_DIR=/tmp/billar-exports
# BILLAR_DB_PATH=/absolute/path/to/billar.db
```

Billar auto-loads `.env`; existing non-empty environment variables take precedence.

## Commands

Use the `Makefile` targets:

```bash
make test
make build
make fmt
make run-health
make run-customer-list
make run-mcp-http
make run-invoice-import FILE=./path/to/invoice-import.json
```

Focused examples:

```bash
go run ./cmd/cli health
go run ./cmd/cli customer list --format json
go run ./cmd/cli invoice import --file ./path/to/invoice-import.json --format toon
go run ./cmd/cli invoice import --stdin < ./path/to/invoice-import.json
go run ./cmd/mcp-http
```

CLI commands support `--format text|json|toon` where the command exposes formatted output.

## Environment

`BILLAR_DB_PATH` is optional. When unset or blank, startup resolves the first available persistent SQLite path in this order:

1. `$XDG_DATA_HOME/billar/billar.db`
2. `os.UserConfigDir()/billar/billar.db`
3. `$HOME/.local/share/billar/billar.db` (for example, `~/.local/share/billar/billar.db`)

Set `BILLAR_DB_PATH=/absolute/path/to/billar.db` to override this default. Billar creates the parent directory for the default path before opening the database.

`BILLAR_EXPORT_DIR` roots MCP file-output tools such as `invoice.render_pdf`.

## MCP HTTP setup

MCP is served over HTTP only.

- Endpoint: `http://127.0.0.1:8080/v1/mcp`
- Health: `http://127.0.0.1:8080/healthz`
- Auth: `Authorization: Bearer <api-key>`
- Required config: `MCP_API_KEYS` (one or more comma-separated keys)
- Listen address: `MCP_HTTP_LISTEN_ADDR` (defaults to `127.0.0.1:8080`)

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

### Invoice PDF export

CLI usage writes to an explicit path and returns file metadata in `text`, `json`, or `toon` format:

```bash
go run ./cmd/cli invoice pdf <invoice-id> --out ./exports/invoice.pdf --format json
```

MCP exposes `invoice.render_pdf` with input `{ "invoice_id": "inv_123", "filename": "invoice.pdf" }` or `{ "invoice_id": "inv_123", "output_path": "nested/invoice.pdf" }`. MCP output paths must stay relative to `BILLAR_EXPORT_DIR`; absolute paths, traversal (`..`), and separators in `filename` are rejected.

## Architecture

- `internal/core` — domain types
- `internal/app` — services and DTOs
- `internal/connectors` — CLI and MCP transport layer
- `internal/infra` — config, logging, SQLite, PDF rendering, export file writing

## Notes

- `cmd/mcp-http` is the only MCP entrypoint.
- Legacy OAuth/OIDC and stdio MCP flows have been removed.
- `session.status` keeps an internal fixed identity for compatibility, but hides synthetic identity fields from text output.

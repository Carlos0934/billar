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
BILLAR_EXPORT_DIR=/absolute/path/to/pdf-exports
```

Billar stores SQLite data in a persistent per-user database by default. When
`BILLAR_DB_PATH` is unset or blank, startup resolves the first available path in
this order:

1. `$XDG_DATA_HOME/billar/billar.db`
2. `os.UserConfigDir()/billar/billar.db`
3. `$HOME/.local/share/billar/billar.db` (for example, `~/.local/share/billar/billar.db`)

Set `BILLAR_DB_PATH=/absolute/path/to/billar.db` to override this default. A
non-empty `BILLAR_DB_PATH` always wins, and Billar creates the parent directory
before opening the database.
`BILLAR_EXPORT_DIR` enables MCP file-output tools such as `invoice.render_pdf`; paths supplied to MCP are resolved under this directory.

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

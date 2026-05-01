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
# BILLAR_BACKUP_DIR=/absolute/path/to/billar-backups
```

Billar auto-loads `.env` from the current working directory only; existing non-empty environment variables take precedence. When using a globally installed binary from another directory, export `BILLAR_DB_PATH`, `BILLAR_EXPORT_DIR`, and `BILLAR_BACKUP_DIR` in your shell if you do not want cwd-local `.env` discovery.

For global onboarding, run `billar setup`. It creates the resolved DB parent, export, and backup directories with user-only permissions where supported, reports created vs already-existing paths, and does not write or update `.env`.

## Global install

Install a binary literally named `billar` into your Go bin directory:

```bash
make install
```

The target uses `go env GOBIN`; when `GOBIN` is empty it falls back to `$(go env GOPATH)/bin`. Override the install directory when needed with `make BINDIR=/custom/bin install` (and uninstall with the same `BINDIR` value). Ensure that directory is on `PATH`, then `billar --help` and all CLI commands work from outside the checkout. Remove the default install with `make uninstall`.

After installing globally, the usual first-run flow is:

```bash
billar setup
billar doctor --format json
```

`billar setup` is the only setup command that creates runtime directories. `billar doctor` is read-only: it reports DB/export/backup path readiness, whether each path came from an environment override or a default, and suggests `billar setup` when required directories are missing.

## Commands

Use the `Makefile` targets:

```bash
make test
make build
make install
make fmt
make run-health
make run-customer-list
make run-mcp-http
make run-invoice-import FILE=./path/to/invoice-import.json
```

Focused examples:

```bash
go run ./cmd/cli health
go run ./cmd/cli setup --format json
go run ./cmd/cli customer list --format json
go run ./cmd/cli backup create --format text
go run ./cmd/cli backup list --format toon
go run ./cmd/cli invoice import --file ./path/to/invoice-import.json --format toon
go run ./cmd/cli invoice import --stdin < ./path/to/invoice-import.json
go run ./cmd/mcp-http
```

CLI commands support `--format text|json|toon` where the command exposes formatted output.

### CLI-first billing operations

The CLI/app service path is the trusted surface for day-to-day billing work. Direct SQLite access is emergency repair-only and should require explicit operator approval; do not use ad-hoc SQL for normal invoice mutations. The MCP server remains available, but MCP billing writes are not trusted in the current operational posture.

Useful billing commands:

```bash
go run ./cmd/cli doctor --format json
go run ./cmd/cli invoice inspect --id inv_123 --format json
go run ./cmd/cli invoice update-metadata --id inv_123 --invoice-date 2026-05-01 --payment-terms "Net 15"
```

`invoice update-metadata` changes non-financial invoice metadata only (`invoice_date`, `due_date`, `payment_terms`, `payment_communication`). Omitted or empty flags leave existing values unchanged.

## Environment

`BILLAR_DB_PATH` is optional. When unset or blank, startup resolves the first available persistent SQLite path in this order:

1. `$XDG_DATA_HOME/billar/billar.db`
2. `os.UserConfigDir()/billar/billar.db`
3. `$HOME/.local/share/billar/billar.db` (for example, `~/.local/share/billar/billar.db`)

Set `BILLAR_DB_PATH=/absolute/path/to/billar.db` to override this default. Billar creates the parent directory for the default path before opening the database.

`BILLAR_EXPORT_DIR` roots relative PDF exports. If it is unset, `billar invoice pdf <invoice-id>` requires an explicit `--out <path>` and returns an actionable error naming both `BILLAR_EXPORT_DIR` and `--out` when neither is available.

`BILLAR_BACKUP_DIR` is optional. When unset or blank, backups default to `<db-parent>/backups`; set `BILLAR_BACKUP_DIR=/absolute/path/to/backups` to override it.

## Local backups

Create and list SQLite-consistent local snapshots:

```bash
billar backup create --format json
billar backup list --format text
```

`billar backup create` writes a backup database file named like `billar-<utc>-schemaN.db` plus a matching `.db.json` sidecar containing `created_at`, `source_db_path`, `size_bytes`, `sha256`, and `schema_version`. Backup artifacts use `.db` and `.db.json`; Billar does not create `.sqlite` backup files. Backups contain sensitive billing and legal data; they are unencrypted local snapshots and should be protected like the live database.

`billar backup list` reads the backup directory and sidecar metadata, returning the canonical `backups` array for machine formats. Entries without a sidecar can still be listed with `metadata: false`.

Restore is intentionally deferred to future work and is not available/not implemented as a command in this slice. Until a safe restore lifecycle exists, the manual workaround is to stop Billar, copy the desired backup `.db` over the configured `BILLAR_DB_PATH`, then restart Billar.

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

CLI usage writes to an explicit path or, when `BILLAR_EXPORT_DIR` is set, to a default invoice filename under that export root. It returns file metadata in `text`, `json`, or `toon` format:

```bash
go run ./cmd/cli invoice pdf <invoice-id> --out ./exports/invoice.pdf --format json
go run ./cmd/cli invoice pdf <invoice-id> --format text
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

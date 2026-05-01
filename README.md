# Billar

Billar is a Go billing application with:

- a CLI entrypoint in `cmd/cli`
- SQLite-backed storage

## Prerequisites

- Go 1.25.8

## Setup

Copy `.env.example` to `.env` for local runs, then adjust values as needed:

```env
LOG_LEVEL=info
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
```

CLI commands support `--format text|json|toon` where the command exposes formatted output.

### CLI-first billing operations

The CLI/app service path is the only supported surface for day-to-day billing work. Direct SQLite access is emergency repair-only and should require explicit operator approval; do not use ad-hoc SQL for normal invoice mutations.

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

Create, list, and safely restore SQLite-consistent local snapshots:

```bash
billar backup create --format json
billar backup list --format text
billar backup restore --id billar-20260501T120000Z-schema4 --dry-run --format json
billar backup restore --file /path/to/billar-20260501T120000Z-schema4.db --confirm billar-20260501T120000Z-schema4.db
```

`billar backup create` writes a backup database file named like `billar-<utc>-schemaN.db` plus a matching `.db.json` sidecar containing `created_at`, `source_db_path`, `size_bytes`, `sha256`, and `schema_version`. Backup artifacts use `.db` and `.db.json`; Billar does not create `.sqlite` backup files. Backups contain sensitive billing and legal data; they are unencrypted local snapshots and should be protected like the live database.

`billar backup list` reads the backup directory and sidecar metadata, returning the canonical `backups` array for machine formats. Entries without a sidecar can still be listed with `metadata: false`.

`billar backup restore` validates a sidecar-backed `.db` before replacing the configured `BILLAR_DB_PATH`. Select exactly one source with `--id <backup-id>` (looked up under `BILLAR_BACKUP_DIR`) or `--file <path-to-backup.db>`. Use `--dry-run` to print the restore plan without changing the live database. Destructive restores require `--confirm <token>` matching the backup ID or backup file basename, or `--force` for automation. Validation checks the sidecar, size, sha256, SQLite integrity, schema version, and required Billar tables; `--force` does not bypass validation.

When the live DB exists, restore first creates a pre-restore safety snapshot in the backup directory and reports it in the output. If the live DB does not exist, restore proceeds with `safety_snapshot: null` and a warning. Every destructive restore attempt emits a `concurrent_processes` warning: stop other Billar processes and external SQLite clients before running it. Machine-format failures include `error.code` alongside the stable exit code meanings: `0` success/dry-run, `2` usage or confirmation errors, `3` validation failures, `4` snapshot/copy/rename runtime failures, and `5` unexpected internal errors.

## Migration: MCP tools → CLI commands

Billar is CLI-only. Use these command forms for workflows formerly exposed through the removed tool surface:

| Former workflow | CLI command |
|---|---|
| `session.status` | Removed; CLI runs as the local operator. |
| `health.check` | `billar health` |
| `doctor.report` | `billar doctor [--format json\|toon]` |
| `customer_profile.create` | `billar customer create` |
| `customer_profile.list` | `billar customer list` |
| `customer_profile.get` | `billar customer get --id <customer-profile-id>` |
| `customer_profile.update` | `billar customer update --id <customer-profile-id>` |
| `customer_profile.delete` | `billar customer delete --id <customer-profile-id>` |
| `service_agreement.create` | `billar agreement create` |
| `service_agreement.list_by_customer_profile` | `billar agreement list --customer <customer-profile-id>` |
| `service_agreement.update_rate` | `billar agreement rate-update --id <agreement-id>` |
| `service_agreement.activate` | `billar agreement activate --id <agreement-id>` |
| `service_agreement.deactivate` | `billar agreement deactivate --id <agreement-id>` |
| `time_entry.record` | `billar time add` |
| `time_entry.list_by_customer_profile` | `billar time list --customer <customer-profile-id>` |
| `time_entry.list_unbilled` | `billar time list-unbilled --customer <customer-profile-id>` |
| `invoice.draft` | `billar invoice draft --customer-id <customer-profile-id>` |
| `invoice.issue` | `billar invoice issue --id <invoice-id>` |
| `invoice.discard` | `billar invoice discard --id <invoice-id>` |
| `invoice.show` | `billar invoice show --id <invoice-id>` |
| `invoice.inspect` | `billar invoice inspect --id <invoice-id>` |
| `invoice.list` | `billar invoice list --customer-id <customer-profile-id> [--status <status>]` |
| `invoice.line.add` | `billar invoice line add --invoice-id <invoice-id>` |
| `invoice.line.remove` | `billar invoice line remove --invoice-id <invoice-id> --line-id <line-id>` |
| `invoice.render_pdf` | `billar invoice pdf <invoice-id> --out <path>` |
| `invoice.import` | `billar invoice import --file <path>` or `billar invoice import --stdin` |
| `setup.*` | `billar setup` |
| `backup.*` | `billar backup create` / `billar backup list` / `billar backup restore` |

### Invoice PDF export

CLI usage writes to an explicit path or, when `BILLAR_EXPORT_DIR` is set, to a default invoice filename under that export root. It returns file metadata in `text`, `json`, or `toon` format:

```bash
go run ./cmd/cli invoice pdf <invoice-id> --out ./exports/invoice.pdf --format json
go run ./cmd/cli invoice pdf <invoice-id> --format text
```

## Architecture

- `internal/core` — domain types
- `internal/app` — services and DTOs
- `internal/connectors` — CLI transport layer
- `internal/infra` — config, logging, SQLite, PDF rendering, export file writing

## Notes

- The CLI is the sole supported operational interface.

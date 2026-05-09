# Billar

Billar is a CLI-only local billing tool for a single operator. It stores billing data in SQLite, renders invoice PDFs, and keeps day-to-day work on the `billar` command surface.

## Prerequisites

- Go 1.25.x (this repo currently uses Go 1.25.8).
- Your Go bin directory on `PATH` after install.

## Install

```bash
make install
```

`make install` builds a binary named `billar`. The Makefile chooses `BINDIR` from `go env GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is empty. Override it with `make BINDIR=/custom/bin install` when needed.

## First run

```bash
billar setup
billar doctor --format json
billar health
```

`billar setup` creates the resolved database parent, export directory, and backup directory. It does not write `.env`. `billar doctor` checks DB/export/backup readiness and reports where each path came from.

Configuration is optional. Billar auto-loads `.env` from the current working directory only; existing non-empty environment variables win over `.env` values.

Common optional overrides:

- `BILLAR_DB_PATH=/absolute/path/to/billar.db`
- `BILLAR_EXPORT_DIR=/absolute/path/to/exports`
- `BILLAR_BACKUP_DIR=/absolute/path/to/backups`

When unset, `BILLAR_DB_PATH` resolves to a per-user `billar.db`, and `BILLAR_EXPORT_DIR` plus `BILLAR_BACKUP_DIR` default under that DB parent. See `.env.example` and [operations](docs/operations.md).

## Happy-path billing flow

Create the records needed for hourly billing:

```bash
billar customer create --json '{"type":"company","legal_name":"Acme Corp","default_currency":"USD"}'
billar customer list --format text
billar agreement create --customer-id cus_123 --json '{"name":"Default hourly","billing_mode":"hourly","hourly_rate":10000,"currency":"USD"}'
billar agreement update-rate --id sa_123 --json '{"hourly_rate":12000}'
billar time-entry record --json '{"customer_profile_id":"cus_123","service_agreement_id":"sa_123","description":"Build billing flow","hours":120,"billable":true,"date":"2026-05-01T00:00:00Z"}'
billar time-entry list --customer-id cus_123
```

Draft, issue, inspect, import, and export invoices:

```bash
billar invoice draft --customer-id cus_123 --period-start 2026-05-01 --period-end 2026-05-31 --due-date 2026-06-15
billar invoice issue --id inv_123
billar invoice inspect --id inv_123 --format json
billar invoice import --file examples/imports/INV-2026-00001.json
billar invoice pdf inv_123 --out ./exports/invoice.pdf
```

The CLI/app service path is the supported path for billing. Direct SQLite edits are emergency repair-only and require explicit operator approval.

## Backups and restore

```bash
billar backup create --format json
billar backup list --format text
billar backup restore --id billar-20260501T120000Z-schema4 --dry-run --format json
billar backup restore --file /path/to/billar-20260501T120000Z-schema4.db --confirm billar-20260501T120000Z-schema4.db
```

`billar backup create` writes a SQLite snapshot named like `billar-<utc>-schemaN.db` plus a matching `.db.json` sidecar with size, sha256, source path, timestamp, and schema version. Backup artifacts use `.db` and `.db.json`; no .sqlite backup filenames are produced.

Backups contain sensitive billing and legal data. They are unencrypted local snapshots, so protect them like the live database.

`billar backup restore` selects exactly one source with `--id` or `--file`. Use `--dry-run` to review the plan. Destructive restore requires `--confirm <token>` or `--force`; validation still checks the sidecar, size, sha256, SQLite integrity, schema version, and required tables. When a live DB exists, restore first writes a pre-restore safety snapshot and reports it. Every restore plan includes a `concurrent_processes` warning: stop other Billar processes and SQLite clients first. Exit code meanings: `0` success or dry-run, `2` usage/confirmation, `3` validation, `4` snapshot/copy/rename runtime, `5` unexpected internal error.

See [operations](docs/operations.md) for the runbook.

## Developer commands

The Makefile is the command source of truth:

```bash
make test
make build
make fmt
make run-health
make run-customer-list
make run-invoice-import FILE=examples/imports/INV-2026-00001.json
```

Most output-bearing commands support `--format text|json|toon`.

## Docs map

- [Operations](docs/operations.md): setup, config, doctor, backups, restore, exit codes.
- [Invoices](docs/invoices.md): import, inspect, metadata updates, PDF export, repair policy.
- [Technical blueprint](docs/technical_blueprint.md): architecture layers and boundaries.

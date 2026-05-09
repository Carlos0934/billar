# Billar Technical Blueprint

Billar is a CLI-only local billing application. The architecture is a pragmatic modular monolith: core billing rules stay independent, application services coordinate use cases, the CLI translates operator commands, and infrastructure owns storage, configuration, rendering, exports, and backups.

Operational runbooks live in [docs/operations.md](operations.md). Invoice import/PDF details live in [docs/invoices.md](invoices.md).

## Scope

Billar manages:

- legal entities shared by issuer and customer roles
- customer profiles
- customer-specific service agreements
- billable time entries
- invoice drafting, issuing, importing, inspection, metadata corrections, and PDF rendering
- local SQLite backup and restore; restore implemented through the CLI backup service

The system intentionally remains single-operator and local-first. It avoids distributed services, event sourcing, CQRS, workflow engines, cloud sync, and multi-tenant concerns.

## Architecture style

- Pragmatic modular monolith
- Simple layered architecture
- CLI-only connector
- SQLite persistence behind infrastructure seams
- PDF rendering behind an application-facing rendering boundary
- Boring Go code over heavy framework abstractions

## Layers

### `internal/core`

`internal/core` contains domain types and business rules. It models legal entities, customers, issuers, service agreements, time entries, invoices, money, hours, currencies, and validation. Core code must not know about SQLite, CLI flags, filesystems, config loading, logging, or PDF generation.

### `internal/app`

`internal/app` exposes application services and DTOs. It coordinates use cases such as customer creation, agreement changes, time entry recording, invoice drafting/issuing/importing, metadata updates, PDF rendering requests, setup checks, and backup/restore orchestration. Application services consume storage/rendering/export seams rather than concrete infrastructure.

### `internal/connectors/cli`

`internal/connectors/cli` is the sole operator connector. It parses commands, validates transport-level flags, calls `internal/app` services, and writes `text`, `json`, or `toon` output from canonical DTOs. Representative command forms include `billar health`, `billar doctor`, `billar invoice import --file`, and `billar invoice pdf`.

### `internal/infra`

`internal/infra` implements technical details: SQLite storage, migrations, config resolution, logging, PDF rendering, export filesystem writes, setup path checks, and backup files. Infrastructure may depend inward on app/core contracts; core must not depend outward on infrastructure.

## Boundaries

### Storage boundary

SQLite access belongs in `internal/infra`. CLI code must not bypass `internal/app` to mutate core state, and `internal/core` must never issue SQL. Invoice totals come from invoice lines persisted through application services.

### Access/auth boundary

Billar currently assumes a single local operator. Access/auth concerns are limited to local process execution and filesystem permissions. If richer authorization appears later, it should be introduced as an application boundary without leaking into core billing calculations.

### Rendering boundary

PDF rendering is an infrastructure implementation behind app-level requests. Renderers format already-issued invoice data; they do not calculate invoice totals or mutate billing state.

### Configuration boundary

`internal/infra/config` resolves `.env`, `BILLAR_DB_PATH`, `BILLAR_EXPORT_DIR`, and `BILLAR_BACKUP_DIR`. Executable config behavior is the source of truth for docs and operations.

## Architectural decisions

- SQLite is the local persistence engine.
- Money and durations are integer-backed billing values.
- Hourly billing is the first-class billing mode.
- Customer profiles may have one or more service agreements.
- Issued invoices must remain reproducible.
- Normal billing operations are CLI/app-service first; direct SQLite edits are emergency repair-only and require explicit operator approval.
- Backups are local SQLite snapshots with metadata sidecars and are sensitive, unencrypted artifacts.

## Command surface reference

The CLI command surface is defined by `cmd/cli/main.go` and `internal/connectors/cli`. Makefile targets such as `make test`, `make build`, `make fmt`, and `make run-health` are development conveniences, not separate runtime connectors.

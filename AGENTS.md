# Billar Agent Notes

Billar is a CLI-only local billing tool. Use this file for repo-specific rules that are easy to miss; prefer executable sources over prose when they conflict.

## Project skills

| Skill | Use When | Path |
|---|---|---|
| `golang-patterns` | Writing or reviewing Billar Go code, especially `internal/core`, `internal/app`, `internal/connectors`, `internal/infra`, tests, and early scaffolding | `.agents/skills/golang-patterns/SKILL.md` |
| `architecture-billar` | Designing or reviewing package layout, service boundaries, connector flow, storage/auth/rendering boundaries, and blueprint alignment | `.agents/skills/architecture-billar/SKILL.md` |
| `billar-command-output` | Creating or reviewing Billar command outputs, canonical DTOs, CLI format handling, and human-readable text rendering | `.agents/skills/billar-command-output/SKILL.md` |

## Source of truth

- Commands: `Makefile`, `cmd/cli/main.go`, and `internal/connectors/cli/command.go`.
- Runtime config: `internal/infra/config/*`.
- Architecture intent: `docs/technical_blueprint.md`, verified against code.

## Commands

- `make test`
- `make build`
- `make fmt` (`gofmt -w ./cmd ./internal`; format other touched Go files explicitly)
- `make run-health`
- `make run-customer-list`
- `make run-invoice-import FILE=...`

## Coding boundaries

- Go version: 1.25.8. Prefer the standard library; ask before adding external dependencies.
- Main entrypoint: `cmd/cli/main.go`.
- Keep layers intact: `internal/core` for domain rules, `internal/app` for services/DTOs/seams, `internal/connectors/cli` for transport translation, and `internal/infra` for config/logging/SQLite/PDF/export implementations.
- CLI commands must call shared `internal/app` services. Do not access SQLite from `internal/core`, and do not let CLI code bypass `internal/app` for core state changes.
- Money and durations are integer-backed billing values; never use floats for billing calculations. Invoice totals come from invoice lines, not renderers.

## Docs and output rules

- Keep `docs/technical_blueprint.md` architecture-focused. Put setup, restore, and invoice runbooks in `docs/operations.md` or `docs/invoices.md`.
- Document canonical command names exactly as wired by the CLI.
- `internal/connectors/cli` supports `text`, `json`, and `toon`. Use `OutputResult{Payload, TextWriter}`; keep one canonical DTO for machine formats with both `json` and `toon` tags.

## Local metadata

`.atl/`, `.env`, `go.work*`, and `skills-lock.json` are local/ignored metadata. Do not intentionally add them unless the user explicitly asks.

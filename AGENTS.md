# Billar Agent Notes

Use project-local skills before generic guidance; this file only records repo-specific rules an agent is likely to miss.

| Skill | Use When | Path |
|---|---|---|
| `golang-patterns` | Writing or reviewing Billar Go code, especially `internal/core`, `internal/app`, `internal/connectors`, `internal/infra`, tests, and early scaffolding | `.agents/skills/golang-patterns/SKILL.md` |
| `architecture-billar` | Designing or reviewing package layout, service boundaries, connector flow, storage/auth/rendering boundaries, and blueprint alignment | `.agents/skills/architecture-billar/SKILL.md` |
| `billar-command-output` | Creating or reviewing Billar command outputs, canonical DTOs, CLI format handling, and human-readable text rendering | `.agents/skills/billar-command-output/SKILL.md` |

## Rules

- Prefer executable sources over prose when they conflict: `Makefile`, `cmd/*`, and `internal/infra/config/*` outrank README/docs. Use `docs/technical_blueprint.md` for architecture intent, but verify against code.
- `Makefile` is the command source of truth. Current targets: `make test`, `make build`, `make fmt`, `make run-health`, `make run-customer-list`, `make run-mcp-http`, `make run-invoice-import FILE=...`.
- `make fmt` only runs `gofmt -w ./cmd ./internal`; if you touch Go files elsewhere, format them explicitly.
- This is a Go 1.25.8 project. Prefer stdlib; require explicit user approval before adding any new external dependency.
- Main entrypoints: `cmd/cli/main.go` wires the CLI; `cmd/mcp-http/main.go` wires the HTTP MCP server.
- Keep boundaries intact: `internal/core` domain types/rules; `internal/app` services, DTOs, and consumed seams; `internal/connectors/*` transport translation; `internal/infra/*` config/logging/SQLite/PDF/export implementations.
- CLI and MCP must call shared `internal/app` services; connectors/infra must not bypass app services for core state changes.
- `cmd/mcp-http` serves `/v1/mcp` plus `/healthz`, requires `MCP_API_KEYS`, and uses `Authorization: Bearer <api-key>` only. Do not assume OAuth/OIDC or stdio MCP exists.
- Config auto-loads `.env`; existing non-empty environment variables win. `MCP_HTTP_LISTEN_ADDR` defaults to `127.0.0.1:8080`; `BILLAR_DB_PATH` defaults to a per-user SQLite DB; `BILLAR_EXPORT_DIR` roots MCP PDF/file outputs.
- `internal/connectors/cli` supports `text`, `json`, and `toon`. Use `OutputResult{Payload, TextWriter}`; keep one canonical DTO for machine formats and add both `json` and `toon` tags on structured output fields.
- Money and durations are integer-backed billing values; never introduce floats for billing calculations. Invoice totals come from invoice lines, not renderers.
- `.atl/`, `.env`, `go.work*`, and `skills-lock.json` are local/ignored metadata; do not intentionally add them unless the user explicitly asks.

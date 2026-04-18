# Local Skills

Use project-local skills before generic guidance.

This file defines repo-wide operating rules; implementation and architecture guidance belongs in the skills below.

| Skill | Use When | Path |
|---|---|---|
| `golang-patterns` | Writing or reviewing Billar Go code, especially `internal/core`, `internal/app`, `internal/connectors`, `internal/infra`, tests, and early scaffolding | `.agents/skills/golang-patterns/SKILL.md` |
| `architecture-billar` | Designing or reviewing package layout, service boundaries, connector flow, storage/auth/rendering boundaries, and blueprint alignment | `.agents/skills/architecture-billar/SKILL.md` |
| `billar-command-output` | Creating or reviewing Billar command outputs, canonical DTOs, CLI format handling, and human-readable text rendering | `.agents/skills/billar-command-output/SKILL.md` |

## Rules

- Prefer executable sources (`Makefile`, `cmd/*`, `internal/infra/config/*`) over prose when they conflict; `docs/technical_blueprint.md` is useful context but can lag behind the code.
- `Makefile` is the source of truth for project commands; prefer documented targets before ad-hoc commands.
- Verified commands: `make test`, `make build`, `make fmt`, `make run-health`, `make run-customer-list`, `make run-mcp-http`.
- `make fmt` only runs `gofmt -w ./cmd ./internal`; if you touch other Go paths, format them explicitly.
- Main entrypoints: `cmd/cli/main.go` (CLI), `cmd/mcp-http/main.go` (HTTP MCP with Bearer API-key auth).
- Keep package boundaries intact: `internal/core` for domain types, `internal/app` for services/DTOs/access logic, `internal/connectors/*` for CLI/MCP transport, `internal/infra/*` for config/logging/SQLite.
- `internal/connectors/cli` supports `text`, `json`, and `toon`; when changing command output, keep the canonical DTO consistent across machine-readable formats.
- `cmd/mcp-http` now requires `MCP_API_KEYS`; it serves `/v1/mcp` with `Authorization: Bearer <api-key>` and no OAuth/OIDC flow.
- Config auto-loads `.env`, but existing non-empty environment variables win.
- `MCP_HTTP_LISTEN_ADDR` defaults to `127.0.0.1:8080`.
- `opencode.json` may contain tool/server config; the HTTP MCP server expects Bearer API keys only — do not assume OAuth is wired.
- Default to project-local skills for Billar-specific work.
- Prefer the Go standard library first before introducing external dependencies.
- Require explicit user approval before adding any new external dependency.

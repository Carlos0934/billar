---
name: architecture-billar
description: >
  Billar architecture guidance for a pragmatic modular monolith with clear layer
  boundaries and connector-friendly application services. Trigger: when designing,
  scaffolding, or reviewing system structure, service boundaries, connectors, storage,
  auth, or rendering decisions in this repo.
license: Apache-2.0
---

## When to Use

- Use this skill for structure and boundary decisions; use `golang-patterns` for Go coding, seams, and tests.
- Creating or reviewing package structure and service boundaries
- Scaffolding new application slices such as health, session, customer, or invoice flows
- Deciding what belongs in core, application, connectors, or infrastructure
- Checking whether an adapter choice fits the blueprint

## Critical Patterns

### Architecture style

- Build Billar as a pragmatic modular monolith.
- Prefer a flat, readable structure over DDD ceremony.
- Keep the internal model explicit and boring.

### Four layers

| Layer | Responsibility |
|---|---|
| `internal/core` | Business types, invariants, validation, money/hours, invoice rules |
| `internal/app` | Use-case orchestration, commands, DTOs, service seams |
| `internal/connectors` | CLI, MCP, and auth-facing adapters |
| `internal/infra` | SQLite, session persistence, OAuth/OIDC, PDF rendering, config |

### Dependency direction

- `core` depends on nothing project-specific below it.
- `app` may depend on `core` and small consumed interfaces.
- `connectors` depend on `app` for operations and DTOs.
- `infra` implements seams required by `app`; it must not own business rules.
- Connectors and infra must not bypass application services to reach core state changes.

### Business logic placement

- Keep business rules in plain Go types and application services.
- Invoice totals come from invoice lines, not renderers.
- Authentication gates access, but auth rules must not leak into billing logic.
- Persistence stores state, but SQLite details stay outside billing logic.

### Connector-friendly design

- CLI and MCP should expose the same use cases through different input/output translation.
- Application services should be shaped so both connectors can call them directly.
- Early scaffolding should preserve this path even for small slices like health or session status.

### Blueprint-specific boundaries

- SQLite is the initial persistence target and belongs in `internal/infra/sqlite`.
- Session/auth belongs behind application services and infra auth/session seams.
- PDF generation belongs behind a renderer boundary; it must not calculate invoice data.
- Use `docs/technical_blueprint.md` as the primary source of truth when skill text and code drift.

## Minimal Example

```text
connectors/cli -> app/service -> core rules
                         |
                         -> infra/sqlite
                         -> infra/auth
                         -> infra/pdf
```

## Commands

```bash
go test ./...
go vet ./...
gofmt -w .
```

## Resources

- Primary source: `docs/technical_blueprint.md`

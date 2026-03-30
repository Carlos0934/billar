# AGENTS.md

Repository guidance for coding agents working in `billar`.

This project is a Go modular monolith following Clean DDD + Hexagonal architecture.
The main baseline is documented in `docs/technical_blueprint.md`.

## Project Snapshot

- Language: Go `1.25.8`
- Module: `github.com/Carlos0934/billar`
- Architecture: Clean DDD + Hexagonal, modular monolith
- Current implemented contexts:
  - `internal/domain/billing/...`
  - `internal/domain/access/...`
  - `internal/application/access/...`
- Persistence/adapters are intentionally not implemented yet in most slices.

## Source of Truth

Before changing behavior, read these first when relevant:

1. `docs/technical_blueprint.md`
2. `internal/domain/...` and `internal/application/...` existing code
3. `AGENTS.md` (this file)

If spec-driven artifacts are in use, align code with the active SDD artifacts first.

## Build / Test / Lint Commands

Run commands from the repository root.

### Full verification

- Run all tests:
  - `go test ./...`
- Run tests uncached:
  - `go test -count=1 ./...`
- Run vet:
  - `go vet ./...`
- Build all packages:
  - `go build ./...`

Recommended full local verification before commit:

1. `gofmt -w $(rg --files -g '*.go')`
2. `go test ./...`
3. `go vet ./...`

### Run a single package

- Billing values:
  - `go test ./internal/domain/billing/billing_values`
- Customer:
  - `go test ./internal/domain/billing/customer`
- Service agreement:
  - `go test ./internal/domain/billing/service_agreement`
- Access session:
  - `go test ./internal/domain/access/session`
- Access application:
  - `go test ./internal/application/access/...`

### Run a single test

Use `-run` with a package path.

- Single test function:
  - `go test ./internal/domain/billing/billing_values -run TestNewHours`
- Single customer test:
  - `go test ./internal/domain/billing/customer -run TestCustomerLifecycleAffectsInvoiceReadiness`
- Single access test:
  - `go test ./internal/application/access/unlock_session -run TestUnlockSessionUnlocksLockedAccessWithValidSecret`

### Run tests with verbose output

- `go test -v ./...`
- `go test -v ./internal/application/access/unlock_session -run TestUnlockSessionUnlocksLockedAccessWithValidSecret`

### Coverage

- Whole repo coverage:
  - `go test -cover ./...`
- Single package coverage:
  - `go test -cover ./internal/domain/billing/billing_values`

## Formatting and Imports

- Always use `gofmt`.
- Do not manually align whitespace.
- Let Go standard import grouping stand unless there is a strong reason otherwise.
- Prefer standard library imports first, then external/module imports as `gofmt` arranges them.
- Avoid unused imports and dead code.

## Naming Conventions

### Packages and folders

- Use explicit bounded-context names.
- Avoid generic folders like:
  - `shared`
  - `common`
  - `models`
  - `types`
- For multi-word folders, prefer `_` if words must be split.
- Never use `-` in Go package or folder names.

Examples used in this repo:

- `billing_values`
- `service_agreement`
- `unlock_session`
- `get_session_status`
- `access_dto`

### Types

- Public constructors should be explicit: `NewX(...)`.
- Use value objects for domain primitives with invariants.
- Keep aggregate-local enums/types inside the aggregate package.
- Put cross-aggregate reusable value objects in the context-specific shared package:
  - here: `internal/domain/billing/billing_values`

### Functions and methods

- Prefer small methods with one clear responsibility.
- Use domain language from the blueprint.
- Avoid vague names like `Process`, `HandleData`, `DoStuff`.

## Architecture Rules

- Follow strict inward dependency flow.
- Domain owns business rules.
- Application orchestrates use cases.
- Adapters must depend inward, never the reverse.
- CLI and MCP should call application use cases only.
- Repositories are per aggregate, not per table.
- Do not leak persistence, transport, or encryption concerns into domain entities.

## Error Handling

- Prefer package-level sentinel errors in `errors.go` for stable validation and transition errors.
- Avoid scattering repeated inline `fmt.Errorf(...)` validation messages when a stable error variable is appropriate.
- Use `errors.Is(...)` in tests for sentinel errors.
- Keep error messages lowercase and without trailing punctuation.
- Wrap errors only when adding useful context.

Examples already present:

- `internal/domain/access/session/errors.go`
- `internal/domain/billing/billing_values/errors.go`
- `internal/domain/billing/customer/errors.go`
- `internal/domain/billing/service_agreement/errors.go`

## Domain Modeling Guidelines

- Entities should contain behavior, not just fields.
- Value objects should be immutable and self-validating.
- Use integer scaled precision for money and hours.
- Do not use floats for monetary or hours domain values.
- Current billing precision baseline:
  - money: 4 decimal digits
  - hours: 4 decimal digits
- Use ISO-style codes where modeled as value objects:
  - `CurrencyCode`
  - `CountryCode`

## Testing Guidelines

- Prefer TDD when adding behavior: RED → GREEN → REFACTOR.
- Write package-external tests (`packagename_test`) for public behavior where practical.
- Test behavior and invariants, not implementation details.
- Add focused unit tests for new value objects, aggregate invariants, and use cases.
- Keep tests deterministic; inject clocks/generators/stores through ports or fakes.
- Prefer in-memory fakes in application-layer tests.

## Current Repository Conventions

- Shared billing primitives live in `internal/domain/billing/billing_values`.
- Access domain state lives in `internal/domain/access/session`.
- Access use cases live in `internal/application/access/...`.
- `CountryCode` is a value object; address country is not a raw string.
- `Hours` is a shared billing value object.

## What Not to Do

- Do not introduce generic shared packages.
- Do not bypass the application layer from adapters.
- Do not use floats for billing values.
- Do not add adapter concerns into domain packages.
- Do not invent new folder naming styles inconsistent with `_` multi-word convention.

## Cursor / Copilot Rules

No repo-local Cursor rules were found:

- `.cursorrules` → not present
- `.cursor/rules/` → not present

No repo-local Copilot instructions were found:

- `.github/copilot-instructions.md` → not present

If those files are added later, update this document to reflect or summarize them.

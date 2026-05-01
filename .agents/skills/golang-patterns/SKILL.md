---
name: golang-patterns
description: >
  Billar-specific Go implementation guidance for package layout, service seams, and
  test-first application code. Trigger: when writing or reviewing Go code in this repo,
  especially services, seams, connectors, tests, or early scaffolding.
license: Apache-2.0

---

## When to Use

- Use this skill for Go code shape and testing; use `architecture-billar` for layer ownership and boundary decisions.
- Writing or reviewing Go code in Billar
- Adding `internal/core`, `internal/app`, `internal/connectors`, or `internal/infra` code
- Scaffolding service slices, including health and billing seams

## Critical Patterns

### Package and module shape

- Follow the blueprint package structure unless there is a concrete reason not to:

```text
cmd/
  cli/

internal/
  core/
  app/
  connectors/
  infra/
```

- Keep core types and rules in `internal/core`.
- Put application services, commands, and DTOs in `internal/app`.
- Put CLI entrypoints in `internal/connectors`.
- Put SQLite, PDF, and config implementations in `internal/infra`.

### Service seams

- Keep business rules in plain Go structs and functions.
- Application services coordinate use cases; they do not become mini-frameworks.
- Define small interfaces in the consuming package when a seam is needed.
- CLI should call application services.
- Do not let SQLite or rendering details leak into core logic.

### Testing and TDD

- Write or update tests first when adding service behavior or core rules.
- Test business rules in `internal/core` with table-driven tests.
- Test application services by mocking only the seams they consume.
- Keep connector tests focused on translation and wiring, not duplicated business logic.
- Prefer fast unit tests before adding integration coverage.

### Boring Go rules

- Prefer plain structs over patterns-heavy abstractions.
- Use integer-backed money and hours types; never use floats for billing values.
- Return early, wrap errors with context, and keep control flow obvious.
- Avoid package-level mutable state.
- Add abstractions only when a second concrete need appears.
- Prefer the standard library first; if an external library is proposed, justify the gap and get user approval per repo policy.

## Minimal Example

```go
package app

type InvoiceReader interface {
    GetByID(ctx context.Context, id string) (core.Invoice, error)
}

type InvoiceInspector struct {
    invoices InvoiceReader
}

func (s InvoiceInspector) Inspect(ctx context.Context, id string) (core.Invoice, error) {
    invoice, err := s.invoices.GetByID(ctx, id)
    if err != nil {
        return core.Invoice{}, fmt.Errorf("inspect invoice: %w", err)
    }
    return invoice, nil
}
```

## Commands

```bash
go test ./...
go test ./internal/...
go test -race ./...
go vet ./...
gofmt -w .
```

## Resources

- Primary source: `docs/technical_blueprint.md`

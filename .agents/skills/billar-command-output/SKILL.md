---
name: billar-command-output
description: >
  Billar-specific guidance for creating CLI command outputs with the canonical DTO,
  formatter strategy, and text override pattern. Trigger: when adding or reviewing
  command output models, CLI output formatting, text views, or new output-bearing
  commands in this repo.
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## When to Use

- Use this skill when adding or reviewing Billar command outputs, output DTOs, or CLI formatting behavior.
- Use `architecture-billar` for broader layer ownership and package boundaries.
- Use `golang-patterns` for general Go implementation, seams, and tests.

## Critical Patterns

### Output ownership

- Command output shape belongs at the connector boundary, not in `internal/core`.
- Keep business rules in core and application orchestration in `internal/app`; output translation belongs in `internal/connectors`.
- Application services may return canonical DTOs for connector consumption, but connector-specific text rendering stays out of core.

### Command result model

- Use one canonical payload for structured formats.
- `json` and `toon` must serialize the same canonical DTO.
- `text` may use an optional custom text override when the generic fallback is not clear enough.
- Match the current Billar pattern: canonical DTO plus optional `TextWriter` override.

### Allowed formats

| Format | Purpose | Source of truth |
|---|---|---|
| `text` | Human-first CLI output | Custom `TextWriter` or generic text fallback |
| `json` | Stable machine-readable output | Canonical payload |
| `toon` | Tag-based structured output | Canonical payload with `toon` tags |

### Text fallback vs custom text

Use generic text fallback when:

- The payload is a simple scalar.
- The payload is a small flat struct or string-keyed map.
- Field-by-field output like `name: value` is already readable enough.

Provide custom text when:

- The command needs a more human-oriented summary or label.
- Field ordering, wording, grouping, or omission matters for readability.
- The payload is a list, nested structure, or otherwise awkward for the generic fallback.
- The command should present a future table or multi-line view for humans.

### Design patterns to keep

- Canonical DTO plus optional text override.
- Formatter strategy selected by output format.
- Small builder/helpers for human text views; keep them boring and connector-local.
- Do not fork separate DTOs per structured format unless there is a real external compatibility need.

## Recommended Practices

- Use stable field names for canonical DTOs because `json` and `toon` share them.
- Add both `json` and `toon` tags on structured output fields.
- Keep DTOs simple, explicit, and easy to serialize.
- Prefer flat DTOs first; only add nesting when it materially improves the contract.
- Treat canonical payloads as the long-lived contract for machine-readable output.
- Keep fallback text simple; do not turn it into a mini rendering framework.
- For future list or table commands, keep the canonical payload machine-friendly and add custom `text` rendering for the human view.
- Follow the `Makefile` for normal repo commands, but do not assume one make target per output format.

## Key Learnings

- Billar currently supports exactly `text`, `json`, and `toon`.
- `toon` output only works predictably when DTO fields include explicit `toon` tags.
- `json` and `toon` should stay aligned through the same canonical payload.
- The generic text fallback is intentionally narrow and should remain simple.
- The current output strategy and small text builder are enough for now; extend them incrementally instead of introducing a larger rendering abstraction.

## Minimal Example

```go
type StatusDTO struct {
    Name   string `json:"name" toon:"name"`
    Status string `json:"status" toon:"status"`
}

result := cli.OutputResult{
    Payload: StatusDTO{Name: "billar", Status: "ok"},
    TextWriter: func(w io.Writer) error {
        _, err := io.WriteString(w, "Billar Health\n")
        return err
    },
}
```

## Commands

```bash
make test
make fmt
make run-health
```

## Resources

- Output formatter path: `internal/connectors/cli/output.go`
- Example DTO: `internal/app/health_service.go`
- Primary source: `docs/technical_blueprint.md`

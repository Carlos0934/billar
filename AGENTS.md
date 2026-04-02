# Local Skills

Use project-local skills before generic guidance.

This file defines repo-wide operating rules; implementation and architecture guidance belongs in the skills below.

| Skill | Use When | Path |
|---|---|---|
| `golang-patterns` | Writing or reviewing Billar Go code, especially `internal/core`, `internal/app`, `internal/connectors`, `internal/infra`, tests, and early scaffolding | `.agents/skills/golang-patterns/SKILL.md` |
| `architecture-billar` | Designing or reviewing package layout, service boundaries, connector flow, storage/auth/rendering boundaries, and blueprint alignment | `.agents/skills/architecture-billar/SKILL.md` |
| `billar-command-output` | Creating or reviewing Billar command outputs, canonical DTOs, CLI format handling, and human-readable text rendering | `.agents/skills/billar-command-output/SKILL.md` |

## Rules

- `docs/technical_blueprint.md` is the primary architecture source of truth.
- `Makefile` is the source of truth for project commands; prefer documented make targets before ad-hoc commands.
- Default to project-local skills for Billar-specific work.
- Prefer the Go standard library first before introducing external dependencies.
- Require explicit user approval before adding any new external dependency.

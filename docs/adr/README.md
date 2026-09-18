# adr-rg ADRs

This directory contains the architectural decisions for `adr-rg` itself.

## Current decisions

| ID | Decision | Status |
| --- | --- | --- |
| ADR-0001 | Adopt Go for the adr-rg CLI | adopted |
| ADR-0002 | Markdown ADRs with YAML frontmatter | adopted |
| ADR-0003 | Repository layout and generated index | adopted |
| ADR-0004 | GoReleaser and GitHub Actions releases | adopted |
| ADR-0005 | Explicit rationale for durable decisions | adopted |
| ADR-0006 | JSON as canonical structured output | adopted |
| ADR-0007 | Source installation and language scaffolding | adopted |

## Working with ADRs

```bash
adr new --lang go --tag tooling --path "internal/**" "Decision title"
adr check --strict
adr index
adr review --base main
adr scaffold rust
```

`index.yaml` is generated; use `adr index` rather than editing it manually. ADR
retained decisions remain queryable through the generated index.

## Required sections

Every ADR contains `Context`, `Decision`, `Alternatives Considered`, and
`Consequences`. Frontmatter may additionally declare constraints, invariants, and
scope paths used by `adr review` (`detect-bs` remains a compatibility alias).

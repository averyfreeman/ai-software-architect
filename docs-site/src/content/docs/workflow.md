---
title: Workflow
description: The day-to-day workflow for creating and validating ADRs.
---

## Create a decision

```bash
adr init
adr new --tag database --tag storage --path "src/db/**" --path "migrations/**" "Use PostgreSQL for persistence"
```

ADRs live in `docs/adr/` by default. Scope paths tell reviewers which decisions may matter for a code change.

## Review applicable decisions

```bash
adr review --base main
adr list --path "src/db/users.go"
adr show ADR-0001
```

The detector is a review queue based on scope globs, not a semantic proof that the implementation complies.

## Validate and index

```bash
adr check --strict
adr index --check
```

The index is generated metadata. Keep ADR Markdown as the source of truth and check the generated index in CI.

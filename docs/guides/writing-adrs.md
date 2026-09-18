# Writing ADRs

An ADR should let a future engineer understand the decision without excavating
the original meeting or pull request.

## When to write one

Write an ADR when a decision:

- affects architecture or a long-lived interface;
- carries meaningful trade-offs or operational consequences;
- establishes rules for multiple contributors; or
- should be revisited through an explicit superseding decision.

Routine implementation details usually do not need their own record.

## Required body

Every ADR contains:

1. `Context`: the problem, forces, and decision drivers;
2. `Decision`: what the project chose and why;
3. `Alternatives Considered`: credible options and why they lost;
4. `Consequences`: positive, negative, and follow-up effects.

## Metadata that earns its keep

Use a stable ID, a lifecycle status, and a narrow scope:

~~~yaml
adr_id: ADR-0001
title: "Use a repository boundary for persistence"
status: adopted
date: 2026-01-16
scope:
  paths:
    - "src/db/**"
tags:
  - database
constraints:
  - "Database access goes through the repository boundary"
invariants:
  - "Migrations remain reversible"
supersedes: []
superseded_by: []
related_adrs: []
~~~

Scope paths are globs used by `adr review` (`detect-bs` is a compatibility
alias). Keep them narrow enough to identify the code that genuinely deserves
review.

## Constraints and invariants

A constraint is a rule the implementation must follow. An invariant is a
property that must remain true as the system evolves.

Prefer language a reviewer can check:

- “Database access goes through the repository boundary.”
- “Every request has a trace identifier.”

Avoid turning vague preferences into mandatory rules. A decision with too many
constraints becomes a speed bump nobody trusts.

## Rationale

Tie reasons to the decision drivers and name the trade-offs you accepted. Strict
validation may require explicit rationale headings for the adopted option and
rejected alternatives.

## Supersede; do not rewrite

When a decision changes, create a new ADR and connect both records:

~~~bash
adr new --paths "src/db/**" "Replace the persistence boundary"
adr check --strict
adr index
~~~

Update `supersedes` and `superseded_by`, then regenerate the index. The old
record is part of the project’s memory.

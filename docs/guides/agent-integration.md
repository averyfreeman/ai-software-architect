# Tool integration

adr-rg gives coding tools and plugins a small, stable JSON surface for reading
decisions before a change and reviewing them afterward.

## Before implementation

~~~bash
adr list --format json
adr show --format json ADR-0001
adr review --base main --json
~~~

Use `list` for discovery, `show` for the full decision context, and
`review` for path-scoped review context. (`detect-bs` remains an alias.) Review
results include:

- `changed_files`;
- `applicable_adrs`;
- matched scope paths and files; and
- the applicable constraints and invariants.

The caller still has to read the ADR and evaluate the implementation. A path
match is a prompt to review, not proof of a violation.

## After implementation

~~~bash
adr review --base main --json
adr check --strict
adr index --check
~~~

Prefer JSON for scripts and plugin calls, text for interactive review, and
version-controlled ADRs plus the generated index for durable context.

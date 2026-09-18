# Why adr-rg?

Important decisions tend to get scattered across pull requests, chat threads,
and people’s memory. adr-rg gives them a small, repository-native home:
Markdown for people, YAML metadata for tools, and Git history for context.

## Structure that pays rent

An ADR becomes more useful when it says not only what was chosen, but where the
choice applies and what later work must preserve.

- Stable IDs make decisions easy to cite.
- Status and supersession links preserve lifecycle history.
- Scope paths connect a record to the code it governs.
- Constraints and invariants expose the durable implications.
- A generated index makes the decision set discoverable.
- Required sections keep the record understandable.

## Shared context, less rediscovery

People and coding tools can query a bounded slice of repository context:

~~~bash
adr list --path "src/db/users.go"
adr show ADR-0001
adr review --base main --json
~~~

The result is a better starting point, not an architecture oracle. The caller
still reads the decision, weighs the trade-offs, and reviews the code.

## Keep it proportional

Use an ADR for a durable choice, not every implementation detail. Keep scope
patterns narrow and constraints concrete. adr-rg validates document structure
and reports applicability; it is not a policy engine or a substitute for tests,
static analysis, or engineering judgment.

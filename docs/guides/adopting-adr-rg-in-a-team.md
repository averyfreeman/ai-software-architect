# Adopt adr-rg without the ceremony

adr-rg works best as a lightweight repository convention, not a gate on every
line of code.

## Start small

- Add the template and record genuinely durable decisions.
- Keep scope paths narrow and constraints concrete.
- Review ADRs like other design documentation.

## Make it part of the loop

- Ask contributors and tools to run `adr list` or `adr show` before editing.
- Run `adr review --base main` during review (`detect-bs` remains an alias).
- Use `--format json` when another tool consumes the result.

## Automate the boring checks

- Run `adr check --strict` in CI.
- Run `adr index --check` to catch stale generated metadata.
- Ask for explicit human review when the Bullshit Detector reports an applicable
  decision.

The detector tells you where to look. It should not turn every path match into a
bureaucratic approval process. Create a new ADR when the decision is durable,
not for every implementation detail.

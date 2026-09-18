# CI integration

Use adr-rg in CI to validate ADR structure, catch stale indexes, and surface
decisions that deserve review.

## Recommended checks

~~~bash
go build ./cmd/adr
adr check --strict
adr index --check
adr review --base origin/main --json
~~~

The first two checks validate the records. The index check verifies committed
generated metadata. The Bullshit Detector produces a review signal from Git path
scope; it does not inspect source semantics.

## GitHub Actions

~~~yaml
name: ADR governance

on: [push, pull_request]

jobs:
  adr-rg:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go build -o adr ./cmd/adr
      - run: ./adr check --strict
      - run: ./adr index --check
~~~

Install the CLI with `go install` or build it from source in production CI. If
another tool needs applicability data, consume `--format json`.

## Exit codes

- `0`: success; non-strict validation warnings may exist;
- `1`: command, Git, or input error;
- `2`: validation failure, strict-mode warning, or stale index.

CI can require valid records, a current index, or human review. Semantic
enforcement needs a separate policy or static-analysis system.

# Contributing

The production implementation is Go-only.

## Local checks

Use Go 1.26 or newer:

    gofmt -w .
    go test ./...
    go test -race ./...
    go vet ./...
    go run ./cmd/generate-schemas --check
    make build-all

The Codex plugin is assembled with cmd/codex-plugin. Do not add Python, uv, PyInstaller, generated fallback imports, or a second contract implementation to production code.

## Contract changes

Edit internal/architect first, then run the schema generator. Checked-in schemas must be unchanged after:

    go run ./cmd/generate-schemas
    go run ./cmd/generate-schemas --check

Only schema version 1.1.0 is accepted by production validators.

## Scope

Keep application source immutable during the architecture workflow. Native hooks may write only validated .ai-architect artifacts after explicit approval. Preserve historical evaluation ledgers and release evidence without rewriting their records.

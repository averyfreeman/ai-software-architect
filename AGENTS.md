# Repository instructions

This repository has one production implementation: Go.

- Treat internal/architect as the canonical contract and deterministic-analysis authority.
- Keep the five Codex hook events fail-closed and preserve the two supported runtime invocations.
- Generate checked-in JSON Schemas with go run ./cmd/generate-schemas and keep the generation check clean.
- Use the Makefile for native builds and Go cmd/codex-plugin for packaging.
- Do not add Python, uv, PyInstaller, fallback imports, legacy schema branches, or compatibility aliases to production paths.
- Preserve append-only evaluation ledgers and release evidence as historical records.
- Use apply_patch for file edits, run gofmt, and validate with go test ./..., go test -race ./..., and go vet ./....

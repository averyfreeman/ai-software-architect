# Development handoff

This repository has completed the production cutover to Go.

The canonical deterministic layer is internal/architect. It owns schema 1.1.0 contracts, strict YAML decoding, ADR and bundle validation, artifact secret scanning, workspace boundaries, dependency analysis, rendering, and bounded repository snapshots. internal/codex owns explicit routing, durable continuation state, response validation, artifact preflight, post-write verification, compaction restoration, and fail-closed handling for all five hook events.

Build and package with the Go commands documented in CONTRIBUTING.md and RELEASING.md. The package includes only native binaries and bundled skills. Python evaluation files and examples may remain as development material, but they are not imported, packaged, or required by product CI.

The supported native targets are windows/amd64, darwin/arm64, linux/amd64, and linux/arm64. Linux binary smoke tests run on Linux; Linux Codex host lifecycle support is intentionally not claimed until a separate host validation campaign is complete.

Preserve append-only evaluation ledgers and release evidence as historical records. Current behavior, schemas, manifests, and release checks are owned by Go sources and checked-in generated artifacts.

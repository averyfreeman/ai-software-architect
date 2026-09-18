# AI Software Architect

AI Software Architect is a local Codex plugin for evidence-based architecture decisions, ADR authoring, and coding handoff. The production implementation is Go-only and uses short-lived native binaries for deterministic validation and hook control.

## Development

Requirements are Go 1.26 or newer and a host build toolchain. No Python, uv, package download, or persistent service is required.

- go test ./...
- go test -race ./...
- go vet ./...
- go run ./cmd/generate-schemas --check
- make build-all
- make smoke-linux

The Linux lifecycle is smoke-tested on a native Linux runner. Linux plugin lifecycle support is not advertised until a separate host validation campaign is complete.

## Native plugin targets

The release matrix builds:

- Windows x86-64: windows/amd64
- macOS Apple Silicon: darwin/arm64
- Linux x86-64: linux/amd64
- Linux ARM64: linux/arm64

Build a runtime with make build-linux-amd64 or package a plugin with:

    go run ./cmd/codex-plugin package --target linux-amd64

The packager injects VERSION into the runtime and plugin manifest, validates the extracted layout, and writes a SHA-256 sidecar.

## Runtime interface

The native runtime intentionally exposes only:

- --codex-hook --event UserPromptSubmit, PreToolUse, PostToolUse, PostCompact, or Stop
- --repository-snapshot --root .

Hook input and output are bounded JSON. The runtime does not start a server, execute repository code, or perform network access.

## Layout

- internal/architect: canonical contracts, schema generation, workspace safety, analysis, artifacts, rendering, and snapshots.
- internal/codex: routing, continuation state, response validation, and five-event hook control.
- cmd/ai-architect-runtime: short-lived native Codex runtime.
- cmd/codex-plugin: cross-build, assembly, validation, and ZIP packaging.
- shared/skills: user-facing workflow content bundled into the plugin.

See docs/INSTALL_CODEX_PLUGIN.md and docs/RELEASING.md for installation and release procedures.

# Releasing

The release source of truth is VERSION. The Go packager injects that value into every native runtime and plugin manifest.

## Checks

Run:

    go test ./...
    go test -race ./...
    go vet ./...
    go run ./cmd/generate-schemas --check
    make build-all
    make smoke-linux

On a native Linux runner, execute the Linux runtime and validate a package after extraction. Windows reparse-point and macOS path fixtures belong in their native host jobs. Do not advertise Linux Codex host lifecycle support until that host campaign passes.

## Release matrix

The release workflow packages all four targets:

- windows-amd64
- darwin-arm64
- linux-amd64
- linux-arm64

For a local package:

    go run ./cmd/codex-plugin package --target linux-amd64 --version "$(sed -n '1p' VERSION)"

The command writes a ZIP and a SHA-256 sidecar, validates the staged layout, and rejects missing hooks, extra runtimes, or malformed metadata.

## Clean-machine requirement

Installation and binary execution must work without Python, uv, PyInstaller, network downloads, or project dependencies. The only runtime entry points are the five Codex hook events and the bounded repository snapshot command.

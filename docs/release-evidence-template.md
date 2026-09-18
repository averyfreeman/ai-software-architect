# Native release evidence

Release version: VERSION

## Required automated evidence

- [ ] go test ./...
- [ ] go test -race ./...
- [ ] go vet ./...
- [ ] go run ./cmd/generate-schemas --check
- [ ] native builds for windows/amd64, darwin/arm64, linux/amd64, and linux/arm64
- [ ] Linux amd64 binary smoke test on a native Linux runner
- [ ] package extraction and Go validator checks for every target
- [ ] SHA-256 sidecars match the uploaded archives

## Clean-machine evidence

- [ ] Windows x86-64 installs and launches without Python, uv, PyInstaller, or network downloads
- [ ] macOS Apple Silicon installs and launches without Python, uv, PyInstaller, or network downloads
- [ ] Linux binaries execute the bounded snapshot command without Python or network downloads
- [ ] Linux Codex host lifecycle remains unadvertised until separately verified
- [ ] Windows reparse-point fixtures pass
- [ ] macOS path and Linux filesystem fixtures pass

## Manual workflow evidence

- [ ] explicit selection and clarification routing
- [ ] approval persistence and one-shot continuation
- [ ] static inspection and source immutability
- [ ] artifact preflight secret scanning and complete bundle validation
- [ ] post-write verification and fail-closed failures
- [ ] compaction checkpoint restoration

Historical release evidence is append-only and must not be rewritten to make old records look current.

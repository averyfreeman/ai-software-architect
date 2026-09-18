# Plugin submission notes

The submission artifact is assembled by cmd/codex-plugin from the Go runtime, the five-hook manifest, bundled skills, and the icon asset.

Validate an extracted package with:

    go run ./cmd/codex-plugin validate --target linux-amd64 ./extracted-plugin

The manifest version comes from VERSION. The package has exactly one native runtime, no Python or uv dependency, and no persistent process. The Linux host lifecycle remains unsupported until its separate native-host campaign is complete.

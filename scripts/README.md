# Scripts

Product build and release operations are implemented by Go:

    make build-all
    make smoke-linux
    go run ./cmd/codex-plugin package --target windows-amd64
    go run ./cmd/codex-plugin validate --target windows-amd64 ./extracted-plugin

PowerShell scripts that remain in this directory support development-only evaluation history workflows. They are not part of product CI, release packaging, or clean-machine installation, and they must not be copied into a plugin archive.

Use VERSION as the sole release version source. Do not add compatibility aliases, Python runtime dependencies, uv commands, or PyInstaller steps to product workflows.

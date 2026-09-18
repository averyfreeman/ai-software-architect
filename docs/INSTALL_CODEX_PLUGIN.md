# Install the Codex plugin

Release archives are native Go packages. Choose the archive for the host:

- Windows x86-64: windows-amd64
- macOS Apple Silicon: darwin-arm64
- Linux x86-64: linux-amd64
- Linux ARM64: linux-arm64

Verify the adjacent .sha256 file with the host checksum tool, extract the archive, and install the extracted directory through Codex's local plugin flow. The archive contains .codex-plugin/plugin.json, hooks/hooks.json, one composite skill under skills/ai-software-architect/, legal notices, assets/, and exactly one native runtime.

The runtime needs no Python, uv, PyInstaller, network access, virtual environment, or first-run download.

To inspect a package before installation:

    go run ./cmd/codex-plugin validate --target linux-amd64 ./extracted-plugin

The Linux runtime can execute the bounded repository snapshot command. Linux Codex host lifecycle support remains unadvertised until separately verified on the host.

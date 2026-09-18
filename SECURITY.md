# Security policy

The production control plane is implemented in Go and is designed to fail closed.

- Repository content is treated as untrusted data; the runtime never imports, executes, compiles, builds, or tests analyzed project code.
- Shell inspection is limited to static allowlisted reads and the bounded repository snapshot command.
- Writes are limited to validated .ai-architect artifacts. Proposed content is reconstructed, secret-scanned, schema-validated, and checked again after persistence.
- Workspace paths reject traversal, protected credential locations, symlink escapes, and platform reparse-point indirection.
- Hook input, state, snapshots, and generated outputs have fixed size and lifetime bounds.

Report security issues privately to the repository maintainers rather than opening a public issue with exploit details.

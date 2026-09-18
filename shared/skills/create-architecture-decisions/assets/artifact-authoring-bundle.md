# Architecture artifact authoring bundle

After explicit approval, prepare one complete reviewable bundle:

- .ai-architect/project-context.md
- .ai-architect/architecture-contract.yaml
- .ai-architect/implementation-plan.md
- .ai-architect/decisions/ADR-NNN[-slug].md

Use schema version 1.1.0. Preserve the nested contract and ADR shapes from the examples in this skill. The trusted hook reconstructs the proposed contents, scans for secrets, validates every contract and ADR, cross-validates decision identifiers, and verifies persisted bytes after the write. Never modify application source.

---
title: Commands and model
description: Command reference and the ADR record shape.
---

| Command | Purpose |
| --- | --- |
| `adr init` | Create the ADR directory, template, and generated index |
| `adr new "Title"` | Create a numbered ADR |
| `adr list` | List records with status, tag, and scope filters |
| `adr show <id>` | Show one ADR’s metadata and governance fields |
| `adr check` | Validate ADR files (`check adr` remains compatible) |
| `adr index` | Generate or check the YAML index |
| `adr review [--base <ref>]` | Find ADRs applicable to changed Git paths (`detect-bs` is an alias) |

Each record combines YAML frontmatter with `Context`, `Decision`, `Alternatives Considered`, and `Consequences` sections. Use constraints for implementation rules and invariants for properties that must remain true.

JSON is available for automation:

```bash
adr list --format json
adr show --format json ADR-0001
```

---
adr_id: ADR-0007
title: "Source Installation and Language Scaffolding"
status: adopted
date: 2026-09-15
scope:
  paths:
    - "cmd/**"
    - "internal/**"
    - "docs/templates/**"
    - "docs/guides/**"
    - "docs/reference/**"
    - "README.md"
    - "SPEC.md"
    - "Makefile"
    - "go.mod"
    - "demo/**"
    - "scripts/**"
    - "tools/**"
tags:
  - scaffolding
  - installation
  - cli
  - workflow
constraints:
  - The supported Go toolchain is Go 1.26 or later
  - Source installation must be documented through go install and the Makefile
  - Scaffold generation must not perform Git, remote, or network side effects
  - review is the canonical command name for scope applicability review; detect-bs remains an alias
  - Scaffold templates must work from an installed binary without repository-relative files
invariants:
  - Generated scaffold Git defaults remain editable in .adr-scaffold.yaml
  - Existing scaffold files are never overwritten silently unless --force is provided
  - Language aliases resolve to one deterministic canonical profile
  - JSON scaffold output remains deterministic and machine-readable
supersedes: []
superseded_by: []
related_adrs:
  - ADR-0001
  - ADR-0004
  - ADR-0006
---

# ADR-0007: Source Installation and Language Scaffolding

## Context

The repository’s release-download installer scripts depended on archive links
that were no longer reliable. The project also needs a quick, repeatable way to
create agent-oriented repository guidance across several programming languages.
That workflow must remain safe for existing repositories and usable when `adr`
is installed outside this checkout.

Decision drivers:

- Make local Go development and installation obvious.
- Keep scaffolding offline and deterministic.
- Preserve user files and avoid surprising Git or remote mutations.
- Make the language catalog extensible without adding a CLI framework.

## Decision

### Source-first installation with embedded scaffolds: Adopted

**Adopted because:**

- `go install` and the Makefile use the Go toolchain already required to build
  the project.
- Embedded assets allow an installed binary to scaffold without depending on
  the source checkout or broken download URLs.
- A catalog separates language aliases and development guidance from command
  dispatch code.
- Interactive collision handling and `--force` make replacement explicit.
- `.adr-scaffold.yaml` exposes Git habits without executing them.

**Adopted despite:**

- Users without Go must use the existing GoReleaser release artifacts.
- Embedded templates increase the binary’s source asset set.
- Interactive prompts require a separate non-interactive `--force` path.

The CLI uses Go 1.26 or later. `review` is the canonical scope review command;
`detect-bs` remains a compatibility alias. Tar-download installer scripts are
removed; release archives and checksums remain GoReleaser outputs.

## Alternatives Considered

### Release-download installer scripts: Rejected

**Rejected because:**

- The existing links were broken and required platform-specific shell logic.
- Archive naming and checksum handling duplicated GoReleaser behavior.
- The scripts created a second installation contract to document and maintain.

**Rejected despite:**

- They could install a binary without a local Go toolchain.
- They offered repo-local installation for demos.

### Git-automating scaffold command: Rejected

**Rejected because:**

- Initialization, commits, tags, pushes, and remote creation are consequential
  operations that require project-specific approval.
- A local file generator is safe to run in an existing repository.

**Rejected despite:**

- Fully automated repository setup would be faster for a brand-new project.
- The Rust reference workflow already describes optional GitHub CLI setup.

### External template checkout: Rejected

**Rejected because:**

- It would make scaffolding dependent on network access and a moving checkout.
- Installed binaries would not have a stable template source.

**Rejected despite:**

- External repositories could grow the language catalog independently.
- Users could customize templates without rebuilding the CLI.

## Consequences

**Positive:**

- `make build`, `make test`, `make vet`, and `make install` provide a clear local workflow.
- `adr scaffold --lang rust` and language aliases work offline.
- Existing files are protected by default and replacement choices are visible.
- Git habits are reviewable and editable as ordinary project configuration.

**Negative:**

- Template changes require a new CLI build or release.
- The catalog must be kept current as language tooling changes.
- Users must explicitly perform Git setup after scaffolding.

## Agent Guidance

- Do not reintroduce tar-download installer scripts without superseding this ADR.
- Add a language profile to the embedded catalog before documenting it as supported.
- Keep scaffold operations local and side-effect free.
- Run `adr index` after changing ADR metadata.

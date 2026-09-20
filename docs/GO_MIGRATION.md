# Go consolidation track

## Git BBQ implementation track

The current clean-rewrite direction is Git BBQ: a Go-first scaffold and Codex
hook process that treats Matt Pocock skills as the semantic architecture
authority. The public binary is `git-bbq`, the public plugin identity is `Git
BBQ`, and the Go module is `github.com/averyfreeman/git-bbq`.

Git BBQ writes Matt-native project files rather than a second architecture
document set:

- `.gitbbq-manifest.yaml` stores project problem, selected languages, dependency
  pin, and hook state.
- `.githabits.yaml` stores granular Git lifecycle policy.
- `CONTEXT.md` and `docs/adr/0001-slug.md` are semantic sources of truth.
- `architecture-contract.yaml`, `implementation-plan.md`, and
  `docs/adr/index.json` are generated projections.
- `.agents/skills/` contains scaffold-owned githabits and language skills.
- `.agents/mattpocock/` is the pinned Matt dependency location for generated
  projects.

The initial Git BBQ implementation is intentionally additive while the existing
AI Software Architect migration is still dirty: the new package and CLI are
tested independently, and the old `.ai-architect` path remains untouched until
the clean cutover is explicitly applied.

The legacy AI Software Architect migration also has a Go implementation. Its
compatibility binary is `ai-architect`; Git BBQ's canonical public binary is
`git-bbq`. Keep the legacy path isolated while new scaffolding and lifecycle
work moves through Git BBQ.

## Legacy compatibility quick path

```sh
make go-check
make go-build
bin/ai-architect questions --format json
bin/ai-architect setup --answers answers.json --dir /path/to/project --format json
bin/ai-architect decision new --dir /path/to/project --title "Choose a boundary"
bin/ai-architect decision check --dir /path/to/project
bin/ai-architect validate-bundle --dir /path/to/project --format json
```

The setup answer file contains free-text project motivation plus tri-state options:
`yes`, `no`, or `make-default`. The five motivation answers are required for every
project and are copied into every new ADR. `auto_reason: yes` permits provisional
agent-filled placeholders, but the generated warning requires human review before
acceptance.

## Generated project contract

- `.adr-scaffold.yaml` controls language-aware generated docs, skills, ADRs,
  project context, implementation plan, Pages workflow, and future memory.
- `.githabits.yaml` controls Git initialization, commit/tag/push preferences,
  SemVer, and optional `gh repo create` metadata.
- `.ai-architect/project-context.md` is the project motivation source.
- `.ai-architect/architecture-contract.yaml` is the machine-readable boundary
  starter.
- `.ai-architect/decisions/ADR-NNN[-slug].md` is the decision source of truth.
- `.ai-architect/decisions/index.json` is the generated retrieval index.
- `AGENTS.md` and `.agents/skills/` contain operating procedure and focused
  instructions; they are intentionally not merged with ADR history.

Existing files are preserved by default. `--force` is an explicit overwrite
request and should be used only after reviewing the generated paths.

## MCP boundary

`bin/ai-architect-mcp` uses the official Go MCP SDK over STDIO. It exposes five
read-only, intent-specific tools: contract validation, artifact scanning,
complete four-artifact bundle validation, decision listing, and bounded
inline-source dependency analysis. The bundle check requires accepted ADRs to
match the contract's `decision_ids`, bounds narratives, and redacts suspected
secret values from diagnostics. It does not accept an arbitrary absolute root,
execute repository code, use a network, or write files. The CLI and host-native
approval flow remain the write path.

This is deliberate. The available third-party projects named `codebase-mcp` are
not a single stable standard, and a broad write-capable MCP would duplicate host
permissions and enlarge the prompt-injection boundary. Add a write tool only with
a separate threat model, idempotency contract, approval parameter, and evaluation
coverage.

## Current migration state

The first exact-parity slice is complete for the architecture contract and ADR
artifact boundary. Go now models unresolved clarification questions and optional
null fields, rejects unknown YAML fields, enforces the Pydantic collection,
text-length, uniqueness, identifier, and cross-reference rules, and keeps legacy
1.0.0 ADRs valid while requiring the motivation, matrix, and action-contract
fields introduced by 1.1.0. Canonical nested-contract, legacy-record, current-
record, boundary, and multi-category secret-scan regression tests cover the
slice.

The Python schemas, Codex hooks, packaging runtime, and legacy Python MCP package
remain in place for compatibility. Git BBQ now covers the deterministic migration,
projection artifact, ownership, uninstall, and five-event lifecycle parity gate;
the remaining gates are the separate exploratory evaluation evidence, a Go Codex
runtime, clean-machine packaging, and behavioral evaluation before Python removal.
Bundle validation does not yet replace the Codex pre-write hook.

# Merge action plan: agent-first architecture scaffold

Status: Git BBQ lifecycle slices 1–14 are complete through `git-bbq-v0.1.13`;
the deterministic M-15 migration, artifact, ownership, hook, and uninstall
parity gate is verified in the current worktree, while the M-15 exploratory
evaluation gate and staged Go migration remain in progress. The roadmap has a fixed 17-slice
denominator: completed work reduces the remaining count; it never adds a new
slice merely because implementation details become visible.

This document is the durable handoff for merging `adr-repo-governance` and
`ai-software-architect`. It records what was implemented, why some requested
capabilities are intentionally not defaults, and the gates that still prevent a
complete Python removal or unattended publication.

## Slice ledger

The previous seven-item unfinished-work list did not record completed increments,
so it is retired. The active ledger is:

| Slice | Scope | Status / exit evidence |
| --- | --- | --- |
| GBBQ-01 | Go scaffold and githabits policy foundation | Complete: `git-bbq-v0.1.0` |
| GBBQ-02 | Read-only action planning | Complete: `git-bbq-v0.1.1` |
| GBBQ-03 | Approval-gated action execution | Complete: `git-bbq-v0.1.2` |
| GBBQ-04 | Repository init and branch execution | Complete: `git-bbq-v0.1.3` |
| GBBQ-05 | Approval-gated remote setup | Complete: `git-bbq-v0.1.4` |
| GBBQ-06 | Persist configured remote state | Complete: `git-bbq-v0.1.5` |
| GBBQ-07 | End-to-end approved push verification | Complete: `git-bbq-v0.1.6` |
| GBBQ-08 | SemVer policy and annotated-tag execution | Complete: `git-bbq-v0.1.7` |
| GBBQ-09 | Release progression and observed-tag protection | Complete: `git-bbq-v0.1.8` |
| GBBQ-10 | Approval-gated GitHub provider provisioning | Complete: `git-bbq-v0.1.9` |
| M-11 | Python deterministic domains and Codex hook/control-plane parity | Complete: `git-bbq-v0.1.10` |
| M-12 | Fixture parity for contracts, protected paths, Windows behavior, and hook protocol | Complete: `git-bbq-v0.1.11` |
| M-13 | Go-based Codex packaging, release, notices, and clean-machine install tests | Complete: `git-bbq-v0.1.12` |
| M-14 | Legacy scaffold language-profile parity for Java and C# | Complete: `git-bbq-v0.1.13` |
| M-15 | Approved migration command plus exploratory evaluations, uninstall, immutability, artifact, and lifecycle gates | In progress: deterministic migration/artifact/ownership/hook/uninstall parity is verified in the current worktree; exploratory evaluation evidence remains |
| M-16 | Retrieval benchmark gate: SQLite FTS5 first, sqlite-vec only if justified | Deterministic benchmark implemented: ranking, latency, context reduction, rebuild, stale-document, and corruption-recovery evidence is generated from the versioned fixture; sqlite-vec is not selected |
| M-17 | Python runtime retirement after every preceding gate passes | Not ready: M-15 exploratory evidence, Go Codex host runtime, package/clean-machine, lifecycle, behavioral, and candidate release-evidence gates remain; compatibility path is retained |

The next active slice is `M-15`, the approved migration command plus exploratory
and lifecycle gates. It is already counted below. The deterministic migration,
artifact, ownership, hook, and uninstall parity portion is complete in this
worktree; M-15 remains active until its separate exploratory evaluation evidence
is recorded. The M-16 benchmark is now the maintained evidence path for the
retrieval gate; it does not authorize a vector dependency without a measured
material improvement.

The M-17 readiness assessment is recorded in
`docs/M17_RUNTIME_RETIREMENT_READINESS.md`. The Go core and deterministic M-16
baseline do not authorize Python removal: M-15 exploratory evidence, the Go
Codex host runtime, package and clean-machine acceptance, Codex Desktop lifecycle,
behavioral evaluation, and exact-candidate release evidence remain outstanding.
The Python schemas, hooks, packaging runtime, legacy MCP package, and adapters
remain in place until those gates pass.

Three concrete slices remain after `M-14`. This count is the fixed remainder
of the 17-slice roadmap, not a list that grows when a slice is decomposed:

1. `M-15` Approved migration command plus exploratory evaluations, uninstall, immutability, artifact, and lifecycle gates.
2. `M-16` Retrieval benchmark gate: SQLite FTS5 first, sqlite-vec only if justified.
3. `M-17` Python runtime retirement after every preceding gate passes.

## Outcome implemented in this slice

The target repository is `/Users/avery/build/low-level-tools/ai-software-architect`.
The new public binary name is `ai-architect`, and the new project artifact root is
`.ai-architect/`.

Implemented:

- A Go 1.26+ module under `cmd/`, `internal/architect/`, and `internal/mcpserver/`.
- `ai-architect questions --format json` for host-agent setup discovery.
- `ai-architect setup --answers answers.json` with `yes`, `no`, and
  `make-default` answers for binary choices.
- Separate `.adr-scaffold.yaml` and `.githabits.yaml` files. Scaffold toggles do
  not contain Git mutation policy, and Git defaults do not contain scaffold policy.
- Language detection and tailored Go, TypeScript, Python, Rust, Java, C#, and
  generic project skills. Existing files are preserved unless `--force` is explicit.
- Optional `AGENTS.md`, `CLAUDE.md`, scoped skills, project context, architecture
  contract, implementation plan, Pages workflow, static docs starter, and memory
  placeholder generation.
- `ai-architect decision new/index/list/show/check` using Markdown plus structured
  YAML frontmatter and a generated JSON index.
- ADR schema 1.1.0 fields for the five mandatory motivation answers, a comparable
  decision matrix, and a correct/incorrect action contract with good/bad outcomes.
- A compatibility extension to the existing Pydantic schema. Legacy 1.0.0
  records remain valid; 1.1.0 records require the new fields.
- Bounded static dependency evidence for Go, TypeScript/JavaScript, Python, and
  Rust source supplied to the core; no import, execution, build, or test of the
  analyzed project occurs.
- Secret-like content scanning that returns categories and line numbers, never
  suspected values.
- Explicit Git and `gh repo create` planning plus `--approve` execution. Stored
  preferences are never treated as permission.
- Read-only `git-bbq migrate` inventory plus an additive `--approve` migration
  path that archives incompatible legacy Git habits, converts validated ADRs,
  preserves `.ai-architect/`, generates the contract, implementation-plan, and
  ADR-index projections without overwriting conflicts, and validates the resulting
  project.
- Hash-based ownership ledgers and an approval-gated `git-bbq uninstall` path
  remove only unchanged generated files while preserving modified, symlinked,
  non-regular, and unowned parent paths.
- An official Go MCP STDIO adapter with five read-only tools: contract validation,
  artifact scanning, complete bundle validation, decision listing, and bounded
  inline dependency analysis.
- Atomic four-artifact bundle validation for pre-persistence candidates and the
  canonical project directory. It requires accepted ADRs, exact contract
  `decision_ids`, bounded context/handoff narratives, and secret scanning without
  returning suspected values. The CLI command is `validate-bundle`; the same
  check is available as a read-only MCP tool.
- Make targets: `go-build`, `go-test`, `go-vet`, `go-check`, `questions`, and `mcp`.
- Regression tests for the Go core and MCP inventory, plus Python compatibility
  tests for the enriched ADR contract.
- Exact-parity coverage for the canonical nested architecture contract, legacy
  1.0.0 ADR compatibility, and 1.1.0 ADR requirements, including bounds,
  uniqueness, optional/null fields, cross-reference validation, and independent
  secret-finding categories.

The original dirty worktrees were preserved. No commit, tag, push, remote creation,
release, plugin installation, or Pages deployment was performed.

## Python-to-Go replacement map

The first slice replaces only the deterministic scaffold/domain boundary. The
remaining Python host adapter is intentionally still present until parity gates
pass.

| Existing Python responsibility | Go 1.26+ replacement | Status |
| --- | --- | --- |
| Pydantic artifact/config models | Typed Go structs, `encoding/json`, strict `yaml.v3` decoding, and explicit validators | Implemented for the new scaffold/ADR/contract core; legacy schema remains for compatibility |
| PyYAML/frontmatter parsing | `gopkg.in/yaml.v3` plus bounded frontmatter extraction, alias rejection, duplicate-key rejection, and known-field decoding | Implemented |
| Python MCP package | Official `github.com/modelcontextprotocol/go-sdk` over STDIO | Implemented for five read-only tools, including atomic bundle validation |
| `pathlib`, `os`, and repository walking | Standard-library `os`, `path/filepath`, and `io/fs` with symlink and size limits | Implemented for static evidence and scaffold writes |
| subprocess/PowerShell Git orchestration | `os/exec` with `exec.CommandContext`, argument arrays, timeouts, and explicit approval | Implemented as a plan/approval boundary; release packaging is not migrated |
| pytest/race-adjacent checks | `testing`, `go test`, `go test -race`, and `go vet` | Implemented for the new Go packages |
| `uv`/PyInstaller/Codex packaging | `go.mod`, Makefile targets, and a CI Go job | Build wiring is staged; Codex bundle and clean-machine install migration remains |

No third-party Go dependency was added for vector search. SQLite FTS5 and
sqlite-vec remain evaluation candidates after the retrieval gate.

## Deliberate architectural decisions

### One product, one agent-facing binary

Use `ai-architect`, not `adr`, to avoid ambiguity with the former standalone ADR
project and to make the tool’s scope clear to an agent. The `decision` subcommands
are diagnostics and recording primitives behind an agent-led question and approval
workflow. The CLI does not replace the host’s user-approval surface.

### ADR format: structured Nygard/MADR hybrid

The implementation keeps Markdown because it is diffable and portable, YAML
frontmatter because it is directly parseable, and JSON because a generated index is
cheap for agents to retrieve. The selected ADR body is intentionally more
structured than a minimal Nygard record:

1. mandatory motivation in frontmatter;
2. context and decision drivers;
3. a stable-option decision matrix with ordinal fit, benefits, drawbacks, risks,
   evidence, and outcome;
4. the decision and explicit accepted/rejected rationale;
5. good and bad consequences;
6. an agent action contract with correct and incorrect examples; and
7. observable confirmation criteria.

This combines Nygard’s concise decision history with MADR’s metadata and option
structure. Y-Statements remain useful for very small decisions, but are too terse
as the sole record for the requested agent action and consequence guidance. The
ADR template catalogue compares these families in
[ADR Templates](https://adr.github.io/adr-templates/); a recent controlled study
found Nygard and MADR strongest in its expert screen and Nygard strongest in its
controlled task, which supports using the concise core with explicit structured
extensions rather than copying a large human-oriented template.

### Mandatory motivation and auto-reason warning

Every new project and every new 1.1.0 ADR asks:

- Why are you building this?
- What problem does it solve?
- Who is it for?
- Do suitable alternatives already exist?
- Why will this solution solve problems they cannot?

If `auto_reason` is `yes`, missing answers receive provisional agent-generated
placeholders and a visible warning. An accepted ADR cannot retain those
placeholders. This preserves momentum without laundering an unarticulated project
motivation into certainty.

### Keep AGENTS and ADRs separate

`AGENTS.md` and scoped skills are operating procedure: how an agent should work in
the repository. ADRs are historical decision evidence: why a material choice was
made, what was rejected, and what consequences are accepted. They are linked in
the read order but never merged. This avoids making every future agent consume all
decision history as instructions and prevents historical trade-offs from becoming
accidental standing policy.

### File-format decision

Use the following narrow division of labor:

| Data | Format | Reason |
| --- | --- | --- |
| Agent operating rules | Markdown | Native to Agent Skills and easy to review. |
| ADR narrative and action examples | Markdown | Keeps rationale and examples diffable. |
| ADR/contract structured authority | YAML frontmatter/YAML | Human-editable and schema-validatable. |
| Generated decision retrieval index | JSON | Stable, low-friction machine consumption. |
| Behavioral acceptance | Existing Gherkin/fixtures | Already part of the repository contract. |

TOON is not the default. The current TOON specification is a working draft and
the available benchmark evidence reports token savings with accuracy and
multi-turn parsing trade-offs. A format benchmark also found model/format
interaction matters more than a universal winner. If a future retrieval fixture
shows a material win, TOON can be used as an ephemeral transport, never as the
authoritative source.

## Configuration contract

`.adr-scaffold.yaml` owns generated content:

```yaml
version: 1
language: go
features:
  agents: true
  skills: true
  claude: false
  adr: true
  context: true
  contract: true
  implementation_plan: true
  pages: false
  memory: false
paths:
  architect: .ai-architect
  decisions: .ai-architect/decisions
  docs_site: docs-site
```

`.githabits.yaml` owns lifecycle preferences and remote metadata:

```yaml
version: 1
branch: main
versioning: semver
initial_tag: v0.1.0
commit_style: conventional-commits
init: false
commit: false
tag: false
push: false
create_remote: false
remote:
  visibility: private
```

`make-default` is persisted in the appropriate file and reused by later setup
runs. It is not an authorization token. Git, `gh`, tag, push, and deployment
actions still require a visible plan and explicit approval.

## Agent setup questions and implied defaults

Ask the first six for every project. Ask the remaining questions once per project
unless the user changes the scaffold or Git policy:

| Question | Response | Default |
| --- | --- | --- |
| Primary language | text or `auto` | `auto` detection |
| Why building? | required text | none |
| Problem solved? | required text | none |
| Intended audience? | required text | none |
| Existing alternatives? | required text | none |
| Why this solution? | required text | none |
| Permit provisional auto-reason? | yes/no/make-default | no |
| Generate `AGENTS.md`? | yes/no/make-default | yes |
| Generate scoped skills? | yes/no/make-default | yes |
| Enable ADR workflow? | yes/no/make-default | yes |
| Generate project context? | yes/no/make-default | yes |
| Generate contract starter? | yes/no/make-default | yes |
| Generate implementation plan? | yes/no/make-default | yes |
| Generate `CLAUDE.md` pointer? | yes/no/make-default | no |
| Generate Pages workflow/docs? | yes/no/make-default | no |
| Create memory placeholder? | yes/no/make-default | no |
| Initialize Git if absent? | yes/no/make-default | no |
| Create initial commit? | yes/no/make-default | no |
| Create initial tag? | yes/no/make-default | no |
| Push to origin? | yes/no/make-default | no |
| Create GitHub origin with `gh`? | yes/no/make-default | no |

The host agent should ask these questions, build an answer object, and invoke the
CLI. The CLI remains deterministic and does not attempt to infer motivation from
the repository. If the user cannot articulate motivation and declines auto-reason,
stop setup with a clear explanation; this is a project-risk finding, not a missing
technical detail to paper over.

## MCP and codebase-memory decision

The new Go MCP adapter is deliberately read-only. Its inputs are bounded inline
source or a fixed `.ai-architect/decisions` read relative to the process directory;
it does not accept arbitrary absolute roots, execute code, use a network, or write
files. The host-native approval path and `ai-architect` CLI own writes.

Do not make a third-party `codebase-mcp` a required component yet. The name refers
to multiple projects with different storage and capability models, including
implementations that expose broad indexing or write-like operations. Making one a
default would add an unreviewed prompt-injection and permission boundary and would
duplicate the host’s repository tools. If a codebase graph later proves necessary,
add it behind a bounded relative-root contract, allowlisted operations, preview,
idempotency key, explicit approval, secret/path checks, and a clean-machine
uninstall test.

The sequential-thinking MCP server is a useful optional reasoning adjunct, not a
required project dependency. It can record revisions and branches, but it does not
discover or choose tools. Tool selection, safety, and durable decisions remain in
the host workflow and Go contracts.

## Memory and embeddings gatecheck

Do not store canonical docs only in sqlite-vec. Keep raw Markdown/YAML/JSON as the
portable, reviewable source and derive indexes from it. Start with generated JSON
plus SQLite FTS5 if keyword retrieval is insufficient. Add sqlite-vec only if a
representative evaluation shows a material retrieval improvement that justifies
the dependency, rebuild behavior, backup story, and cross-platform packaging cost.

The gate must measure at least recall@k or MRR/nDCG on real project questions,
answer-support coverage, p50/p95 latency, token/context reduction, index rebuild
correctness, stale-document behavior, and failure recovery. A small project with
dozens of ADRs is normally not enough evidence for a vector database.

The deterministic M-16 harness is maintained at
`tools/retrieval_benchmark/benchmark.py`, with its versioned corpus at
`tests/fixtures/retrieval/m16.json` and operator documentation in
`docs/M16_RETRIEVAL_BENCHMARK.md`. It keeps a JSON lexical scan as a transparent
reference and evaluates the SQLite FTS5 projection against the same labels. The
generated report is written under `.tmp/retrieval/`; it is disposable evidence,
not a hand-maintained source file. The gate currently selects FTS5 and records
sqlite-vec as unselected until a like-for-like representative comparison shows
a material improvement.

The current 25-iteration generated report records FTS5 recall@3 `1.0`, MRR@5
`1.0`, nDCG@5 `0.989965`, answer-support coverage@5 `1.0`, p50/p95 latency
`0.053875`/`0.069883` ms across 200 query samples, and mean context reduction
`0.612606`. Both canonical-order and reversed-order rebuilds match all 13
documents; the stale replacement removes the old body and indexes the new one;
corruption is detected and recovery rebuilds all 13 documents. These latency
values are descriptive for this host and fixture, not production capacity
claims. The fixture SHA-256 is recorded in the generated report so the evidence
can be reproduced without editing it.

OpenAI’s official embeddings documentation lists `text-embedding-3-large` with a
default 3072-dimensional output and a `dimensions` shortening parameter. That
describes API capability, not inclusion in a ChatGPT/Codex subscription. OpenAI
documents ChatGPT and API billing as separate; use a separately authorized API key
and billing account, or a local embedding provider, and provide a raw-document
fallback when unavailable. Never make project setup depend on an unverified API
entitlement. See the [embeddings guide](https://developers.openai.com/api/docs/guides/embeddings),
the [3-large model page](https://developers.openai.com/api/docs/models/text-embedding-3-large),
and [API billing guidance](https://help.openai.com/en/articles/9039756-managing-your-work-in-the-api-platform-with-projects).

## Automation and Pages

The generated Pages workflow uses GitHub’s Actions artifact flow with
`configure-pages`, `upload-pages-artifact`, and `deploy-pages`; it does not create
or mutate a `gh-pages` branch. The workflow has explicit `contents: read`,
`pages: write`, and `id-token: write` permissions. Generation is opt-in and actual
publication remains subject to repository CI and user-approved push policy. See
[GitHub’s custom Pages workflow documentation](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages).

Remote creation uses the documented `gh repo create --source . --remote origin`
shape only after `ai-architect git bootstrap --approve`. The implementation plans
commands first and uses argument arrays rather than a shell. See the
[GitHub CLI repository-create manual](https://cli.github.com/manual/gh_repo_create).

## Good and bad action sequences

Good sequence:

1. The agent asks the setup questions and records the user’s answers.
2. It runs `ai-architect setup --answers answers.json --format json`.
3. It reads project context, contract, the decision index, and only relevant ADRs.
4. It proposes a decision with a matrix and waits for approval.
5. It runs `ai-architect decision check`, then shows the exact Git/Pages action plan.
6. It performs an external action only after explicit approval and reports evidence.

Result: the next agent can retrieve motivation, constraints, decision history, and
the permitted workflow without guessing; external changes are reviewable and
bounded.

Bad sequence:

1. The agent invents a project motivation from filenames.
2. It merges the motivation into `AGENTS.md` and treats it as permanent policy.
3. It writes an accepted ADR with one option and no trade-off.
4. It enables a broad write-capable MCP or auto-creates a remote from a stored
   `push: true` preference.
5. It publishes a mutable `gh-pages` branch before validating generated docs.

Result: historical assumptions become instructions, alternatives disappear,
permissions are confused with preferences, and an agent or malicious repository
content can cause an irreversible external action.

The correct/incorrect action examples in each ADR serve the same purpose locally:
they make the expected action and failure outcome explicit. They are examples,
not hidden policies; the actual policy remains in the scoped skill or `AGENTS.md`.

## Validation evidence from the current deterministic M-15 slice

The current worktree has focused regression evidence for additive migration,
generated projection artifacts, merged ownership hashes, read-only assessment,
symlink/reparse-point parent refusal, traversal rejection, uninstall rechecks,
preservation of unowned directories, and fail-closed responses for all five
Codex lifecycle events. The slice does not claim the separate exploratory
evaluation gate, clean-machine publication, merge, push, or release operations.

Passed:

```text
uv lock --check
uv run ruff check .
uv run mypy
uv run pytest -q                 # 182 passed, 1 skipped (Windows-only fixture)
go test ./...
go vet ./...
go build ./cmd/ai-architect ./cmd/ai-architect-mcp ./cmd/git-bbq
go test -race ./...
```

The focused M-15 regression command `go test ./internal/gitbbq ./cmd/git-bbq`
passes migration artifact/ownership checks, symlinked-parent and traversal
refusal, uninstall rechecks, unowned-directory preservation, and fail-closed
responses for all five hook events. The race-enabled Go suite also passes. The
full deterministic chain above was run with isolated temporary uv and Go caches
because the host default caches are permission-protected.

The real CLI was smoke-tested in an isolated temporary project: setup generated
the selected files, `decision new` created `ADR-001`, `decision check` passed, and
the JSON index was generated. The Go template, symlink refusal, Git-ref
validation, and atomic bundle rules also have regression coverage. The MCP
server was tested over in-memory transports for tool inventory, contract
validation, and the invalid-bundle diagnostic path. Bundle tests cover accepted
decisions, exact contract references, bounded narratives, and secret-value
redaction. A fresh CLI smoke first rejected an accepted ADR whose ID was absent
from the contract, then passed after the exact `decision_ids` link was added.

The current deterministic run records the exact command and result in the
runner handoff. No generated artifact, release package, evaluation ledger, or
packaged output was hand-edited during this slice.

## References and implementation files

- Go entrypoint: `cmd/ai-architect/main.go`
- Go MCP entrypoint: `cmd/ai-architect-mcp/main.go`
- Go core: `internal/architect/`
- MCP adapter: `internal/mcpserver/`
- Compatibility schema: `shared/schemas/src/ai_architect_schemas/models.py`
- Canonical ADR skill/template: `shared/skills/create-architecture-decisions/`
- Migration notes: `docs/GO_MIGRATION.md`
- Build shortcuts: `Makefile`

## Migration workstream context

The slice ledger above decomposes the former seven coarse workstreams into
concrete, reviewable increments. Do not restore the retired checklist; update
the ledger whenever a slice reaches its exit gate, and change the remaining
count when the next slice is selected.

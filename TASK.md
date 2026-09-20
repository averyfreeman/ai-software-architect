# Automated Git BBQ completion pipeline

## Goal

Complete the remaining automation-safe repository work from the current `main`
baseline in isolated, verified slices. Treat the three remaining roadmap gates in
`MERGE_ACTION_PLAN.md` as the scope: M-15 deterministic migration and lifecycle
parity, M-16 retrieval benchmarking, and M-17 Python runtime retirement only after
all preceding gates pass. These execution slices refine those fixed roadmap gates;
they do not add new roadmap work.

The runner may implement source and documentation changes, run the repository's
deterministic checks, and create its own verified checkpoint commits and annotated
tags. It must preserve the root worktree's unrelated untracked `.agents/` embedded
checkout and must never stage that path, overwrite it, or treat it as project
source. It must not choose a release version from stale TODO text: current version
metadata and release evidence are authoritative.

Manual Codex Desktop, clean-machine, authenticated GitHub publication, merge, and
push operations are outside the runner's authority. If any such gate is required,
record the exact blocker and handoff rather than claiming completion or weakening a
validation rule.

## Acceptance criteria

- Every unchecked item in `PLAN.md` is completed only after its implementation and
  the verification command below pass in the runner worktree.
- Mypy is green on the supported host while Windows-specific ctypes, reparse-point,
  and lifecycle behavior remains covered without broad error suppression.
- M-15 deterministic migration, artifact, hook, contract, ownership, and lifecycle
  behavior has parity-focused tests and documented evidence; static repository
  analysis remains read-only and bounded.
- M-16 retrieval evidence measures ranking quality, latency, context reduction,
  rebuild behavior, stale-document behavior, and failure recovery before any
  optional vector dependency is considered.
- M-17 removes or retires Python runtime responsibilities only when the Go path,
  packaging, clean-machine, lifecycle, and behavioral gates are proven; otherwise
  it leaves the compatibility path intact and records the unmet prerequisite.
- Generated artifacts, release packages, and evaluation outputs are not hand-edited
  or committed unless their maintained generator and provenance checks require it.
- No application-source changes are made by exploratory or release validation
  workflows, and no secret, prompt, source, or personal data is added to evidence.
- The runner's checkpoint commits and annotated tags are scoped to reviewed files;
  `.agents/`, external caches, and unrelated user work remain untouched.
- The final run status and logs identify any remaining manual release gates and give
  the exact reviewed commit/tag handoff needed before a user-owned push or publish.

## Verification

```sh
uv lock --check && uv run ruff check . && uv run mypy && uv run pytest -q && go test ./... && go vet ./... && go build ./cmd/ai-architect ./cmd/ai-architect-mcp ./cmd/git-bbq
```

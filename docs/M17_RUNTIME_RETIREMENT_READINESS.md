# M-17 Go runtime retirement readiness

**Status: NOT READY — no Python runtime cutover is authorized.**

This assessment records the M-17 boundary for the current candidate. T004 now
provides a repository-contained Go Codex host runtime, opt-in candidate
packaging/provenance, and sanitized evidence preparation. It does not delete or
disable the Python schemas, Codex hooks, packaging runtime, legacy Python MCP
package, or compatibility adapters. A Go build and deterministic test pass are
necessary but are not sufficient to retire the host runtime.

## Gate matrix

| Prerequisite | State | Evidence or blocker |
| --- | --- | --- |
| Go deterministic core | Pass | The current Go test, vet, build, and race suites pass for the migrated packages. |
| M-15 migration and behavioral gate | Blocked | `MERGE_ACTION_PLAN.md` explicitly leaves the five-fixture exploratory evaluation evidence outstanding. |
| M-16 retrieval gate | Pass for deterministic baseline | The versioned FTS5 benchmark records ranking, latency, context, rebuild, stale-document, and recovery evidence; it does not authorize a runtime cutover. |
| Go Codex host runtime | Repository candidate prepared; external acceptance blocked | `cmd/ai-architect-runtime/` and `internal/codexruntime/` implement the five-hook short-lived host path with bounded snapshots, state, continuation, policy, and artifact checks; Codex Desktop lifecycle and behavioral evidence remain external. |
| Packaging and runtime smoke | Repository candidate path prepared; exact-candidate gate blocked | `adapters/codex/build_plugin.py`, `validate_plugin.py`, and `scripts/prepare_m17_candidate_evidence.py` build, hash, validate, and record an opt-in Go candidate; clean-machine and candidate-specific smoke evidence remain required. |
| Codex Desktop lifecycle | Blocked | Release Gate E requires manual install, five-hook review, workflow execution, first-attempt uninstall, and reinstall for the exact candidate. |
| Clean-machine behavior | Blocked | Release Gate F requires clean Windows x86-64 and macOS Apple Silicon environments without Python, uv, or development caches. |
| Release evidence | Blocked | `docs/release-evidence-template.md` remains incomplete for the exact reviewed candidate and cannot be replaced by this assessment. |

The first three rows do not make M-17 ready because the remaining rows exercise
the shipped host behavior and its installation boundary. The compatibility path
therefore remains authoritative until every blocked row has candidate-specific
evidence.

## Repository-contained preparation

The automation-safe portion of T004 is intentionally opt-in:

- `build_plugin.py --build-go-runtime` cross-builds the short-lived Go host
  executable for the selected package target and records its implementation,
  source, target path, and SHA-256 in generated provenance.
- `build-codex-plugin.ps1 -GoRuntime` and
  `run-release-candidate-gates.ps1 -RuntimeImplementation go` exercise that
  candidate path while leaving the Python default unchanged.
- `prepare_m17_candidate_evidence.py` validates the exact package and writes a
  sanitized, candidate-specific evidence record under `.tmp/`; it explicitly
  records the remaining external gates and `authorized: false` cutover state.

These repository checks prepare evidence; they do not constitute Codex Desktop,
clean-machine, exploratory, behavioral, release-publication, merge, or push
approval.

## Required handoff before cutover

The following work is outside this automation-safe slice and must be completed
against one reviewed candidate:

1. Run the maintained five-fixture exploratory campaign and review every
   expected and forbidden behavior, preserving its generated evidence without
   changing application source.
2. Build and inspect the exact Go-backed package with
   `run-release-candidate-gates.ps1 -RuntimeImplementation go`, then generate
   the candidate evidence record for that exact package.
3. Complete Release Gate E in Codex Desktop, including first-attempt uninstall
   and reinstall with all five hooks reviewed.
4. Complete Release Gate F on clean Windows x86-64 and macOS Apple Silicon
   systems that have no Python, uv, or development caches.
5. Complete the per-version release evidence record, rerun the deterministic
   verification command, and review the resulting commit and package hashes.

Only after those records are complete may a separately approved M-17 change
retire Python runtime responsibilities. That change must preserve canonical
artifacts and compatibility migration behavior, and must be reviewed as a
candidate-specific cutover rather than inferred from this readiness document.

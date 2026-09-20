# Automated execution slices

- [x] T001: Repair cross-platform static typing and establish a green deterministic baseline without weakening Windows safety behavior.
- [x] T002: Complete the deterministic M-15 migration, artifact, ownership, hook, and lifecycle parity gates with focused regression evidence and authoritative roadmap updates.
- [x] T003: Build the M-16 retrieval benchmark and SQLite FTS5 gate, measuring ranking quality, latency, context reduction, rebuilds, stale documents, and recovery before considering sqlite-vec.
- [ ] T004: Complete M-17 Go-runtime retirement readiness and repository-contained prerequisite work, including the Go Codex host runtime, candidate packaging/provenance, and evidence preparation; do not remove Python compatibility paths or claim cutover before external gates pass.
- [ ] T005: Execute the candidate-specific M-17 Go-runtime cutover only after M-15, M-16, package, clean-machine, lifecycle, behavioral, and release-evidence gates are demonstrably satisfied; preserve compatibility migration behavior and stop on any unmet gate.
- [ ] T006: Run final deterministic conformance and packaging validation, reconcile release-version and evidence drift, and produce a precise user-owned tag, commit, push, and publication handoff without performing publication.

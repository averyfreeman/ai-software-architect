<!--
SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
SPDX-License-Identifier: MIT
-->

# Git BBQ

This context names the Git BBQ concepts shared by Matt-native architecture
records, project scaffolding, Git habits, and Codex lifecycle hooks.

## Language

**Codex control plane**:
The deterministic host-side coordination layer that routes activated architecture
turns, protects tool use, and verifies durable workflow state.

_Avoid_: architecture brain, model controller

**Hook event**:
A host notification at one lifecycle point where the control plane may inject
context, deny an operation, validate a result, or remain silent.

_Avoid_: callback, model event

**Continuation**:
A single-use follow-up turn that resumes the immediately preceding clarification
or decision request without requiring another architecture invocation.

_Avoid_: session, retry, new workflow

**Checkpoint**:
A minimal restorable record of the current architecture workflow phase and any
artifact expectations that must survive compaction.

_Avoid_: transcript, prompt snapshot, cache

**Lifecycle**:
The ordered progression of activation, continuation, checkpoint, validation, and
terminal cleanup effects across the five Codex hook events.

_Avoid_: orchestration script, event loop

**Architecture artifact bundle**:
The coherent set of Matt-native context and ADR files plus generated projections
prepared and validated together for a recorded decision.

_Avoid_: handoff files, duplicate ADR set, partial write

**Matt-native ADR**:
An ADR stored as a sequential `docs/adr/NNNN-slug.md` Markdown file using the
Matt Pocock domain-modeling convention, with optional status, options, and
consequences sections.

_Avoid_: ADR-NNN, structured ADR bundle, decision YAML

**Githabits**:
The project-specific policy governing Git initialization, staging, commits, tags,
remote setup, and pushes, with independent action controls.

_Avoid_: Git preference, automatic Git permission

**Projection**:
A generated contract, implementation plan, or index derived from Matt-native
documents; it is never a second source of architectural truth.

_Avoid_: duplicate artifact, alternate ADR

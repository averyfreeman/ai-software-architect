---
adr_id: ADR-0006
title: "JSON as Canonical Structured Output"
status: adopted
date: 2026-09-13
scope:
  paths:
    - "cmd/**"
    - "internal/**"
    - "docs/**"
    - "demo/**"
    - "SPEC.md"
tags:
  - format
  - cli
  - output
constraints:
  - JSON must be the canonical machine-readable output for normal commands
  - YAML remains the generated format for ADR index files
  - Human-oriented commands default to text output
  - Structured output must be deterministic and preserve ADR semantics
invariants:
  - JSON consumers do not need an alternate parser for CLI integration
  - The ADR index is generated from ADR metadata
  - review identifies decisions for review and does not claim semantic compliance
supersedes: []
superseded_by: []
related_adrs:
  - ADR-0002
---

# ADR-0006: JSON as Canonical Structured Output

## Context

adr-rg serves both humans working in a repository and automation that needs to
query ADR metadata. Every additional representation increases parser,
validation, documentation, and interoperability costs. The CLI must integrate
with common shell tools, JSON Schema-oriented consumers, AI Software Architect,
and generic language libraries without requiring a project-specific decoder.

## Decision

Use JSON as the canonical machine-readable output for normal `adr` commands.
Keep text as the default for human-oriented output and YAML for the generated
ADR index.

### JSON as Canonical Output: Adopted

**Adopted because:**
- JSON is supported by standard libraries and common automation tools
- A single canonical structured representation reduces conversion ambiguity
- JSON remains compatible with MCP-style JSON-RPC integrations and plugin APIs
- Deterministic JSON is straightforward to diff, validate, cache, and archive
- The format does not require consumers to install a project-specific parser

**Adopted despite:**
- JSON can use more tokens than compact tabular representations
- JSON is less pleasant to read than the CLI's text presentation for humans

## Alternatives Considered

### YAML for All Structured Output: Rejected

**Rejected because:**
- YAML has more parser and scalar-normalization edge cases than JSON
- It is not the existing contract for programmatic command output
- Unnecessarily broad YAML output would blur the role of the generated index

**Rejected despite:**
- Familiarity to repository maintainers
- Human readability for configuration and index inspection

## Consequences

JSON is the stable integration contract for AI Software Architect and other
tools. Consumers can use standard parsers, while humans retain concise text
output. The ADR index continues to be YAML because it is a generated repository
artifact, not a general command-response protocol. LLM-facing callers may
project JSON into another prompt representation at their own boundary, but
that projection is not an `adr-rg` wire contract.

The CLI keeps a single canonical structured representation. Consumers that need
machine-readable output should use `--format json`; human-oriented callers can
use text output as appropriate.

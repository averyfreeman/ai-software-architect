# AGENTS.md: the project-scoped "Start Here" document

# Purpose
This document defines the primary operating parameters, baseline architectural layout, and style guidelines for autonomous agents interacting with this repository. Detailed procedural execution data is delegated to specific skill definitions located in `.agents/skills/`.

# Project Scaffolding & Layout
The evolving filesystem layout must adhere to the following baseline structure. Agents are expected to output tree structures mapping to this standard when communicating layout changes.

.
├── .agents
│   └── skills
│       ├── repo-init/SKILL.md
│       ├── rustdoc-generation/SKILL.md
│       ├── starlight-doc-deployment/SKILL.md
│       └── version-tag-and-commit/SKILL.md
├── .github
│   └── workflows
│       └── pages.yml
├── docs
│   ├── astro.config.mjs
│   └── src
├── src
│   └── main.rs
├── AGENTS.md
├── Cargo.lock
├── Cargo.toml
├── CLAUDE.md
├── NEXT_STEPS.md
└── README.md

# Brief Coding Style Guide
* **Language:** Rust
* **Idioms:** Strict adherence to idiomatic Rust patterns.
* **Error Handling:** Prefer explicit `Result` and `Option` handling over `unwrap()` or `expect()`.
* **Memory Safety:** Maintain strict borrow checker compliance without unnecessary `clone()` operations or `unsafe` blocks unless explicitly required by FFI or hardware interfaces.
* **Formatting:** All code must conform to canonical `rustfmt` specifications.

# Documentation Requirements
* **Inline Documentation:** Automated extraction via standard Rust comment tags (`///` and `//!`) is strictly required for all public APIs, modules, and structures.
* **Reference:** See `.agents/skills/language-generation/SKILL.md`
* **Presentation Layer:** Every project requires an Astro Starlight documentation site deployed via GitHub pages. No exceptions.
* **Reference:** See `.agents/skills/starlight-doc-deployment/SKILL.md`

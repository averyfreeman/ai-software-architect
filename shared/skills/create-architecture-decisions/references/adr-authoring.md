<!--
SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
SPDX-License-Identifier: MIT
-->

# ADR Authoring

Record one material decision per ADR. State the context and forces that existed when deciding, enumerate credible options by stable identifiers, name the selected option, and capture positive and negative consequences without promotional language.

Every new 1.1.0 ADR frontmatter record must answer five motivation questions:
why the project is being built, what problem it solves, who it is for, whether
suitable alternatives already exist, and why this solution solves problems those
alternatives cannot. An agent may fill these fields only when the user opts into
auto-reason; label the source and warn that the result needs human review before
acceptance. An inability to articulate the answers is a project-risk signal, not
a reason to invent certainty.

Use `proposed` before approval and `accepted` only after explicit approval. Never edit historical meaning invisibly: supersede an accepted decision with a new ADR and link both records. Make validation criteria observable. Keep confidential values and source excerpts out of the record.

The file begins with safe YAML frontmatter conforming to `ArchitectureDecisionArtifact`. The filename starts with the matching `ADR-NNN`; any slug uses lowercase ASCII letters, digits, and single hyphens. The Markdown body is a deterministic rendering of the same fields. The 1.1.0 form adds a decision matrix and an agent action contract containing good/bad outcomes plus correct/incorrect examples. Keep operating instructions in `AGENTS.md` or a scoped skill; do not merge them into ADR history.

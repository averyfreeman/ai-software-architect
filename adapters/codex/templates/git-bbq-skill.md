---
name: git-bbq
description: Scaffold and maintain an agent-ready repository with Matt-native architecture records and explicit Git habits.
---

# Git BBQ

Use the `git-bbq` executable for repository scaffolding, assessment, projection,
validation, and lifecycle hook handling. Treat Matt Pocock skills as the
semantic architecture authority. Read `CONTEXT.md` and the relevant files under
`docs/adr/` before proposing a material architectural decision.

Project-owned Git behavior is defined by `.githabits.yaml`. Check each action's
policy before initializing Git, creating branches, staging, committing, tagging,
configuring a remote, or pushing. A profile is policy, not a bypass for
validation or user approval.

ADRs use the Matt-native sequential format in `docs/adr/NNNN-slug.md`. Do not
create a second ADR directory or put native structured metadata into an ADR.
Generated contracts, plans, and indexes are projections and may be recreated
from the Matt-native documents.

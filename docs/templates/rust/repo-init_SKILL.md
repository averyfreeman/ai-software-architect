# Skill: Repository Initialization
**Domain:** Git / GitHub CLI
**Scope:** Repository bootstrapping and canonical file generation.

## Execution Parameters
* Generate baseline `.gitignore` targeting environment files (`.env*`) and language-specific build directories (`target/`).
* Generate stub `README.md` containing primary project definitions.
* Treat local initialization (`git init`) as an explicit, user-approved action.
* Treat remote creation via GitHub CLI (`gh repo create <name> --public --source=. --remote=origin`) as an explicit, user-approved action.
* Ensure primary branch name is `main`.

# AGENTS.md: the project-scoped "Start Here" document

This repository uses {{.DisplayName}}. The language profile below defines the
expected build, test, formatting, and documentation commands.

## Project layout

Keep implementation files in the language-appropriate source directories and
keep durable architectural decisions in `docs/adr/`. Keep generated artifacts
out of version control according to `.gitignore`.

## Language standards

- Build: `{{.BuildCommand}}`
- Test: `{{.TestCommand}}`
- Format: `{{.FormatCommand}}`
- Documentation: {{.Documentation}}

Before changing governed code, run `adr review --base main` and read the
applicable decisions. Before finishing, run `adr check --strict` and `adr
index --check`.

## Git defaults

The generated `.adr-scaffold.yaml` records repository Git habits. Scaffolding
does not initialize Git, create remotes, commit, tag, or push automatically.
Ask before performing any external or irreversible Git operation.

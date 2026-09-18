# adr-rg Reference

## Command surface

| Command | Purpose |
| --- | --- |
| `adr init [OPTIONS]` | Initialize the ADR directory and index |
| `adr new [OPTIONS] TITLE` | Create a numbered ADR |
| `adr list [OPTIONS]` | List ADRs with filters |
| `adr show [OPTIONS] IDENTIFIER` | Show one ADR |
| `adr check [OPTIONS]` | Validate ADR files (`check adr` remains compatible) |
| `adr index [OPTIONS]` | Generate or check the index |
| `adr review [--base REF] [OPTIONS]` | Find decisions relevant to changed paths (`detect-bs` is an alias) |
| `adr scaffold [LANG] [OPTIONS]` | Generate a language-specific project scaffold |
| `adr version` | Print version and build metadata |

Common `--dir`, `--format`, and `--json` options may appear before or after the
command. Put command-specific options before the positional title or
identifier.

## Options

| Option | Commands | Meaning |
| --- | --- | --- |
| `--dir PATH` | ADR commands | ADR directory; default `docs/adr` |
| `--dir PATH` | scaffold | Target project directory; default `.` |
| `--format text\|json` | normal commands | Human or machine-readable output |
| `--format text\|json\|yaml` | index | Index output format |
| `--json` | all commands except version | Shorthand for `--format json` |
| `--status STATUS` | new, list | Initial or filtered lifecycle status |
| `--tag TAG` | list | Filter by tag |
| `--path PATH` | list | Repeatable filter by scope-path match; comma-separated values remain supported |
| `--paths GLOBS`, `--path GLOB` | new | Repeatable scope globs; comma-separated values remain supported |
| `--tags TAGS`, `--tag TAG` | new | Repeatable tags; comma-separated values remain supported |
| `--lang LANG` | new | Optional language profile or alias for the ADR context |
| `--no-index` | new | Skip index refresh |
| `--strict` | check | Treat warnings as failures |
| `--check` | index | Check freshness without writing |
| `--base REF` | review | Git diff base reference; auto-detected when unambiguous |
| `--lang LANG` | scaffold | Language template name or alias; may be positional |
| `--force` | scaffold | Overwrite existing scaffold files without prompting |

## Output formats

Text is the default for interactive use. JSON is the canonical machine-readable
format. YAML is reserved for the generated ADR index.

## JSON result shapes

- `init`: `adr_dir`, `template`, and `index`;
- `new`: ADR ID, title, filename, path, number, and resolved canonical language;
- `list`: `count` and `adrs`;
- `show`: ADR metadata, decision, governance fields, and filename;
- `check`: `valid`, `count`, per-file results, errors, and warnings;
- `index`: generation/check status, index path, and ADR count; and
- `review` (also `detect-bs`): `changed_files`, `applicable_adrs`, and `summary`; and
- `scaffold`: selected language, target directory, generated files, and config path.

Short aliases are available for common options: `-d` for `--dir`, `-o` for
`--format`, `-b` for `--base`, `-l` for `--lang`, and `-F` for `--force`.

## ADR frontmatter

~~~yaml
adr_id: ADR-NNNN
title: "Decision title"
status: proposed | adopted | rejected | deprecated | superseded
date: YYYY-MM-DD
scope:
  paths: ["src/**"]
tags: [architecture]
constraints: []
invariants: []
supersedes: []
superseded_by: []
related_adrs: []
~~~

The body requires `Context`, `Decision`, `Alternatives Considered`, and
`Consequences` headings. Strict validation also enforces the configured rationale
pattern. Repeated tag or path filters are ORed within their category; status
and other filter categories are combined with them.

## Index

`docs/adr/index.yaml` is generated from frontmatter. Run `adr index` after
metadata changes and `adr index --check` in CI. Do not edit it directly.

## Bullshit Detector

`adr review` obtains changed paths from `git diff --name-only REF` and matches
each proposed or adopted ADR’s `scope.paths` globs. If `--base` is omitted, an
unambiguous remote or local main ref is selected automatically. It reports matching
ADR IDs, titles, paths, files, constraints, and invariants. Rejected, deprecated,
and superseded records remain queryable history but do not create active signals.

`adr detect-bs` is retained as a compatibility alias. The command identifies
decisions requiring review; it does not prove semantic compliance.

## Scaffold

`adr scaffold LANG` (or `adr scaffold --lang LANG`) writes the default agent files, language guidance,
`.gitignore`, README, skills, and `.adr-scaffold.yaml` into the target directory.
Supported profiles include TypeScript, JavaScript, Go, Rust, C, C++, HTML,
Haskell, Common Lisp/CLISP, Lua, PHP, Erlang, Elixir, and generic projects.

Scaffolding performs no Git or network actions. Existing files are handled by an
interactive overwrite, skip, rename, or alternate-path prompt. Use `--force` to
overwrite all collisions without prompting.

## Language-aware ADRs

`adr new` uses the language-neutral `generic` profile when `--lang` is omitted.
Pass a canonical profile or alias to add an opt-in `Language Context` section to
the ADR body, for example `adr new --lang go "Use Go for the service"` or
`adr new --lang rs "Use Rust for the service"`. Existing local templates remain
compatible; templates without the opt-in block are rendered as before.

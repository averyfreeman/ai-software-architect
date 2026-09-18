# Getting started

adr-rg is a small Go CLI for creating, validating, indexing, and querying
Architecture Decision Records.

## Install

Install from source with Go 1.26 or later:

~~~bash
go install github.com/averyfreeman/adr-repo-governance/cmd/adr@v0.3.0
adr version
~~~

From a checkout, use the Makefile for the XDG user-space layout:

~~~bash
make install
~~~

By default, the binary is copied to `~/.local/share/adr-rg/adr` and linked at
`~/.local/bin/adr`. Set `XDG_DATA_HOME`, `XDG_BIN_HOME`, or `prefix` to change
those locations. On macOS, `make install-mac-layout` uses
`~/Library/Application Support/org.unixgreybeard.adr-rg` for the application
files while keeping the same user-bin symlink.

The release workflow still publishes checksummed archives, but this repository
does not ship tar-download installer scripts.

## Scaffold a repository

Generate the standard agent files, language guidance, and editable Git policy:

~~~bash
adr scaffold rust
adr scaffold --lang go --dir ./new-project
~~~

Scaffolding only writes local files. It does not initialize Git, create a
remote, commit, tag, or push. Existing files are handled interactively; use
`--force` to overwrite them without prompting.

## Start a repository

~~~bash
adr init
~~~

This creates `docs/adr/`, the ADR template, and the generated `index.yaml`.
Use `--dir PATH` when the repository keeps its records elsewhere.

## Create a decision

~~~bash
adr new --lang go --tag database --path "src/db/**" "Use PostgreSQL for persistence"
~~~

Use `--lang LANG` with a canonical language name or alias to add the selected
profile’s build, test, formatting, and documentation conventions to the ADR.
Omit it to create the language-neutral `generic` ADR template.

Finish the generated `Context`, `Decision`, `Alternatives Considered`, and
`Consequences` sections. Add constraints and invariants when the decision needs
to guide later implementation.

## Validate and index

~~~bash
adr check --strict
adr index
adr index --check
~~~

The strict check validates every ADR. `adr index` regenerates derived metadata;
`adr index --check` fails when the committed index is stale.

## Review a change

~~~bash
adr review
adr review --base main --json
~~~

The Bullshit Detector compares changed Git paths with the scopes on proposed and
adopted ADRs. The base is auto-detected when unambiguous, or can be supplied
explicitly. A match tells you what to read; it is not a semantic compliance
verdict.

## A few useful queries

~~~bash
adr list --status adopted
adr list --tag database
adr list --path "src/db/users.go"
adr show ADR-0001
~~~

For the complete flag and output contract, see the [reference](../reference/README.md).

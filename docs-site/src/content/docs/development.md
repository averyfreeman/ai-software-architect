---
title: Development
description: Build and validate adr-repo-governance locally.
---

The repository requires Go for local development. Run the focused checks before publishing changes:

```bash
go test ./...
go vet ./...
adr check --strict
adr index --check
```

Run `adr review --base main` before source changes when reviewing a branch. Do not edit the generated index by hand.

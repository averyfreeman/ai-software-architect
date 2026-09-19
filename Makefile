GO ?= go
BIN_DIR ?= bin

.PHONY: go-build go-test go-vet go-check questions mcp git-bbq git-bbq-plugin

go-build:
	@mkdir -p "$(BIN_DIR)"
	$(GO) build -o "$(BIN_DIR)/git-bbq" ./cmd/git-bbq
	$(GO) build -o "$(BIN_DIR)/ai-architect" ./cmd/ai-architect
	$(GO) build -o "$(BIN_DIR)/ai-architect-mcp" ./cmd/ai-architect-mcp

go-test:
	$(GO) test ./...

go-vet:
	$(GO) vet ./...

go-check: go-test go-vet

questions:
	$(GO) run ./cmd/ai-architect questions --format json

mcp:
	$(GO) run ./cmd/ai-architect-mcp

git-bbq:
	$(GO) run ./cmd/git-bbq

git-bbq-plugin:
	python3 scripts/build_git_bbq_plugin.py --force

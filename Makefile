.PHONY: fmt test race vet schema-check build build-runtime build-all \
	build-windows-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64 \
	build-windows build-macos package package-all validate-package smoke-linux \
	clean check

GO ?= go
VERSION ?= $(shell sed -n '1p' VERSION)
DIST ?= dist
LDFLAGS ?= -X main.version=$(VERSION)

fmt:
	@test -z "$$($(GO)fmt -l .)"

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

schema-check:
	$(GO) run ./cmd/generate-schemas --check

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o "$(DIST)/adr" ./cmd/adr

build-runtime:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o "$(DIST)/runtime/$(TARGET)/ai-architect-runtime/$(RUNTIME_NAME)" ./cmd/ai-architect-runtime

build-windows-amd64:
	$(MAKE) build-runtime TARGET=windows-amd64 RUNTIME_NAME=ai-architect-runtime.exe GOOS=windows GOARCH=amd64

build-darwin-arm64:
	$(MAKE) build-runtime TARGET=darwin-arm64 RUNTIME_NAME=ai-architect-runtime GOOS=darwin GOARCH=arm64

build-linux-amd64:
	$(MAKE) build-runtime TARGET=linux-amd64 RUNTIME_NAME=ai-architect-runtime GOOS=linux GOARCH=amd64

build-linux-arm64:
	$(MAKE) build-runtime TARGET=linux-arm64 RUNTIME_NAME=ai-architect-runtime GOOS=linux GOARCH=arm64

build-windows: build-windows-amd64
build-macos: build-darwin-arm64
build-all: build-windows-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64

package:
	$(GO) run ./cmd/codex-plugin package --target "$(TARGET)" --version "$(VERSION)"

package-all:
	$(GO) run ./cmd/codex-plugin package --target windows-amd64 --version "$(VERSION)"
	$(GO) run ./cmd/codex-plugin package --target darwin-arm64 --version "$(VERSION)"
	$(GO) run ./cmd/codex-plugin package --target linux-amd64 --version "$(VERSION)"
	$(GO) run ./cmd/codex-plugin package --target linux-arm64 --version "$(VERSION)"

validate-package:
	@test -n "$(TARGET)"
	@archive="$(DIST)/release/ai-software-architect-v$(VERSION)-$(TARGET).zip"; \
	test -f "$$archive"; \
	extract_dir="$$(mktemp -d)"; \
	unzip -q "$$archive" -d "$$extract_dir"; \
	$(GO) run ./cmd/codex-plugin validate --target "$(TARGET)" "$$extract_dir"

smoke-linux: build-linux-amd64
	@if [ "$$(uname -s)" = "Linux" ] && [ "$$(uname -m)" = "x86_64" ]; then \
		"$(DIST)/runtime/linux-amd64/ai-architect-runtime/ai-architect-runtime" --repository-snapshot --root . > "$(DIST)/snapshot-linux-amd64.json"; \
		test -s "$(DIST)/snapshot-linux-amd64.json"; \
	else \
		echo "Linux amd64 binary smoke test deferred to a native Linux runner"; \
	fi

clean:
	rm -rf "$(DIST)"

check: fmt test vet schema-check build

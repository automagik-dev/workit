SHELL := /bin/bash

# `make` should build the binary by default.
.DEFAULT_GOAL := build

.PHONY: build gog gogcli gog-help gogcli-help help fmt fmt-check lint lint-full test ci tools
.PHONY: worker-ci build-internal deadcode race coverage

BIN_DIR := $(CURDIR)/bin
BIN := $(BIN_DIR)/gog
CMD := ./cmd/gog

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
COMMIT := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo "")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COVERAGE_MIN ?= 70
LDFLAGS := -X github.com/namastexlabs/gog-cli/internal/cmd.version=$(VERSION) -X github.com/namastexlabs/gog-cli/internal/cmd.branch=$(BRANCH) -X github.com/namastexlabs/gog-cli/internal/cmd.commit=$(COMMIT) -X github.com/namastexlabs/gog-cli/internal/cmd.date=$(DATE)

TOOLS_DIR := $(CURDIR)/.tools
GOFUMPT := $(TOOLS_DIR)/gofumpt
GOIMPORTS := $(TOOLS_DIR)/goimports
DEADCODE := $(TOOLS_DIR)/deadcode
DEADCODE_BASELINE := .deadcode-baseline.txt
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint
LINT_NEW_FROM ?= origin/main

# Allow passing CLI args as extra "targets":
#   make gogcli -- --help
#   make gogcli -- gmail --help
ifneq ($(filter gogcli gog,$(MAKECMDGOALS)),)
RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
$(eval $(RUN_ARGS):;@:)
endif

build:
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD)

# Build with internal defaults for headless OAuth (credentials baked in)
# Usage: make build-internal GOG_CLIENT_ID=... GOG_CLIENT_SECRET=... GOG_CALLBACK_SERVER=...
build-internal:
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LDFLAGS) \
		-X 'github.com/namastexlabs/gog-cli/internal/config.DefaultClientID=$(GOG_CLIENT_ID)' \
		-X 'github.com/namastexlabs/gog-cli/internal/config.DefaultClientSecret=$(GOG_CLIENT_SECRET)' \
		-X 'github.com/namastexlabs/gog-cli/internal/config.DefaultCallbackServer=$(GOG_CALLBACK_SERVER)'" \
		-o $(BIN) $(CMD)

# Build with credentials from ~/.config/gog/credentials.env
# Usage: make build-namastex
build-namastex:
	@mkdir -p $(BIN_DIR)
	@if [ -f "$(HOME)/.config/gog/credentials.env" ]; then \
		. $(HOME)/.config/gog/credentials.env && \
		go build -ldflags "$(LDFLAGS) \
			-X 'github.com/namastexlabs/gog-cli/internal/config.DefaultClientID=$${GOG_CLIENT_ID}' \
			-X 'github.com/namastexlabs/gog-cli/internal/config.DefaultClientSecret=$${GOG_CLIENT_SECRET}' \
			-X 'github.com/namastexlabs/gog-cli/internal/config.DefaultCallbackServer=$${GOG_CALLBACK_SERVER}'" \
			-o $(BIN) $(CMD); \
	else \
		echo "❌ Missing credentials file: $(HOME)/.config/gog/credentials.env"; \
		echo "   Run: ./scripts/setup-credentials.sh"; \
		exit 1; \
	fi

gog: build
	@if [ -n "$(RUN_ARGS)" ]; then \
		$(BIN) $(RUN_ARGS); \
	elif [ -z "$(ARGS)" ]; then \
		$(BIN) --help; \
	else \
		$(BIN) $(ARGS); \
	fi

gogcli: build
	@if [ -n "$(RUN_ARGS)" ]; then \
		$(BIN) $(RUN_ARGS); \
	elif [ -z "$(ARGS)" ]; then \
		$(BIN) --help; \
	else \
		$(BIN) $(ARGS); \
	fi

gog-help: build
	@$(BIN) --help

gogcli-help: build
	@$(BIN) --help

help: gog-help

tools:
	@mkdir -p $(TOOLS_DIR)
	@GOBIN=$(TOOLS_DIR) go install mvdan.cc/gofumpt@v0.9.2
	@GOBIN=$(TOOLS_DIR) go install golang.org/x/tools/cmd/goimports@v0.41.0
	@GOBIN=$(TOOLS_DIR) go install golang.org/x/tools/cmd/deadcode@v0.41.0
	@GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0

fmt: tools
	@$(GOIMPORTS) -local github.com/namastexlabs/gog-cli -w .
	@$(GOFUMPT) -w .

fmt-check: tools
	@$(GOIMPORTS) -local github.com/namastexlabs/gog-cli -w .
	@$(GOFUMPT) -w .
	@git diff --exit-code -- '*.go' go.mod go.sum

lint: tools
	@$(GOLANGCI_LINT) run --new-from-rev=$(LINT_NEW_FROM)

lint-full: tools
	@$(GOLANGCI_LINT) run

pnpm-gate:
	@if [ -f package.json ] || [ -f package.json5 ] || [ -f package.yaml ]; then \
		pnpm lint && pnpm build && pnpm test; \
	else \
		echo "pnpm gate skipped (no package.json)"; \
	fi

test:
	@go test ./...

deadcode: tools
	@tmp_dead=$$(mktemp); \
	$(DEADCODE) ./... > "$$tmp_dead"; \
	if [ ! -f "$(DEADCODE_BASELINE)" ]; then \
		echo "missing $(DEADCODE_BASELINE); generate baseline before running deadcode gate" >&2; \
		rm -f "$$tmp_dead"; \
		exit 1; \
	fi; \
	if ! diff -u "$(DEADCODE_BASELINE)" "$$tmp_dead"; then \
		echo "deadcode gate failed: output changed from baseline" >&2; \
		rm -f "$$tmp_dead"; \
		exit 1; \
	fi; \
	rm -f "$$tmp_dead"; \
	echo "deadcode baseline check: OK"

race:
	@go test -race ./...

coverage:
	@tmp_cov=$$(mktemp); \
	go test -coverprofile="$$tmp_cov" ./... >/dev/null; \
	total=$$(go tool cover -func="$$tmp_cov" | awk '/^total:/ {gsub("%","",$$3); print $$3}'); \
	rm -f "$$tmp_cov"; \
	awk -v total="$$total" -v min="$(COVERAGE_MIN)" 'BEGIN { \
		printf("coverage total: %.1f%% (min %.1f%%)\n", total, min); \
		if (total+0 < min+0) exit 1; \
	}'

ci: pnpm-gate fmt-check lint test deadcode race coverage

worker-ci:
	@pnpm -C internal/tracking/worker lint
	@pnpm -C internal/tracking/worker build
	@pnpm -C internal/tracking/worker test

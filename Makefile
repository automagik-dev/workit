SHELL := /bin/bash

.PHONY: fmt fmt-check lint test ci tools

TOOLS_DIR := $(CURDIR)/.tools
GOFUMPT := $(TOOLS_DIR)/gofumpt
GOIMPORTS := $(TOOLS_DIR)/goimports
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint

tools:
	@mkdir -p $(TOOLS_DIR)
	@GOBIN=$(TOOLS_DIR) go install mvdan.cc/gofumpt@v0.7.0
	@GOBIN=$(TOOLS_DIR) go install golang.org/x/tools/cmd/goimports@v0.38.0
	@GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2

fmt: tools
	@$(GOIMPORTS) -w .
	@$(GOFUMPT) -w .

fmt-check: tools
	@$(GOIMPORTS) -w .
	@$(GOFUMPT) -w .
	@git diff --exit-code -- '*.go' go.mod go.sum

lint: tools
	@$(GOLANGCI_LINT) run

test:
	@go test ./...

ci: fmt-check lint test

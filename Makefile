.PHONY: all build build-all install test test-go test-python lint lint-go lint-python \
       dev-setup init run clean clean-all docs docs-serve docs-deploy \
       mcp-setup mcp-run mcp-install help

# ──────────────────────────────────────────────
# Variables
# ──────────────────────────────────────────────

BINARY   := caof
CMD      := ./cmd/caof
BIN_DIR  := bin

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

WORKSPACE ?= $(HOME)/caof-workspace
GOAL      ?= "Default goal"
AGENTS_DIR := agents
VENV       := $(AGENTS_DIR)/.venv

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

all: build

build:                          ## Compile the Go CLI binary for current platform
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD)

build-all:                      ## Cross-compile for linux and macOS (amd64 + arm64)
	GOOS=linux  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-amd64  $(CMD)
	GOOS=linux  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-arm64  $(CMD)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-darwin-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-darwin-arm64 $(CMD)

install: build                  ## Build and install binary to $GOPATH/bin
	cp $(BIN_DIR)/$(BINARY) $(shell go env GOPATH)/bin/$(BINARY)

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────

test: test-go test-python       ## Run all tests

test-go:                        ## Run Go tests
	go test ./... -v -count=1 -short

test-python:                    ## Run Python agent tests
	cd $(AGENTS_DIR) && python3 -m pytest tests/ -v

# ──────────────────────────────────────────────
# Lint
# ──────────────────────────────────────────────

lint: lint-go lint-python       ## Lint all code

lint-go:                        ## Lint Go code
	golangci-lint run ./...

lint-python:                    ## Lint Python code with ruff
	cd $(AGENTS_DIR) && ruff check .

# ──────────────────────────────────────────────
# Dev
# ──────────────────────────────────────────────

dev-setup: build                ## Set up local dev environment
	cd $(AGENTS_DIR) && pip install -e ".[dev]"
	@echo "Ready. Binary at $(BIN_DIR)/$(BINARY)"

init: build                     ## Bootstrap the full environment
	./$(BIN_DIR)/$(BINARY) init --workspace $(WORKSPACE)

run: build                      ## Submit a goal: make run GOAL="…"
	./$(BIN_DIR)/$(BINARY) run --goal $(GOAL)

# ──────────────────────────────────────────────
# Docs
# ──────────────────────────────────────────────

docs:                           ## Build MkDocs documentation to site/
	mkdocs build --strict

docs-serve:                     ## Serve docs locally with live reload
	mkdocs serve

docs-deploy:                    ## Deploy docs to GitHub Pages
	mkdocs gh-deploy --force

# ──────────────────────────────────────────────
# MCP Server
# ──────────────────────────────────────────────

MCP_DIR  := mcp
MCP_VENV := $(MCP_DIR)/.venv

mcp-setup: $(MCP_VENV)          ## Set up MCP server venv

$(MCP_VENV): $(MCP_DIR)/pyproject.toml
	python3 -m venv $(MCP_VENV)
	$(MCP_VENV)/bin/pip install --upgrade pip -q
	$(MCP_VENV)/bin/pip install -e "$(MCP_DIR)" -q
	@touch $(MCP_VENV)

mcp-run: $(MCP_VENV)            ## Run MCP server locally (stdio)
	cd $(MCP_DIR) && .venv/bin/python -m server

mcp-install:                    ## Install MCP as systemd user service
	./scripts/install-mcp-service.sh --user

# ──────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────

clean:                          ## Remove build artifacts
	rm -rf $(BIN_DIR)

clean-all: clean                ## Remove build artifacts and teardown sessions
	./$(BIN_DIR)/$(BINARY) teardown --force 2>/dev/null || true

# ──────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────

help:                           ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

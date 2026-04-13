.PHONY: all build test test-go test-python lint lint-go lint-python init run clean docs docs-serve docs-deploy

WORKSPACE ?= $(HOME)/caof-workspace
GOAL ?= "Default goal"

all: build

build:                    ## Compile the Go CLI binary
	go build -o bin/caof ./cmd/caof

test: test-go test-python ## Run all tests

test-go:                  ## Run Go tests
	go test ./... -v -count=1

test-python:              ## Run Python tests
	cd agents && python3 -m pytest tests/ -v

lint: lint-go lint-python  ## Lint all code

lint-go:                  ## Lint Go code
	golangci-lint run

lint-python:              ## Lint Python code
	cd agents && ruff check .

init: build               ## Bootstrap the full environment
	./bin/caof init --workspace $(WORKSPACE)

run: init                 ## Submit a goal and start execution
	./bin/caof run --goal $(GOAL)

clean:                    ## Tear down sessions and worktrees
	./bin/caof teardown --force

docs:                     ## Build MkDocs documentation to site/
	mkdocs build --strict

docs-serve:               ## Serve docs locally with live reload
	mkdocs serve

docs-deploy:              ## Deploy docs to GitHub Pages
	mkdocs gh-deploy --force

help:                     ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: all build test test-go test-python lint lint-go lint-python init run clean

WORKSPACE ?= $(HOME)/caof-workspace
GOAL ?= "Default goal"

all: build

build:                    ## Compile the Go CLI binary
	go build -o bin/caof ./cmd/caof

test: test-go test-python ## Run all tests

test-go:                  ## Run Go tests
	go test ./... -v -count=1

test-python:              ## Run Python tests
	cd agents && python -m pytest tests/ -v

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

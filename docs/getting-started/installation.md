# Installation

This guide walks you through installing CAOF and all of its dependencies on your local machine.

## Prerequisites

Before you begin, make sure you have the following installed:

| Dependency | Minimum Version | Purpose |
|-----------|-----------------|---------|
| **Go** | 1.22+ | Compiles the Control Plane CLI binary |
| **Python** | 3.11+ | Runs the agent runtime |
| **Redis** | 7+ | Event bus and short-term memory |
| **tmux** | 3.3+ | Persistent agent process management |
| **Git** | 2.40+ | Worktree support for task isolation |
| **Make** | Any | Build orchestration |

!!! warning "Version requirements are strict"
    Git worktree features used by CAOF require Git 2.40 or later. Redis Streams consumer group features require Redis 7+. Using older versions will cause runtime failures.

### Verify Prerequisites

```bash
go version        # go1.22 or higher
python3 --version # 3.11 or higher
redis-server -v   # v=7.x.x or higher
tmux -V           # tmux 3.3 or higher
git --version     # 2.40 or higher
make --version    # any
```

## Install CAOF

=== "Modern (uv)"

    [uv](https://github.com/astral-sh/uv) is a fast Python package manager that replaces pip and venv. This is the recommended approach.

    ```bash
    # Install uv if needed
    curl -LsSf https://astral.sh/uv/install.sh | sh

    # Clone and setup
    git clone https://github.com/danielckv/agentic-orchestration.git
    cd agentic-orchestration

    # Build Go CLI
    make build

    # Setup Python environment with uv
    cd agents
    uv venv
    uv pip install -e ".[dev]"
    cd ..

    # Verify
    ./bin/caof --help
    ```

=== "Classic (pip + venv)"

    Standard Python tooling using the built-in `venv` module and pip.

    ```bash
    git clone https://github.com/danielckv/agentic-orchestration.git
    cd agentic-orchestration

    # Build Go CLI
    make build

    # Setup Python environment
    cd agents
    python3 -m venv .venv
    source .venv/bin/activate
    pip install -e ".[dev]"
    cd ..

    # Verify
    ./bin/caof --help
    ```

## Verify the Installation

After installation, confirm that the CLI is working:

```bash
./bin/caof --help
```

You should see output listing all available subcommands:

```
CAOF — Collective Agentic Orchestration Framework

Usage:
  caof [command]

Available Commands:
  init        Bootstrap the CAOF workspace
  spawn       Launch an agent in a tmux session
  run         Submit a goal for execution
  status      Show agent and DAG status
  resume      Resume a blocked task
  teardown    Tear down all sessions and worktrees
  help        Help about any command

Flags:
  -h, --help   help for caof
```

## Start Redis

CAOF requires a running Redis instance. Start one locally:

```bash
# Start Redis in the background
redis-server --daemonize yes

# Verify Redis is running
redis-cli ping
# Expected output: PONG
```

!!! tip "Redis on macOS"
    If you installed Redis via Homebrew, you can start it as a service:
    ```bash
    brew services start redis
    ```

## Next Steps

Your environment is ready. Proceed to the [Quick Start](quickstart.md) guide to bootstrap a workspace and run your first goal.

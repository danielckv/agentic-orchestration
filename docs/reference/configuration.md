# Configuration Reference

CAOF is configured through a combination of YAML files, provider-specific configs, and environment variables. The Go CLI embeds the default configuration at compile time; you can override any value by placing a `defaults.yaml` in the workspace directory.

## defaults.yaml

This is the primary configuration file. All fields shown below are the defaults:

```yaml
redis:
  address: "localhost:6379"       # Redis server address
  password: ""                    # Redis password (empty for no auth)
  db: 0                           # Redis database number

workspace:
  path: "~/caof-workspace"        # Root directory for all CAOF operations
  worktree_dir: ".worktrees"      # Subdirectory for git worktrees

heartbeat:
  interval_seconds: 30            # How often agents send heartbeats
  stale_threshold_seconds: 90     # Seconds without heartbeat before an agent is stale

reflection:
  max_revisions: 3                # Rejections before HITL escalation

inference:
  provider: "openai"              # Default inference provider (llama, anthropic, openai)
  model: "gpt-4o"                 # Default model name
  endpoint: ""                    # Override endpoint URL (empty = use provider default)
  timeout_seconds: 120            # Request timeout

registry:
  port: 9400                      # HTTP port for agent registry and metrics

streams:
  tasks_pending: "tasks.pending"           # Sub-tasks ready for execution
  tasks_claimed: "tasks.claimed"           # Workers announce task acquisition
  artifacts_review: "artifacts.review"     # Artifacts submitted for validation
  artifacts_approved: "artifacts.approved" # Approved artifacts
  artifacts_rejected: "artifacts.rejected" # Rejected artifacts (escalation)
  agents_heartbeat: "agents.heartbeat"     # Agent health pings
  consensus_votes: "consensus.votes"       # Voting for critical decisions
```

### Field Reference

#### redis

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `address` | `string` | `localhost:6379` | Redis server host and port |
| `password` | `string` | `""` | Redis authentication password. Leave empty for unauthenticated connections. |
| `db` | `int` | `0` | Redis database number (0-15) |

#### workspace

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | `string` | `~/caof-workspace` | Root directory for workspace operations. Can be overridden with `caof init --workspace`. |
| `worktree_dir` | `string` | `.worktrees` | Subdirectory within the workspace for git worktrees. Each task gets its own worktree at `{path}/{worktree_dir}/task-{id}/`. |

#### heartbeat

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `interval_seconds` | `int` | `30` | How frequently agents publish heartbeat messages to `agents.heartbeat`. |
| `stale_threshold_seconds` | `int` | `90` | If no heartbeat is received within this window, the agent is deregistered and its tasks are re-queued. Should be at least 2x `interval_seconds`. |

#### reflection

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_revisions` | `int` | `3` | Number of consecutive Reflector rejections before a task escalates to human-in-the-loop. |

#### inference

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `provider` | `string` | `openai` | Default inference provider. One of: `llama`, `anthropic`, `openai`. |
| `model` | `string` | `gpt-4o` | Model name passed to the provider. |
| `endpoint` | `string` | `""` | Override the provider's default endpoint URL. Leave empty to use the standard endpoint. |
| `timeout_seconds` | `int` | `120` | Maximum time to wait for an inference response before retrying. |

#### registry

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | `int` | `9400` | HTTP port for the agent registry API and Prometheus `/metrics` endpoint. |

#### streams

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `tasks_pending` | `string` | `tasks.pending` | Stream name for sub-tasks ready for execution. |
| `tasks_claimed` | `string` | `tasks.claimed` | Stream name for worker task acquisition announcements. |
| `artifacts_review` | `string` | `artifacts.review` | Stream name for artifacts awaiting Reflector validation. |
| `artifacts_approved` | `string` | `artifacts.approved` | Stream name for approved artifacts. |
| `artifacts_rejected` | `string` | `artifacts.rejected` | Stream name for rejected artifacts (triggers escalation). |
| `agents_heartbeat` | `string` | `agents.heartbeat` | Stream name for agent health pings. |
| `consensus_votes` | `string` | `consensus.votes` | Stream name for consensus voting messages. |

## Provider Configuration

Each inference provider has its own YAML file in `config/providers/`. These files specify the connection details for each backend.

### config/providers/llama.yaml

```yaml
provider: "llama"
model: "llama-3.2-70b"
endpoint: "http://localhost:8080"
api_key_env: ""
```

### config/providers/anthropic.yaml

```yaml
provider: "anthropic"
model: "claude-sonnet-4-6"
endpoint: "https://api.anthropic.com"
api_key_env: "ANTHROPIC_API_KEY"
```

### config/providers/openai.yaml

```yaml
provider: "openai"
model: "gpt-4o"
endpoint: "https://api.openai.com"
api_key_env: "OPENAI_API_KEY"
```

### Provider Fields

| Field | Type | Description |
|-------|------|-------------|
| `provider` | `string` | Provider identifier (`llama`, `anthropic`, `openai`) |
| `model` | `string` | Model name or path to pass to the provider |
| `endpoint` | `string` | Base URL for the provider's API |
| `api_key_env` | `string` | Name of the environment variable containing the API key. Empty for providers that do not require authentication (e.g., local Llama). |

## Environment Variables (Python Agents)

Python agents read configuration from environment variables, which take precedence over YAML values:

| Variable | Description | Default |
|----------|-------------|---------|
| `CAOF_AGENT_ID` | Unique identifier for this agent | Auto-generated UUID |
| `CAOF_ROLE` | Agent role (`planner`, `coder`, `researcher`, `reviewer`) | Required |
| `CAOF_REDIS_URL` | Redis connection string | `redis://localhost:6379/0` |
| `CAOF_REGISTRY_URL` | Control Plane registry endpoint | `http://localhost:9400` |
| `CAOF_INFERENCE_PROVIDER` | Inference provider name | From `defaults.yaml` |
| `CAOF_INFERENCE_MODEL` | Model name | From `defaults.yaml` |
| `CAOF_HEARTBEAT_INTERVAL` | Heartbeat interval in seconds | `30` |
| `ANTHROPIC_API_KEY` | Anthropic API key | None |
| `OPENAI_API_KEY` | OpenAI API key | None |

!!! tip "Precedence order"
    Environment variables > workspace `defaults.yaml` > embedded `defaults.yaml`.

## Configuration Files Location

| File | Purpose |
|------|---------|
| `config/defaults.yaml` | Default settings (embedded in Go binary at compile time) |
| `config/providers/llama.yaml` | Llama provider connection config |
| `config/providers/anthropic.yaml` | Anthropic provider connection config |
| `config/providers/openai.yaml` | OpenAI provider connection config |
| `templates/task_decomposition.tmpl` | Prompt template for goal decomposition |
| `templates/reflection_audit.tmpl` | Prompt template for artifact auditing |

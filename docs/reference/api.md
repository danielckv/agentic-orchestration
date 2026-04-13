# API Reference

The Control Plane exposes an internal HTTP API for agent registration, discovery, and task introspection. By default, it runs on port `9400` (configurable via `registry.port` in `defaults.yaml`).

## Base URL

```
http://localhost:9400
```

## Agent Registry

### List All Agents

Returns all currently registered agents.

```
GET /registry/agents
```

**Response** `200 OK`:

```json
[
  {
    "agent_id": "coder-01",
    "role": "coder",
    "capabilities": ["python", "file_io", "git"],
    "model": "llama-local",
    "max_concurrent_tasks": 2,
    "current_load": 1,
    "session": "caof-coder-01",
    "pid": 48291
  },
  {
    "agent_id": "reviewer-01",
    "role": "reviewer",
    "capabilities": ["reflection", "diff_audit", "consensus"],
    "model": "gpt-4o",
    "max_concurrent_tasks": 3,
    "current_load": 0,
    "session": "caof-reviewer-01",
    "pid": 48305
  }
]
```

### Get Agent Details

Returns details and current load for a specific agent.

```
GET /registry/agents/{id}
```

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | `string` | Agent identifier (e.g., `coder-01`) |

**Response** `200 OK`:

```json
{
  "agent_id": "coder-01",
  "role": "coder",
  "capabilities": ["python", "file_io", "git"],
  "model": "llama-local",
  "max_concurrent_tasks": 2,
  "current_load": 1,
  "session": "caof-coder-01",
  "pid": 48291
}
```

**Response** `404 Not Found`:

```json
{
  "error": "agent not found",
  "agent_id": "coder-99"
}
```

### Register Agent

Agents call this endpoint on startup to register with the Control Plane.

```
POST /registry/agents
```

**Request Body:**

```json
{
  "agent_id": "coder-01",
  "role": "coder",
  "capabilities": ["python", "file_io", "git"],
  "model": "llama-local",
  "max_concurrent_tasks": 2,
  "current_load": 0,
  "session": "caof-coder-01",
  "pid": 48291
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `agent_id` | `string` | Yes | Unique agent identifier |
| `role` | `string` | Yes | One of: `planner`, `coder`, `researcher`, `reviewer` |
| `capabilities` | `string[]` | Yes | List of capability tags |
| `model` | `string` | Yes | Inference model this agent uses |
| `max_concurrent_tasks` | `int` | Yes | Maximum tasks this agent can handle simultaneously |
| `current_load` | `int` | Yes | Number of tasks currently being processed |
| `session` | `string` | Yes | tmux session name |
| `pid` | `int` | Yes | OS process ID |

**Response** `201 Created`:

```json
{
  "status": "registered",
  "agent_id": "coder-01"
}
```

### Heartbeat

Agents call this periodically (default: every 30 seconds) to signal liveness.

```
PUT /registry/agents/{id}/heartbeat
```

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | `string` | Agent identifier |

**Response** `200 OK`:

```json
{
  "status": "ok",
  "agent_id": "coder-01"
}
```

### Deregister Agent

Gracefully removes an agent from the registry. Called during shutdown or teardown.

```
DELETE /registry/agents/{id}
```

**Response** `200 OK`:

```json
{
  "status": "deregistered",
  "agent_id": "coder-01"
}
```

## Task Introspection

### List Active Tasks

Returns all tasks and their current states.

```
GET /registry/tasks
```

**Response** `200 OK`:

```json
[
  {
    "task_id": "abc-123",
    "parent_task_id": null,
    "role_required": "coder",
    "state": "running",
    "assigned_agent": "coder-01",
    "priority": "high",
    "description": "Implement sorting algorithm",
    "revision_count": 0
  },
  {
    "task_id": "def-456",
    "parent_task_id": "abc-123",
    "role_required": "reviewer",
    "state": "pending",
    "assigned_agent": null,
    "priority": "normal",
    "description": "Review sorting implementation",
    "revision_count": 0
  }
]
```

### Get DAG Visualization Data

Returns the DAG structure for a given goal, suitable for rendering.

```
GET /registry/dag/{id}
```

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | `string` | DAG identifier (matches the root goal's task ID) |

**Response** `200 OK`:

```json
{
  "dag_id": "goal-789",
  "nodes": [
    {
      "task_id": "abc-123",
      "description": "Implement sorting algorithm",
      "role_required": "coder",
      "state": "running",
      "depth": 1,
      "index": 0
    },
    {
      "task_id": "def-456",
      "description": "Write unit tests",
      "role_required": "coder",
      "state": "pending",
      "depth": 2,
      "index": 0
    }
  ],
  "edges": [
    {
      "from": "abc-123",
      "to": "def-456"
    }
  ]
}
```

## Metrics Endpoint

Prometheus-compatible metrics are exposed at:

```
GET /metrics
```

See [Monitoring](../operations/monitoring.md) for the full list of available metrics.

## Stream Message Schemas

These are the message formats used on the Redis Streams event bus. They are not HTTP endpoints, but are included here for reference.

### TaskMessage

**Stream**: `tasks.pending`

```json
{
  "task_id": "string (UUID v4)",
  "parent_task_id": "string (UUID v4) | null",
  "dag_position": {
    "depth": "int",
    "index": "int"
  },
  "role_required": "string (planner | coder | researcher | reviewer)",
  "priority": "string (low | normal | high | critical)",
  "spec": {
    "description": "string",
    "constraints": ["string"],
    "acceptance_criteria": ["string"]
  },
  "context_refs": ["string (rag:// or memory:// URIs)"],
  "created_at": "string (ISO 8601)",
  "ttl_seconds": "int"
}
```

### ArtifactMessage

**Stream**: `artifacts.review`, `artifacts.approved`, `artifacts.rejected`

```json
{
  "artifact_id": "string (UUID v4)",
  "task_id": "string (UUID v4)",
  "agent_id": "string",
  "content": "string",
  "content_type": "string (code_patch | research_summary | data_export | generic)",
  "confidence": "float (0.0 - 1.0)",
  "metadata": {
    "key": "value"
  },
  "timestamp": "string (ISO 8601)"
}
```

### HeartbeatMessage

**Stream**: `agents.heartbeat`

```json
{
  "agent_id": "string",
  "role": "string",
  "status": "string (idle | busy | blocked)",
  "current_task_id": "string | null",
  "load": "int",
  "max_load": "int",
  "timestamp": "string (ISO 8601)"
}
```

### VoteMessage

**Stream**: `consensus.votes`

```json
{
  "decision_id": "string (UUID v4)",
  "voter_agent_id": "string",
  "option_selected": "string",
  "confidence": "float (0.0 - 1.0)",
  "rationale": "string",
  "references": ["string (rag:// URIs)"]
}
```

## Error Responses

All endpoints return errors in a consistent format:

```json
{
  "error": "string (error description)",
  "code": "string (error code, optional)",
  "details": "string (additional context, optional)"
}
```

Common HTTP status codes:

| Code | Meaning |
|------|---------|
| `200` | Success |
| `201` | Resource created |
| `400` | Invalid request body or parameters |
| `404` | Resource not found |
| `409` | Conflict (e.g., agent ID already registered) |
| `500` | Internal server error |

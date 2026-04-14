"""Task lifecycle management for CAOF MCP server.

Each task gets a folder under ~/caof-tasks/<task-id>/ with:
  - task.json        — metadata (status, type, description, timestamps)
  - result.md        — final output / generated content
  - research/        — raw research files (web pages, notes)
  - post.md          — social media post (for social content tasks)
"""

from __future__ import annotations

import json
import logging
import os
import uuid
from datetime import datetime, timezone
from enum import Enum
from pathlib import Path
from typing import Any

from pydantic import BaseModel, Field

logger = logging.getLogger(__name__)

TASKS_ROOT = Path.home() / "caof-tasks"


class TaskType(str, Enum):
    RESEARCH = "research"
    SOCIAL_CONTENT = "social_content"
    GENERIC = "generic"


class TaskStatus(str, Enum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"


class TaskRecord(BaseModel):
    task_id: str = Field(default_factory=lambda: uuid.uuid4().hex[:12])
    task_type: TaskType = TaskType.GENERIC
    description: str = ""
    status: TaskStatus = TaskStatus.PENDING
    created_at: str = Field(default_factory=lambda: datetime.now(timezone.utc).isoformat())
    completed_at: str | None = None
    error: str | None = None
    # For social content tasks
    platform: str | None = None  # twitter, linkedin, both
    topic: str | None = None
    # For research tasks
    web_context: list[str] = Field(default_factory=list)  # URLs or search queries

    @property
    def folder(self) -> Path:
        return TASKS_ROOT / self.task_id

    def save(self) -> None:
        self.folder.mkdir(parents=True, exist_ok=True)
        (self.folder / "task.json").write_text(
            self.model_dump_json(indent=2)
        )

    @classmethod
    def load(cls, task_id: str) -> TaskRecord:
        path = TASKS_ROOT / task_id / "task.json"
        return cls.model_validate_json(path.read_text())

    def set_running(self) -> None:
        self.status = TaskStatus.RUNNING
        self.save()

    def set_completed(self) -> None:
        self.status = TaskStatus.COMPLETED
        self.completed_at = datetime.now(timezone.utc).isoformat()
        self.save()

    def set_failed(self, error: str) -> None:
        self.status = TaskStatus.FAILED
        self.error = error
        self.completed_at = datetime.now(timezone.utc).isoformat()
        self.save()

    def write_result(self, content: str, filename: str = "result.md") -> Path:
        path = self.folder / filename
        path.write_text(content)
        return path

    def write_research(self, content: str, name: str) -> Path:
        research_dir = self.folder / "research"
        research_dir.mkdir(exist_ok=True)
        path = research_dir / name
        path.write_text(content)
        return path

    def list_files(self) -> list[str]:
        if not self.folder.exists():
            return []
        files = []
        for p in sorted(self.folder.rglob("*")):
            if p.is_file():
                files.append(str(p.relative_to(self.folder)))
        return files

    def read_file(self, filename: str) -> str:
        path = self.folder / filename
        if not path.exists():
            raise FileNotFoundError(f"{filename} not found in task {self.task_id}")
        return path.read_text()


def list_all_tasks() -> list[TaskRecord]:
    """List all tasks from disk."""
    if not TASKS_ROOT.exists():
        return []
    tasks = []
    for d in sorted(TASKS_ROOT.iterdir()):
        meta = d / "task.json"
        if meta.exists():
            try:
                tasks.append(TaskRecord.model_validate_json(meta.read_text()))
            except Exception as e:
                logger.warning(f"Skipping bad task {d.name}: {e}")
    return tasks

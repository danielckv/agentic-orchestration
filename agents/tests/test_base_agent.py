import pytest
from unittest.mock import MagicMock, patch
from shared.base_agent import BaseAgent
from shared.config import AgentConfig
from shared.schemas import TaskMessage, TaskSpec


class MockAgent(BaseAgent):
    def __init__(self, config):
        # Don't connect to real Redis in tests
        self.config = config
        if not config.agent_id:
            self.config.agent_id = f"{config.role}-test"
        self.running = False
        self.current_task_id = None
        self.processed_tasks = []
        self._consumer_group = f"{config.role}-group"
        self._consumer_name = self.config.agent_id

    def on_task(self, task: TaskMessage) -> None:
        self.processed_tasks.append(task)


def test_agent_config_and_identity():
    config = AgentConfig(role="coder", model="test-model")
    agent = MockAgent(config)
    assert agent.config.role == "coder"
    assert agent.config.agent_id == "coder-test"


def test_on_task_called():
    config = AgentConfig(agent_id="test-agent", role="coder")
    agent = MockAgent(config)
    task = TaskMessage(
        task_id="t-1",
        role_required="coder",
        spec=TaskSpec(description="test task"),
    )
    agent.on_task(task)
    assert len(agent.processed_tasks) == 1
    assert agent.processed_tasks[0].task_id == "t-1"

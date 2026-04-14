"""Thin inference wrapper for the MCP server.

Reads provider config from the project's config/ directory and calls
the configured LLM (Anthropic, OpenAI, or local Llama).
"""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass, field
from pathlib import Path

import httpx
import yaml

logger = logging.getLogger(__name__)

# Resolve project root relative to this file
_PROJECT_ROOT = Path(__file__).resolve().parent.parent
_CONFIG_PATH = _PROJECT_ROOT / "config" / "defaults.yaml"


@dataclass
class InferenceConfig:
    provider: str = "anthropic"
    model: str = "claude-sonnet-4-6"
    endpoint: str = ""
    api_key: str = ""
    timeout: int = 120

    @classmethod
    def from_project(cls) -> InferenceConfig:
        """Load from project config/defaults.yaml + provider overlay."""
        cfg = cls()
        if _CONFIG_PATH.exists():
            data = yaml.safe_load(_CONFIG_PATH.read_text())
            inf = data.get("inference", {})
            cfg.provider = inf.get("provider", cfg.provider)
            cfg.model = inf.get("model", cfg.model)
            cfg.timeout = inf.get("timeout_seconds", cfg.timeout)

        provider_path = _PROJECT_ROOT / "config" / "providers" / f"{cfg.provider}.yaml"
        if provider_path.exists():
            pdata = yaml.safe_load(provider_path.read_text())
            cfg.endpoint = pdata.get("endpoint", cfg.endpoint)
            cfg.model = pdata.get("model", cfg.model)
            key_env = pdata.get("api_key_env", "")
            if key_env:
                cfg.api_key = os.getenv(key_env, "")

        # Allow env overrides
        cfg.provider = os.getenv("CAOF_INFERENCE_PROVIDER", cfg.provider)
        cfg.model = os.getenv("CAOF_INFERENCE_MODEL", cfg.model)
        cfg.api_key = os.getenv("CAOF_INFERENCE_API_KEY", cfg.api_key)
        cfg.endpoint = os.getenv("CAOF_INFERENCE_ENDPOINT", cfg.endpoint)
        return cfg


class LLM:
    """Provider-agnostic LLM caller."""

    def __init__(self, config: InferenceConfig | None = None):
        self.config = config or InferenceConfig.from_project()
        self._client = httpx.Client(timeout=self.config.timeout)

    def complete(
        self,
        prompt: str,
        system: str = "",
        max_tokens: int = 4096,
        temperature: float = 0.7,
    ) -> str:
        messages = []
        if system:
            messages.append({"role": "system", "content": system})
        messages.append({"role": "user", "content": prompt})

        if self.config.provider == "anthropic":
            return self._anthropic(messages, max_tokens, temperature)
        elif self.config.provider == "openai":
            return self._openai(messages, max_tokens, temperature)
        elif self.config.provider == "llama":
            return self._llama(messages, max_tokens, temperature)
        raise ValueError(f"Unknown provider: {self.config.provider}")

    def _anthropic(self, messages: list, max_tokens: int, temperature: float) -> str:
        system = ""
        filtered = []
        for m in messages:
            if m["role"] == "system":
                system = m["content"]
            else:
                filtered.append(m)
        body: dict = {
            "model": self.config.model,
            "messages": filtered,
            "max_tokens": max_tokens,
            "temperature": temperature,
        }
        if system:
            body["system"] = system
        endpoint = self.config.endpoint or "https://api.anthropic.com"
        resp = self._client.post(
            f"{endpoint}/v1/messages",
            headers={
                "x-api-key": self.config.api_key,
                "anthropic-version": "2023-06-01",
                "Content-Type": "application/json",
            },
            json=body,
        )
        resp.raise_for_status()
        return resp.json()["content"][0]["text"]

    def _openai(self, messages: list, max_tokens: int, temperature: float) -> str:
        endpoint = self.config.endpoint or "https://api.openai.com"
        resp = self._client.post(
            f"{endpoint}/v1/chat/completions",
            headers={
                "Authorization": f"Bearer {self.config.api_key}",
                "Content-Type": "application/json",
            },
            json={
                "model": self.config.model,
                "messages": messages,
                "max_tokens": max_tokens,
                "temperature": temperature,
            },
        )
        resp.raise_for_status()
        return resp.json()["choices"][0]["message"]["content"]

    def _llama(self, messages: list, max_tokens: int, temperature: float) -> str:
        prompt = "\n".join(f"{m['role']}: {m['content']}" for m in messages)
        endpoint = self.config.endpoint or "http://localhost:8080"
        resp = self._client.post(
            f"{endpoint}/completion",
            json={
                "prompt": prompt,
                "n_predict": max_tokens,
                "temperature": temperature,
            },
        )
        resp.raise_for_status()
        return resp.json().get("content", "")

    def close(self) -> None:
        self._client.close()

import json
import logging
import os
from dataclasses import dataclass

import httpx
import yaml

logger = logging.getLogger(__name__)

@dataclass
class InferenceConfig:
    provider: str = "openai"
    model: str = "gpt-4o"
    endpoint: str = ""
    api_key: str = ""
    timeout: int = 120

    @classmethod
    def from_yaml(cls, path: str, provider_name: str = "") -> "InferenceConfig":
        with open(path) as f:
            data = yaml.safe_load(f)
        inf = data.get("inference", {})
        provider_name = provider_name or inf.get("provider", "openai")
        cfg = cls(provider=provider_name, model=inf.get("model", "gpt-4o"), timeout=inf.get("timeout_seconds", 120))
        # Try loading provider-specific config
        provider_path = os.path.join(os.path.dirname(path), "providers", f"{provider_name}.yaml")
        if os.path.exists(provider_path):
            with open(provider_path) as f:
                pdata = yaml.safe_load(f)
            cfg.endpoint = pdata.get("endpoint", "")
            cfg.model = pdata.get("model", cfg.model)
            key_env = pdata.get("api_key_env", "")
            if key_env:
                cfg.api_key = os.getenv(key_env, "")
        return cfg


class InferenceClient:
    """Provider-agnostic inference client."""

    def __init__(self, config: InferenceConfig):
        self.config = config
        self._client = httpx.Client(timeout=config.timeout)

    def complete(self, messages: list[dict], max_tokens: int = 4096, temperature: float = 0.7) -> str:
        """Send completion request, return content string."""
        if self.config.provider == "openai":
            return self._openai_complete(messages, max_tokens, temperature)
        elif self.config.provider == "anthropic":
            return self._anthropic_complete(messages, max_tokens, temperature)
        elif self.config.provider == "llama":
            return self._llama_complete(messages, max_tokens, temperature)
        raise ValueError(f"Unknown provider: {self.config.provider}")

    def _openai_complete(self, messages, max_tokens, temperature):
        resp = self._client.post(
            f"{self.config.endpoint}/v1/chat/completions",
            headers={"Authorization": f"Bearer {self.config.api_key}", "Content-Type": "application/json"},
            json={"model": self.config.model, "messages": messages, "max_tokens": max_tokens, "temperature": temperature},
        )
        resp.raise_for_status()
        return resp.json()["choices"][0]["message"]["content"]

    def _anthropic_complete(self, messages, max_tokens, temperature):
        system = ""
        filtered = []
        for m in messages:
            if m["role"] == "system":
                system = m["content"]
            else:
                filtered.append(m)
        body = {"model": self.config.model, "messages": filtered, "max_tokens": max_tokens, "temperature": temperature}
        if system:
            body["system"] = system
        resp = self._client.post(
            f"{self.config.endpoint}/v1/messages",
            headers={"x-api-key": self.config.api_key, "anthropic-version": "2023-06-01", "Content-Type": "application/json"},
            json=body,
        )
        resp.raise_for_status()
        return resp.json()["content"][0]["text"]

    def _llama_complete(self, messages, max_tokens, temperature):
        prompt = "\n".join(f"{m['role']}: {m['content']}" for m in messages)
        resp = self._client.post(
            f"{self.config.endpoint}/completion",
            json={"prompt": prompt, "n_predict": max_tokens, "temperature": temperature},
        )
        resp.raise_for_status()
        return resp.json().get("content", "")

    def close(self):
        self._client.close()

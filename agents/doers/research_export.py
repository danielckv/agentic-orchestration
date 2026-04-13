import json
import logging
import os
import uuid
from datetime import datetime

from shared.base_agent import BaseAgent
from shared.config import AgentConfig
from shared.inference import InferenceClient, InferenceConfig
from shared.schemas import TaskMessage
from shared.export import export_research

logger = logging.getLogger(__name__)

RESEARCH_EXPORT_PROMPT = """You are a research agent. Research the following topic thoroughly.

Topic: {topic}
Constraints: {constraints}

Return your findings as a JSON array of research entries. Each entry must have:
- "topic": subtopic name
- "summary": 2-3 sentence summary
- "key_findings": array of bullet points
- "sources": array of source descriptions
- "confidence": confidence level "high", "medium", or "low"
- "category": category/tag for grouping

Return ONLY valid JSON array, no markdown."""

class ResearchExportDoer(BaseAgent):
    """Agent that researches topics and exports results to Excel/CSV/JSON."""

    def __init__(self, config: AgentConfig, inference_config: InferenceConfig, output_dir: str = "./research_output"):
        super().__init__(config)
        self.inference = InferenceClient(inference_config)
        self.output_dir = output_dir
        os.makedirs(output_dir, exist_ok=True)

    def on_task(self, task: TaskMessage) -> None:
        topic = task.spec.description
        constraints = task.spec.constraints
        output_format = "excel"  # default

        # Check if format is specified in constraints
        for c in constraints:
            if c.startswith("format:"):
                output_format = c.split(":", 1)[1].strip().lower()

        logger.info(f"Researching: {topic}")

        # Get research from LLM
        prompt = RESEARCH_EXPORT_PROMPT.format(topic=topic, constraints=", ".join(constraints))
        response = self.inference.complete([
            {"role": "system", "content": "You are a thorough research agent. Return structured JSON data."},
            {"role": "user", "content": prompt},
        ])

        # Parse response
        try:
            research_data = json.loads(response)
        except json.JSONDecodeError:
            if "```" in response:
                json_str = response.split("```")[1]
                if json_str.startswith("json"):
                    json_str = json_str[4:]
                research_data = json.loads(json_str.strip())
            else:
                research_data = [{"topic": topic, "summary": response, "key_findings": [], "sources": [], "confidence": "medium", "category": "general"}]

        # Flatten key_findings for tabular format
        export_data = []
        for entry in research_data:
            findings = entry.get("key_findings", [])
            export_data.append({
                "topic": entry.get("topic", ""),
                "summary": entry.get("summary", ""),
                "key_findings": "; ".join(findings) if isinstance(findings, list) else str(findings),
                "sources": "; ".join(entry.get("sources", [])) if isinstance(entry.get("sources"), list) else str(entry.get("sources", "")),
                "confidence": entry.get("confidence", "medium"),
                "category": entry.get("category", ""),
                "researched_at": datetime.utcnow().isoformat(),
            })

        # Generate output filename
        safe_topic = "".join(c if c.isalnum() or c in " -_" else "" for c in topic)[:50].strip().replace(" ", "_")
        timestamp = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
        ext_map = {"excel": ".xlsx", "csv": ".csv", "json": ".json", "markdown": ".md"}
        ext = ext_map.get(output_format, ".xlsx")
        filename = f"research_{safe_topic}_{timestamp}{ext}"
        output_path = os.path.join(self.output_dir, filename)

        # Export
        exported_path = export_research(export_data, output_path, format=output_format)

        # Submit artifact with path to exported file
        self.submit_artifact(
            task_id=task.task_id,
            content=json.dumps({
                "type": "research_export",
                "topic": topic,
                "entries_count": len(export_data),
                "output_path": exported_path,
                "format": output_format,
                "data": export_data,
            }, ensure_ascii=False),
            confidence=0.85,
        )
        logger.info(f"Research exported to {exported_path} ({len(export_data)} entries)")


if __name__ == "__main__":
    config = AgentConfig.from_env()
    if not config.role:
        config.role = "researcher"
    inference_cfg = InferenceConfig()
    agent = ResearchExportDoer(config, inference_cfg)
    agent.start()

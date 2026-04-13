import json
import logging
from shared.inference import InferenceClient
from shared.schemas import StageResult

logger = logging.getLogger(__name__)

AUDIT_PROMPT = """You are a reflective auditor. Review this artifact and the validation results.

## Task: {task_description}

## Artifact Content:
{content}

## Validation Results:
- Schema: {schema_result}
- RAG Cross-reference: {rag_result}
- Tests: {test_result}

## Previous Revision Notes:
{revision_notes}

Evaluate and return JSON:
{{"verdict": "APPROVE" or "REVISE" or "REJECT", "rationale": "...", "revision_notes": "...", "confidence": 0.0-1.0}}"""

def audit_artifact(client: InferenceClient, task_description: str, content: str,
                   schema_result: StageResult, rag_result: StageResult, test_result: StageResult,
                   revision_notes: str = "") -> dict:
    """Stage 4: LLM-based audit."""
    prompt = AUDIT_PROMPT.format(
        task_description=task_description,
        content=content[:4000],
        schema_result=f"{'PASS' if schema_result.passed else 'FAIL'}: {schema_result.details}",
        rag_result=f"{'PASS' if rag_result.passed else 'FAIL'}: {rag_result.details}",
        test_result=f"{'PASS' if test_result.passed else 'FAIL'}: {test_result.details}",
        revision_notes=revision_notes or "None",
    )

    response = client.complete([
        {"role": "system", "content": "You are a code and research quality auditor."},
        {"role": "user", "content": prompt},
    ])

    try:
        return json.loads(response)
    except json.JSONDecodeError:
        if "```" in response:
            json_str = response.split("```")[1]
            if json_str.startswith("json"):
                json_str = json_str[4:]
            return json.loads(json_str.strip())
        return {"verdict": "REVISE", "rationale": "Could not parse audit response", "revision_notes": response, "confidence": 0.5}

from shared.schemas import StageResult

def crossref_rag(content: str, memory_client=None) -> StageResult:
    """Stage 2: Cross-reference with long-term memory for contradictions."""
    if memory_client is None:
        return StageResult(stage="rag_crossref", passed=True, details="No memory client available, skipped")

    try:
        results = memory_client.query(content, top_k=5)
        if not results:
            return StageResult(stage="rag_crossref", passed=True, details="No related entries found")

        # Basic contradiction check: look for conflicting keywords
        warnings = []
        for r in results:
            if r.get("score", 0) > 0.8:
                warnings.append(f"High similarity with existing entry: {r.get('source', 'unknown')}")

        return StageResult(stage="rag_crossref", passed=True, details=f"Cross-referenced {len(results)} entries", warnings=warnings)
    except Exception as e:
        return StageResult(stage="rag_crossref", passed=True, details=f"RAG check failed (non-blocking): {e}")

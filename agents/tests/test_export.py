import json
import os
import tempfile
from shared.export import export_research

SAMPLE_DATA = [
    {
        "topic": "Transformer Attention",
        "summary": "Multi-head attention is the core mechanism.",
        "key_findings": "Scales quadratically; Various optimization approaches exist",
        "sources": "Vaswani et al. 2017",
        "confidence": "high",
        "category": "ML Architecture",
    },
    {
        "topic": "Flash Attention",
        "summary": "Hardware-aware attention computation.",
        "key_findings": "IO-aware; Reduces memory from O(N^2) to O(N)",
        "sources": "Dao et al. 2022",
        "confidence": "high",
        "category": "Optimization",
    },
    {
        "topic": "Sparse Attention",
        "summary": "Reduces computation by attending to subset.",
        "key_findings": "Linear scaling possible; Trade-off with quality",
        "sources": "Child et al. 2019",
        "confidence": "medium",
        "category": "Optimization",
    },
]

def test_export_excel():
    with tempfile.TemporaryDirectory() as tmpdir:
        path = os.path.join(tmpdir, "test.xlsx")
        result = export_research(SAMPLE_DATA, path, format="excel")
        assert os.path.exists(result)
        assert result.endswith(".xlsx") or result.endswith(".csv")  # may fallback to csv

def test_export_csv():
    with tempfile.TemporaryDirectory() as tmpdir:
        path = os.path.join(tmpdir, "test.csv")
        result = export_research(SAMPLE_DATA, path, format="csv")
        assert os.path.exists(result)
        with open(result) as f:
            lines = f.readlines()
        assert len(lines) == 4  # header + 3 rows

def test_export_json():
    with tempfile.TemporaryDirectory() as tmpdir:
        path = os.path.join(tmpdir, "test.json")
        result = export_research(SAMPLE_DATA, path, format="json")
        assert os.path.exists(result)
        with open(result) as f:
            data = json.load(f)
        assert data["count"] == 3
        assert len(data["results"]) == 3

def test_export_markdown():
    with tempfile.TemporaryDirectory() as tmpdir:
        path = os.path.join(tmpdir, "test.md")
        result = export_research(SAMPLE_DATA, path, format="markdown")
        assert os.path.exists(result)
        with open(result) as f:
            content = f.read()
        assert "Transformer Attention" in content
        assert "| Topic |" in content  # table header

def test_export_empty():
    with tempfile.TemporaryDirectory() as tmpdir:
        path = os.path.join(tmpdir, "empty.xlsx")
        result = export_research([], path)
        assert os.path.exists(result)

def test_export_auto_format():
    with tempfile.TemporaryDirectory() as tmpdir:
        path = os.path.join(tmpdir, "auto.csv")
        result = export_research(SAMPLE_DATA, path)  # format="auto" detects from extension
        assert result.endswith(".csv")

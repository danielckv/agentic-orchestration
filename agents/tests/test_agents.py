from reasoning.strategy_selector import select_strategy
from reflectors.schema_validator import validate_schema
from reflectors.security_scan import scan_secrets
from shared.schemas import ArtifactMessage, ArtifactMetadata, StageResult
from doers.tools.file_io import WorktreeFileIO
import tempfile, os

def test_strategy_selector_cot():
    assert select_strategy("Write a sorting function") == "cot"

def test_strategy_selector_tot():
    assert select_strategy("Compare and evaluate multiple approaches to caching") == "tot"

def test_schema_validator_pass():
    artifact = ArtifactMessage(
        artifact_id="a1", task_id="t1", agent_id="coder-01",
        content="print('hello')", metadata=ArtifactMetadata(confidence=0.9),
    )
    result = validate_schema(artifact)
    assert result.passed

def test_schema_validator_empty_content():
    artifact = ArtifactMessage(
        artifact_id="a1", task_id="t1", agent_id="coder-01",
        content="", metadata=ArtifactMetadata(confidence=0.9),
    )
    result = validate_schema(artifact)
    assert not result.passed

def test_security_scan_clean():
    result = scan_secrets("print('hello world')")
    assert result.passed

def test_security_scan_detects_key():
    result = scan_secrets('api_key = "sk-abcdefghijklmnopqrstuvwxyz123456"')
    assert not result.passed

def test_file_io_safety():
    with tempfile.TemporaryDirectory() as tmpdir:
        fio = WorktreeFileIO(tmpdir)
        fio.write("test.txt", "hello")
        assert fio.read("test.txt") == "hello"
        assert fio.exists("test.txt")
        try:
            fio.read("../../etc/passwd")
            assert False, "Should have raised"
        except ValueError:
            pass

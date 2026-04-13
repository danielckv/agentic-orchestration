import subprocess
import logging

logger = logging.getLogger(__name__)

def run_in_sandbox(command: str, cwd: str, timeout: int = 30) -> tuple[str, str, int]:
    """Run a command in a sandboxed subprocess. Returns (stdout, stderr, returncode)."""
    try:
        result = subprocess.run(
            command, shell=True, cwd=cwd, capture_output=True, text=True, timeout=timeout,
        )
        return result.stdout, result.stderr, result.returncode
    except subprocess.TimeoutExpired:
        return "", f"Command timed out after {timeout}s", -1

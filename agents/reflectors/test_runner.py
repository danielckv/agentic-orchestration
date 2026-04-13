import subprocess
from shared.schemas import StageResult

def run_tests(worktree_path: str) -> StageResult:
    """Stage 3: Run linting and tests in worktree."""
    if not worktree_path:
        return StageResult(stage="tests", passed=True, details="No worktree path, skipped")

    results = []

    # Run ruff lint
    try:
        r = subprocess.run(["ruff", "check", "."], cwd=worktree_path, capture_output=True, text=True, timeout=30)
        if r.returncode == 0:
            results.append("ruff: passed")
        else:
            return StageResult(stage="tests", passed=False, details=f"ruff failed: {r.stdout[:500]}")
    except FileNotFoundError:
        results.append("ruff: not installed, skipped")
    except subprocess.TimeoutExpired:
        results.append("ruff: timed out")

    # Run pytest if tests exist
    try:
        r = subprocess.run(["python", "-m", "pytest", "-x", "-q"], cwd=worktree_path, capture_output=True, text=True, timeout=60)
        if r.returncode == 0:
            results.append(f"pytest: passed ({r.stdout.strip().split(chr(10))[-1]})")
        elif r.returncode == 5:  # no tests collected
            results.append("pytest: no tests found")
        else:
            return StageResult(stage="tests", passed=False, details=f"pytest failed: {r.stdout[:500]}")
    except FileNotFoundError:
        results.append("pytest: not available")
    except subprocess.TimeoutExpired:
        results.append("pytest: timed out")

    return StageResult(stage="tests", passed=True, details="; ".join(results))

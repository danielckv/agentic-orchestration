import re
from shared.schemas import StageResult

SECRET_PATTERNS = [
    (r'(?i)(api[_-]?key|apikey)\s*[=:]\s*["\']?[a-zA-Z0-9_\-]{20,}', "Possible API key"),
    (r'(?i)(secret|password|passwd|pwd)\s*[=:]\s*["\']?[^\s"\']{8,}', "Possible secret/password"),
    (r'(?i)(aws_access_key_id|aws_secret)\s*[=:]\s*["\']?[A-Z0-9]{16,}', "AWS credential"),
    (r'sk-[a-zA-Z0-9]{20,}', "OpenAI-style API key"),
    (r'ghp_[a-zA-Z0-9]{36}', "GitHub personal access token"),
]

def scan_secrets(content: str) -> StageResult:
    """Scan content for hardcoded secrets and sensitive patterns."""
    findings = []
    for pattern, description in SECRET_PATTERNS:
        matches = re.findall(pattern, content)
        if matches:
            findings.append(f"{description}: {len(matches)} occurrence(s)")

    if findings:
        return StageResult(stage="security", passed=False, details="Security issues found", warnings=findings)
    return StageResult(stage="security", passed=True, details="No secrets detected")

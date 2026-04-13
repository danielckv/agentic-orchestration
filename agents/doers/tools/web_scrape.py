import logging
import httpx

logger = logging.getLogger(__name__)

def fetch_url(url: str, timeout: int = 30) -> str:
    """Fetch content from a URL. Basic implementation."""
    resp = httpx.get(url, timeout=timeout, follow_redirects=True)
    resp.raise_for_status()
    return resp.text

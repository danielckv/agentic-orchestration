"""Web research utilities for CAOF MCP tasks.

Fetches web pages and extracts text content for research tasks.
The MCP client (e.g. ClawdBot) can also pass pre-fetched web context
via the task description — this module handles the supplemental
live-fetch cases.
"""

from __future__ import annotations

import logging
import re
from urllib.parse import quote_plus

import httpx

logger = logging.getLogger(__name__)

_TIMEOUT = 20
_HEADERS = {
    "User-Agent": "CAOF-Research/0.1 (research bot)",
    "Accept": "text/html,application/xhtml+xml,text/plain",
}


def fetch_page(url: str) -> str:
    """Fetch a URL and return raw text (HTML stripped to plaintext-ish)."""
    try:
        resp = httpx.get(url, headers=_HEADERS, timeout=_TIMEOUT, follow_redirects=True)
        resp.raise_for_status()
        return _strip_html(resp.text)
    except Exception as e:
        logger.warning(f"Failed to fetch {url}: {e}")
        return f"[ERROR fetching {url}: {e}]"


def search_web(query: str, num_results: int = 5) -> list[dict[str, str]]:
    """Search the web using DuckDuckGo HTML endpoint.

    Returns list of {"title": ..., "url": ..., "snippet": ...}.
    """
    try:
        resp = httpx.get(
            f"https://html.duckduckgo.com/html/?q={quote_plus(query)}",
            headers=_HEADERS,
            timeout=_TIMEOUT,
            follow_redirects=True,
        )
        resp.raise_for_status()
        return _parse_ddg_results(resp.text, num_results)
    except Exception as e:
        logger.warning(f"Web search failed for '{query}': {e}")
        return []


def _parse_ddg_results(html: str, limit: int) -> list[dict[str, str]]:
    """Extract results from DuckDuckGo HTML search page."""
    results = []
    # Match result links — DuckDuckGo wraps them in <a class="result__a">
    link_pattern = re.compile(
        r'<a[^>]*class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>', re.DOTALL
    )
    snippet_pattern = re.compile(
        r'<a[^>]*class="result__snippet"[^>]*>(.*?)</a>', re.DOTALL
    )

    links = link_pattern.findall(html)
    snippets = snippet_pattern.findall(html)

    for i, (url, title) in enumerate(links[:limit]):
        snippet = _strip_html(snippets[i]) if i < len(snippets) else ""
        results.append({
            "title": _strip_html(title).strip(),
            "url": url,
            "snippet": snippet.strip(),
        })
    return results


def _strip_html(text: str) -> str:
    """Rough HTML-to-text."""
    text = re.sub(r"<script[^>]*>.*?</script>", "", text, flags=re.DOTALL)
    text = re.sub(r"<style[^>]*>.*?</style>", "", text, flags=re.DOTALL)
    text = re.sub(r"<[^>]+>", " ", text)
    text = re.sub(r"\s+", " ", text)
    return text.strip()

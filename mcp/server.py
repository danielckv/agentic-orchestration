"""CAOF MCP Server — task execution via Model Context Protocol.

Exposes tools for starting research, social-content, and generic tasks
from any MCP client (e.g. ClawdBot on WhatsApp).

Each task runs asynchronously:
  1. Creates ~/caof-tasks/<task-id>/ folder
  2. Performs web research if requested
  3. Calls the configured LLM to produce output
  4. Saves results to the task folder
  5. Returns summary + file listing

Usage:
    python -m server            # stdio transport (default)
    python -m server --sse      # SSE transport on port 8765
"""

from __future__ import annotations

import asyncio
import json
import logging
import sys
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone

from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import TextContent, Tool

from inference import LLM, InferenceConfig
from task_manager import (
    TaskRecord,
    TaskStatus,
    TaskType,
    list_all_tasks,
)
from web_research import fetch_page, search_web

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(name)s] %(levelname)s %(message)s",
)
logger = logging.getLogger("caof-mcp")

app = Server("caof-mcp")
_executor = ThreadPoolExecutor(max_workers=4)


# ── Tool definitions ─────────────────────────────────────────

@app.list_tools()
async def list_tools() -> list[Tool]:
    return [
        Tool(
            name="start_task",
            description=(
                "Start a new task. Supports types: 'research' (web research with "
                "file output), 'social_content' (generate Twitter/LinkedIn posts "
                "from trending news), 'generic' (any description). Returns task ID "
                "and starts execution in background."
            ),
            inputSchema={
                "type": "object",
                "properties": {
                    "description": {
                        "type": "string",
                        "description": "What the task should accomplish.",
                    },
                    "task_type": {
                        "type": "string",
                        "enum": ["research", "social_content", "generic"],
                        "description": "Type of task. Default: generic.",
                    },
                    "platform": {
                        "type": "string",
                        "enum": ["twitter", "linkedin", "both"],
                        "description": "Target platform (social_content tasks only).",
                    },
                    "topic": {
                        "type": "string",
                        "description": "Topic or theme (social_content tasks). If empty, will research trending news.",
                    },
                    "web_urls": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "URLs to fetch for research context.",
                    },
                    "search_queries": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "Web search queries to run for research.",
                    },
                },
                "required": ["description"],
            },
        ),
        Tool(
            name="get_task",
            description="Get the status and result of a task by ID.",
            inputSchema={
                "type": "object",
                "properties": {
                    "task_id": {"type": "string", "description": "The task ID."},
                },
                "required": ["task_id"],
            },
        ),
        Tool(
            name="list_tasks",
            description="List all tasks with their status.",
            inputSchema={
                "type": "object",
                "properties": {
                    "status": {
                        "type": "string",
                        "enum": ["pending", "running", "completed", "failed", "all"],
                        "description": "Filter by status. Default: all.",
                    },
                },
            },
        ),
        Tool(
            name="get_task_file",
            description="Read a specific file from a task's output folder.",
            inputSchema={
                "type": "object",
                "properties": {
                    "task_id": {"type": "string"},
                    "filename": {
                        "type": "string",
                        "description": "Relative path within the task folder (e.g. 'result.md', 'research/page1.txt', 'post.md').",
                    },
                },
                "required": ["task_id", "filename"],
            },
        ),
    ]


# ── Tool handlers ────────────────────────────────────────────

@app.call_tool()
async def call_tool(name: str, arguments: dict) -> list[TextContent]:
    if name == "start_task":
        return await _handle_start_task(arguments)
    elif name == "get_task":
        return await _handle_get_task(arguments)
    elif name == "list_tasks":
        return await _handle_list_tasks(arguments)
    elif name == "get_task_file":
        return await _handle_get_task_file(arguments)
    return [TextContent(type="text", text=f"Unknown tool: {name}")]


async def _handle_start_task(args: dict) -> list[TextContent]:
    task_type = TaskType(args.get("task_type", "generic"))
    task = TaskRecord(
        task_type=task_type,
        description=args["description"],
        platform=args.get("platform"),
        topic=args.get("topic"),
        web_context=args.get("web_urls", []) + args.get("search_queries", []),
    )
    task.save()

    # Run execution in background so tool returns immediately
    asyncio.get_event_loop().run_in_executor(_executor, _execute_task, task.task_id)

    return [TextContent(
        type="text",
        text=json.dumps({
            "task_id": task.task_id,
            "status": "pending",
            "task_type": task.task_type,
            "folder": str(task.folder),
            "message": f"Task created. Use get_task(task_id='{task.task_id}') to check progress.",
        }, indent=2),
    )]


async def _handle_get_task(args: dict) -> list[TextContent]:
    try:
        task = TaskRecord.load(args["task_id"])
    except FileNotFoundError:
        return [TextContent(type="text", text=f"Task {args['task_id']} not found.")]

    info = {
        "task_id": task.task_id,
        "task_type": task.task_type,
        "status": task.status,
        "description": task.description,
        "created_at": task.created_at,
        "completed_at": task.completed_at,
        "files": task.list_files(),
    }

    # Include result content if completed
    if task.status == TaskStatus.COMPLETED:
        try:
            info["result"] = task.read_file("result.md")
        except FileNotFoundError:
            pass
        if task.task_type == TaskType.SOCIAL_CONTENT:
            try:
                info["post"] = task.read_file("post.md")
            except FileNotFoundError:
                pass

    if task.error:
        info["error"] = task.error

    return [TextContent(type="text", text=json.dumps(info, indent=2))]


async def _handle_list_tasks(args: dict) -> list[TextContent]:
    status_filter = args.get("status", "all")
    tasks = list_all_tasks()
    if status_filter != "all":
        tasks = [t for t in tasks if t.status == status_filter]

    items = [
        {
            "task_id": t.task_id,
            "type": t.task_type,
            "status": t.status,
            "description": t.description[:80],
            "created_at": t.created_at,
        }
        for t in tasks
    ]
    return [TextContent(type="text", text=json.dumps(items, indent=2))]


async def _handle_get_task_file(args: dict) -> list[TextContent]:
    try:
        task = TaskRecord.load(args["task_id"])
        content = task.read_file(args["filename"])
        return [TextContent(type="text", text=content)]
    except FileNotFoundError as e:
        return [TextContent(type="text", text=str(e))]


# ── Task execution (runs in thread pool) ─────────────────────

def _execute_task(task_id: str) -> None:
    """Execute a task synchronously (called from thread pool)."""
    task = TaskRecord.load(task_id)
    task.set_running()

    try:
        llm = LLM()

        # Phase 1: Gather research context
        research_context = _gather_research(task, llm)

        # Phase 2: Produce output based on task type
        if task.task_type == TaskType.RESEARCH:
            _execute_research(task, llm, research_context)
        elif task.task_type == TaskType.SOCIAL_CONTENT:
            _execute_social_content(task, llm, research_context)
        else:
            _execute_generic(task, llm, research_context)

        task.set_completed()
        llm.close()
        logger.info(f"Task {task_id} completed")

    except Exception as e:
        logger.error(f"Task {task_id} failed: {e}")
        task.set_failed(str(e))


def _gather_research(task: TaskRecord, llm: LLM) -> str:
    """Fetch web pages, run searches, return combined context."""
    parts: list[str] = []

    for item in task.web_context:
        if item.startswith("http://") or item.startswith("https://"):
            logger.info(f"Fetching URL: {item}")
            content = fetch_page(item)
            # Save raw research
            safe_name = item.split("//")[-1].replace("/", "_")[:60] + ".txt"
            task.write_research(content, safe_name)
            parts.append(f"--- Content from {item} ---\n{content[:5000]}\n")
        else:
            # Treat as search query
            logger.info(f"Searching: {item}")
            results = search_web(item, num_results=5)
            search_text = f"--- Search results for: {item} ---\n"
            for r in results:
                search_text += f"- {r['title']}: {r['url']}\n  {r['snippet']}\n"
            task.write_research(search_text, f"search_{item[:40].replace(' ', '_')}.txt")
            parts.append(search_text)

            # Fetch top 2 results for deeper context
            for r in results[:2]:
                if r["url"].startswith("http"):
                    content = fetch_page(r["url"])
                    parts.append(f"--- Page: {r['title']} ---\n{content[:3000]}\n")

    # For social content without explicit queries, search for trending news
    if task.task_type == TaskType.SOCIAL_CONTENT and not task.web_context:
        topic = task.topic or task.description
        queries = [
            f"{topic} latest news {datetime.now(timezone.utc).strftime('%Y-%m-%d')}",
            f"{topic} trending",
        ]
        for q in queries:
            results = search_web(q, num_results=5)
            search_text = f"--- Trending search: {q} ---\n"
            for r in results:
                search_text += f"- {r['title']}: {r['url']}\n  {r['snippet']}\n"
            task.write_research(search_text, f"trending_{q[:30].replace(' ', '_')}.txt")
            parts.append(search_text)

            for r in results[:2]:
                if r["url"].startswith("http"):
                    content = fetch_page(r["url"])
                    parts.append(f"--- Page: {r['title']} ---\n{content[:3000]}\n")

    return "\n\n".join(parts) if parts else "(no web research performed)"


def _execute_research(task: TaskRecord, llm: LLM, context: str) -> None:
    """Produce a research report."""
    result = llm.complete(
        prompt=(
            f"Based on the following research context, produce a comprehensive "
            f"research report for this request:\n\n"
            f"REQUEST: {task.description}\n\n"
            f"RESEARCH CONTEXT:\n{context[:12000]}\n\n"
            f"Write a well-structured report in Markdown with sections, key findings, "
            f"and source references."
        ),
        system="You are a thorough research analyst. Produce detailed, well-sourced reports.",
        max_tokens=4096,
    )
    task.write_result(result, "result.md")


def _execute_social_content(task: TaskRecord, llm: LLM, context: str) -> None:
    """Research trending topics and generate social media posts."""
    platform = task.platform or "both"
    topic = task.topic or task.description

    # Step 1: Analyze what's trending
    analysis = llm.complete(
        prompt=(
            f"Analyze the following research about '{topic}' and identify:\n"
            f"1. The most interesting/viral-worthy angle\n"
            f"2. Key facts or statistics\n"
            f"3. A unique perspective that would stand out\n\n"
            f"RESEARCH:\n{context[:10000]}"
        ),
        system="You are a social media strategist who identifies viral content angles.",
        max_tokens=1500,
    )
    task.write_research(analysis, "analysis.txt")

    # Step 2: Generate posts
    platform_instructions = ""
    if platform in ("twitter", "both"):
        platform_instructions += (
            "\n\n## Twitter/X Post\n"
            "Write 1-3 tweet variations (max 280 chars each). "
            "Use hooks, hot takes, or thread starters. Include relevant hashtags.\n"
        )
    if platform in ("linkedin", "both"):
        platform_instructions += (
            "\n\n## LinkedIn Post\n"
            "Write a LinkedIn post (500-1500 chars). Professional but engaging tone. "
            "Start with a hook line. Include a call-to-action. No hashtag spam.\n"
        )

    posts = llm.complete(
        prompt=(
            f"Create social media content about '{topic}' based on this analysis:\n\n"
            f"{analysis}\n\n"
            f"Generate the following:{platform_instructions}\n\n"
            f"Make the content unique, opinionated, and share-worthy. "
            f"Reference specific facts from the research. Avoid generic fluff."
        ),
        system=(
            "You are an expert social media content creator. You write posts that "
            "get high engagement because they lead with insight, not hype."
        ),
        max_tokens=2000,
    )
    task.write_result(posts, "post.md")

    # Also save a summary
    summary = f"# Social Content Task\n\n**Topic:** {topic}\n**Platform:** {platform}\n\n{posts}"
    task.write_result(summary, "result.md")


def _execute_generic(task: TaskRecord, llm: LLM, context: str) -> None:
    """Handle a generic task."""
    extra = ""
    if context and context != "(no web research performed)":
        extra = f"\n\nADDITIONAL CONTEXT:\n{context[:10000]}"

    result = llm.complete(
        prompt=f"Complete the following task:\n\n{task.description}{extra}",
        system="You are a capable assistant. Produce thorough, actionable output.",
        max_tokens=4096,
    )
    task.write_result(result, "result.md")


# ── Entry point ──────────────────────────────────────────────

async def run_stdio():
    async with stdio_server() as (read_stream, write_stream):
        await app.run(read_stream, write_stream, app.create_initialization_options())


def main():
    asyncio.run(run_stdio())


if __name__ == "__main__":
    main()

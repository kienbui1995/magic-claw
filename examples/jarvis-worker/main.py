"""
Jarvis Worker — wraps my-jarvis personal AI assistant as a MagiC worker.

my-jarvis handles: chat, tasks, calendar, memory, web search, notes, and 25+ tools.
This wrapper registers jarvis as a worker in a MagiC fleet.

Usage:
    pip install magic-ai-sdk httpx
    MAGIC_URL=http://localhost:18080 \
    MAGIC_WORKER_TOKEN=mct_xxx \
    JARVIS_URL=https://jarvis.pmai.space \
    JARVIS_API_KEY=your-64-char-key \
    python main.py
"""

import logging
import os

import httpx
from magic_ai_sdk import Worker, capability

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("jarvis-worker")

JARVIS_URL = os.environ["JARVIS_URL"].rstrip("/")
JARVIS_API_KEY = os.environ["JARVIS_API_KEY"]
MAGIC_URL = os.environ.get("MAGIC_URL", "http://localhost:18080")
MAGIC_WORKER_TOKEN = os.environ.get("MAGIC_WORKER_TOKEN", "")

_headers = {"X-API-Key": JARVIS_API_KEY}


def _call(method: str, path: str, **kwargs):
    """Synchronous helper — my-jarvis API calls."""
    url = f"{JARVIS_URL}/api/public/v1{path}"
    with httpx.Client(timeout=30) as client:
        resp = client.request(method, url, headers=_headers, **kwargs)
        resp.raise_for_status()
        return resp.json()


class JarvisWorker(Worker):
    """MagiC worker backed by my-jarvis personal AI assistant."""

    # ── Conversational AI ──────────────────────────────────────────────────

    @capability(
        name="chat",
        description="Vietnamese personal AI assistant. Handles general questions, "
                    "creates tasks, searches memory, and coordinates other capabilities. "
                    "Input: {message: str, conversation_id?: str}",
    )
    def chat(self, task: dict) -> dict:
        inp = task.get("input", {})
        result = _call("POST", "/chat", json={
            "message": inp.get("message", ""),
            "conversation_id": inp.get("conversation_id"),
        })
        log.info("chat → model=%s", result.get("model"))
        return {"response": result["response"], "model": result.get("model")}

    # ── Task Management ────────────────────────────────────────────────────

    @capability(
        name="task_create",
        description="Create a new personal task. "
                    "Input: {title: str, due_date?: 'YYYY-MM-DD', priority?: 'low|medium|high|urgent'}",
    )
    def task_create(self, task: dict) -> dict:
        return _call("POST", "/tools/task_create/invoke",
                     json={"args": task.get("input", {})})

    @capability(
        name="task_list",
        description="List personal tasks. Input: {status?: 'pending|done|all'}",
    )
    def task_list(self, task: dict) -> dict:
        return _call("POST", "/tools/task_list/invoke",
                     json={"args": task.get("input", {})})

    @capability(
        name="task_update",
        description="Update task status or title. "
                    "Input: {task_id: str, status?: str, title?: str}",
    )
    def task_update(self, task: dict) -> dict:
        return _call("POST", "/tools/task_update/invoke",
                     json={"args": task.get("input", {})})

    # ── Calendar ───────────────────────────────────────────────────────────

    @capability(
        name="calendar_create",
        description="Create a calendar event. "
                    "Input: {title: str, date: 'YYYY-MM-DD', time?: 'HH:MM', description?: str}",
    )
    def calendar_create(self, task: dict) -> dict:
        return _call("POST", "/tools/calendar_create/invoke",
                     json={"args": task.get("input", {})})

    @capability(
        name="calendar_list",
        description="List upcoming calendar events. Input: {days?: int}",
    )
    def calendar_list(self, task: dict) -> dict:
        return _call("POST", "/tools/calendar_list/invoke",
                     json={"args": task.get("input", {})})

    # ── Memory ─────────────────────────────────────────────────────────────

    @capability(
        name="memory_search",
        description="Search personal memory (semantic + keyword). "
                    "Input: {query: str, limit?: int}",
    )
    def memory_search(self, task: dict) -> dict:
        return _call("POST", "/tools/memory_search/invoke",
                     json={"args": task.get("input", {})})

    @capability(
        name="memory_save",
        description="Save something to personal memory. Input: {content: str}",
    )
    def memory_save(self, task: dict) -> dict:
        return _call("POST", "/tools/memory_save/invoke",
                     json={"args": task.get("input", {})})

    # ── Web ────────────────────────────────────────────────────────────────

    @capability(
        name="web_search",
        description="Search the web via DuckDuckGo. Input: {query: str}",
    )
    def web_search(self, task: dict) -> dict:
        return _call("POST", "/tools/web_search/invoke",
                     json={"args": task.get("input", {})})

    @capability(
        name="summarize_url",
        description="Summarize the content of a URL. Input: {url: str}",
    )
    def summarize_url(self, task: dict) -> dict:
        return _call("POST", "/tools/summarize_url/invoke",
                     json={"args": task.get("input", {})})

    # ── Notes ──────────────────────────────────────────────────────────────

    @capability(
        name="note_save",
        description="Save a note. Input: {content: str, title?: str}",
    )
    def note_save(self, task: dict) -> dict:
        return _call("POST", "/tools/note_save/invoke",
                     json={"args": task.get("input", {})})

    @capability(
        name="note_search",
        description="Search personal notes. Input: {query: str}",
    )
    def note_search(self, task: dict) -> dict:
        return _call("POST", "/tools/note_search/invoke",
                     json={"args": task.get("input", {})})

    # ── Vietnam-specific ───────────────────────────────────────────────────

    @capability(
        name="weather_vn",
        description="Vietnam weather forecast. Input: {city: str}",
    )
    def weather_vn(self, task: dict) -> dict:
        return _call("POST", "/tools/weather_vn/invoke",
                     json={"args": task.get("input", {})})

    @capability(
        name="news_vn",
        description="Vietnam news headlines. Input: {topic?: str}",
    )
    def news_vn(self, task: dict) -> dict:
        return _call("POST", "/tools/news_vn/invoke",
                     json={"args": task.get("input", {})})


if __name__ == "__main__":
    worker = JarvisWorker(
        name="JarvisWorker",
        endpoint=f"http://jarvis-worker:9001",  # this container's address
        worker_token=MAGIC_WORKER_TOKEN,
    )
    log.info("Registering JarvisWorker with MagiC at %s", MAGIC_URL)
    worker.serve(magic_url=MAGIC_URL, port=9001)

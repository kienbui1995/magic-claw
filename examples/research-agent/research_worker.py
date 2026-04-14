"""
Research Agent Worker — demonstrates MagiC AI features.

This worker handles 3 capabilities:
  - research.search: simulates web search
  - research.summarize: uses LLM to summarize findings
  - research.analyze: uses LLM to analyze and produce report

Requires: pip install magic-ai-sdk requests
"""

import json
import os
import requests
from http.server import HTTPServer, BaseHTTPRequestHandler

MAGIC_URL = os.getenv("MAGIC_URL", "http://localhost:8080")
WORKER_PORT = int(os.getenv("WORKER_PORT", "9100"))
SESSION_ID = "research-session-1"


def search(topic: str) -> dict:
    """Simulate web search — in production, use SerpAPI/Tavily/etc."""
    return {
        "results": [
            f"Research finding 1 about {topic}: Recent studies show significant progress...",
            f"Research finding 2 about {topic}: Industry experts suggest that...",
            f"Research finding 3 about {topic}: A 2025 meta-analysis found that...",
        ]
    }


def llm_chat(messages: list, strategy: str = "best") -> str:
    """Call MagiC LLM gateway."""
    resp = requests.post(f"{MAGIC_URL}/api/v1/llm/chat", json={
        "messages": messages,
        "strategy": strategy,
    })
    if resp.status_code != 200:
        return f"LLM error: {resp.text}"
    return resp.json().get("content", "")


def render_prompt(name: str, vars: dict) -> str:
    """Render a prompt template from the registry."""
    resp = requests.post(f"{MAGIC_URL}/api/v1/prompts/render", json={
        "name": name,
        "vars": vars,
    })
    if resp.status_code != 200:
        return ""
    return resp.json().get("rendered", "")


def save_memory(role: str, content: str):
    """Save conversation turn to agent memory."""
    requests.post(f"{MAGIC_URL}/api/v1/memory/turns", json={
        "session_id": SESSION_ID,
        "agent_id": "research-agent",
        "role": role,
        "content": content,
    })


def handle_task(task_type: str, input_data: dict) -> dict:
    topic = input_data.get("topic", "AI agents")

    if task_type == "research.search":
        results = search(topic)
        save_memory("assistant", f"Searched for: {topic}")
        return results

    elif task_type == "research.summarize":
        results = input_data.get("results", [])
        if not results and "_deps" in input_data:
            dep_data = list(input_data["_deps"].values())[0]
            if isinstance(dep_data, dict):
                results = dep_data.get("results", [])

        prompt_text = render_prompt("research.summarize", {
            "topic": topic,
            "results": "\n".join(results) if results else "No results found.",
        })
        if not prompt_text:
            prompt_text = f"Summarize research about {topic}: {results}"

        summary = llm_chat([
            {"role": "system", "content": "You are a research assistant."},
            {"role": "user", "content": prompt_text},
        ], strategy="cheapest")

        save_memory("assistant", f"Summary: {summary[:200]}...")
        return {"summary": summary}

    elif task_type == "research.analyze":
        summary = input_data.get("summary", "")
        if not summary and "_deps" in input_data:
            dep_data = list(input_data["_deps"].values())[0]
            if isinstance(dep_data, dict):
                summary = dep_data.get("summary", "")

        prompt_text = render_prompt("research.analyze", {
            "topic": topic,
            "summary": summary,
        })
        if not prompt_text:
            prompt_text = f"Analyze this research about {topic}: {summary}"

        analysis = llm_chat([
            {"role": "system", "content": "You are a senior research analyst."},
            {"role": "user", "content": prompt_text},
        ], strategy="best")

        save_memory("assistant", f"Analysis complete for: {topic}")
        return {"analysis": analysis, "topic": topic}

    return {"error": f"Unknown task type: {task_type}"}


class WorkerHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        payload = body.get("payload", {})
        task_type = payload.get("task_type", "")
        task_id = payload.get("task_id", "")
        input_data = payload.get("input", {})

        try:
            result = handle_task(task_type, input_data)
            response = {
                "type": "task.complete",
                "payload": {"task_id": task_id, "output": result, "cost": 0},
            }
        except Exception as e:
            response = {
                "type": "task.fail",
                "payload": {"task_id": task_id, "error": {"code": "error", "message": str(e)}},
            }

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(response).encode())

    def log_message(self, format, *args):
        print(f"[research-worker] {args[0]}")


def register():
    resp = requests.post(f"{MAGIC_URL}/api/v1/workers/register", json={
        "name": "ResearchAgent",
        "capabilities": [
            {"name": "research.search", "description": "Search the web for a topic"},
            {"name": "research.summarize", "description": "Summarize search results using LLM"},
            {"name": "research.analyze", "description": "Analyze and produce research report"},
        ],
        "endpoint": {"type": "http", "url": f"http://localhost:{WORKER_PORT}"},
    })
    print(f"Registered: {resp.json().get('id', 'error')}")


if __name__ == "__main__":
    register()
    print(f"Research worker listening on :{WORKER_PORT}")
    HTTPServer(("0.0.0.0", WORKER_PORT), WorkerHandler).serve_forever()

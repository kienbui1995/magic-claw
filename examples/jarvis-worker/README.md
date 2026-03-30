# Jarvis Worker

Wraps [my-jarvis](https://jarvis.pmai.space) personal AI assistant as a MagiC worker.

## Capabilities

| Capability | Description |
|-----------|-------------|
| `chat` | General Vietnamese AI assistant |
| `task_create/list/update` | Personal task management |
| `calendar_create/list` | Calendar events |
| `memory_search/save` | Semantic memory |
| `web_search` | DuckDuckGo search |
| `summarize_url` | Webpage summarization |
| `note_save/search` | Notes |
| `weather_vn` | Vietnam weather |
| `news_vn` | Vietnam news |

## Quick Start

**1. Get a Jarvis API key**

```
jarvis.pmai.space → Settings → API Keys → Generate
```

**2. Set up MagiC worker token**

```bash
curl -X POST http://localhost:18080/api/v1/orgs/org_default/tokens \
  -H "Authorization: Bearer $MAGIC_API_KEY" \
  -d '{"name": "jarvis-worker"}'
```

**3. Run with Docker Compose**

```bash
cp .env.example .env
# fill in values
docker compose up
```

**4. Submit a task**

```bash
curl -X POST http://localhost:18080/api/v1/tasks \
  -H "Authorization: Bearer $MAGIC_API_KEY" \
  -d '{
    "type": "chat",
    "input": {"message": "Tôi có task gì hôm nay?"},
    "routing": {"required_capabilities": ["chat"]}
  }'
```

## Run without Docker

```bash
pip install magic-ai-sdk httpx

export MAGIC_URL=http://localhost:18080
export MAGIC_WORKER_TOKEN=mct_xxx
export JARVIS_URL=https://jarvis.pmai.space
export JARVIS_API_KEY=your-key

python main.py
```

## Architecture

```
MagiC Server
    │  task.assign
    ▼
JarvisWorker (this)
    │  POST /api/public/v1/chat
    │  POST /api/public/v1/tools/{name}/invoke
    ▼
my-jarvis backend
    │
    ▼
LangGraph + LiteLLM + PostgreSQL + Redis
```

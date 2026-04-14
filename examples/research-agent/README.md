# Research Agent — MagiC AI Framework Example

A multi-step research agent that demonstrates all MagiC AI features working together:
- **LLM Gateway** — routes to cheapest model for summarization, best model for analysis
- **Prompt Registry** — versioned prompts for each research step
- **Agent Memory** — remembers conversation context + stores research findings
- **Worker Orchestration** — DAG workflow: search → summarize → analyze → report

## Architecture

```
User Question
     │
     ▼
┌─────────────┐     ┌──────────────┐
│ Search Step │────▶│ Summarize    │
│ (web search)│     │ (cheapest LLM)│
└─────────────┘     └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │ Analyze      │
                    │ (best LLM)   │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │ Report       │
                    │ (best LLM)   │
                    └──────────────┘
```

## Run

```bash
# 1. Start MagiC server with LLM provider
export OPENAI_API_KEY=sk-...
magic serve &

# 2. Register prompts
curl -X POST http://localhost:8080/api/v1/prompts \
  -H "Content-Type: application/json" \
  -d '{"name":"research.summarize","content":"Summarize the following search results about {{topic}}:\n\n{{results}}\n\nProvide a concise 3-paragraph summary."}'

curl -X POST http://localhost:8080/api/v1/prompts \
  -H "Content-Type: application/json" \
  -d '{"name":"research.analyze","content":"Based on this research summary about {{topic}}:\n\n{{summary}}\n\nProvide:\n1. Key findings\n2. Gaps in the research\n3. Recommended next steps"}'

# 3. Start the research worker
python research_worker.py &

# 4. Submit a research workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @workflow.json
```

## Files

- `research_worker.py` — Python worker that handles search, summarize, analyze steps
- `workflow.json` — Example DAG workflow definition
- `run.sh` — One-command demo script

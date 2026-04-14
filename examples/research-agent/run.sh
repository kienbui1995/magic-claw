#!/bin/bash
# Research Agent Demo — run everything in one command
set -e

MAGIC_URL="${MAGIC_URL:-http://localhost:8080}"

echo "=== MagiC Research Agent Demo ==="
echo ""

# 1. Register prompt templates
echo "📝 Registering prompt templates..."
curl -s -X POST "$MAGIC_URL/api/v1/prompts" \
  -H "Content-Type: application/json" \
  -d '{"name":"research.summarize","content":"Summarize the following search results about {{topic}}:\n\n{{results}}\n\nProvide a concise 3-paragraph summary."}' > /dev/null

curl -s -X POST "$MAGIC_URL/api/v1/prompts" \
  -H "Content-Type: application/json" \
  -d '{"name":"research.analyze","content":"Based on this research summary about {{topic}}:\n\n{{summary}}\n\nProvide:\n1. Key findings\n2. Gaps in the research\n3. Recommended next steps"}' > /dev/null

echo "✅ Prompts registered"

# 2. Start worker in background
echo "🤖 Starting research worker..."
python research_worker.py &
WORKER_PID=$!
sleep 1

# 3. Submit workflow
echo "🚀 Submitting research workflow..."
RESULT=$(curl -s -X POST "$MAGIC_URL/api/v1/workflows" \
  -H "Content-Type: application/json" \
  -d @workflow.json)

WF_ID=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "unknown")
echo "📋 Workflow ID: $WF_ID"

# 4. Poll for completion
echo "⏳ Waiting for completion..."
for i in $(seq 1 30); do
  STATUS=$(curl -s "$MAGIC_URL/api/v1/workflows/$WF_ID" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])" 2>/dev/null || echo "pending")
  if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
    echo "✅ Workflow $STATUS"
    curl -s "$MAGIC_URL/api/v1/workflows/$WF_ID" | python3 -m json.tool
    break
  fi
  sleep 2
done

# 5. Show memory
echo ""
echo "🧠 Agent memory:"
curl -s "$MAGIC_URL/api/v1/memory/turns?session_id=research-session-1" | python3 -m json.tool

# Cleanup
kill $WORKER_PID 2>/dev/null
echo ""
echo "Done!"

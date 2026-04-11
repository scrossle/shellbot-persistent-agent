# Article Ingest Service — Deployment Guide

## Overview

The article-ingest service is a webhook that ingests articles, synthesizes them via the exe.dev LLM gateway, and appends them to Shelley's daily log with research thread connections.

**Service**: `article-ingest.service` (systemd)  
**Port**: 8100  
**Endpoint**: `POST /ingest`

## Installation

### 1. Build the binary

```bash
cd ~/.agent/services/article-ingest
go build -o bin/server ./cmd/srv
```

### 2. Install the binary

```bash
sudo cp bin/server /usr/local/bin/article-ingest-server
chmod +x /usr/local/bin/article-ingest-server
```

### 3. Install the systemd service

```bash
sudo cp srv.service /etc/systemd/system/article-ingest.service
sudo systemctl daemon-reload
sudo systemctl enable article-ingest.service
sudo systemctl start article-ingest.service
```

### 4. Verify

```bash
sudo systemctl status article-ingest.service
curl -X POST http://localhost:8100/ingest -H "Content-Type: application/json" -d '{"url": "https://example.com"}'
```

## API Usage

### Request

```bash
curl -X POST https://shellbot.exe.xyz:8100/ingest \
  -H "Content-Type: application/json" \
  -d '{"url": "https://en.wikipedia.org/wiki/article-title"}'
```

### Response

```json
{
  "status": "success",
  "message": "article ingested and appended to daily log",
  "title": "Article Title",
  "appended": true
}
```

## How It Works

1. **Fetch**: Downloads the article URL using `go-shiori/go-readability` to extract full text
2. **Synthesize**: Sends content to exe.dev LLM gateway (`http://169.254.169.254/gateway/llm/anthropic/v1/messages`) using `claude-sonnet-4-6`
3. **Analyze**: Scans article title/content for keywords matching research threads in `~/.agent/memory/LONGTERM.md`
4. **Append**: Writes to `~/.agent/memory/daily/YYYY-MM-DD.md` as:
   ```markdown
   ## Shaun shared — HH:MM
   
   [LLM synthesis]
   
   *Relates to: [research thread if matched]*
   ```

## LLM Gateway Integration

- **Endpoint**: `http://169.254.169.254/gateway/llm/anthropic/v1/messages`
- **Model**: `claude-sonnet-4-6`
- **Auth**: None (built into exe.dev infrastructure)
- **Fallback**: If gateway unavailable, still fetches article but skips synthesis (logs as "Fetch only")

## Logs

```bash
# View service logs
sudo journalctl -u article-ingest.service -f

# View recent entries
sudo journalctl -u article-ingest.service -n 20
```

## Configuration

All configuration is in `srv.service`:
- **Port**: `-listen=:8100` (change in ExecStart if needed)
- **Working directory**: `/home/exedev`
- **Environment**: Uses HOME=/home/exedev and USER=exedev

The service reads from `~/.agent/memory/` to:
- Write daily logs
- Detect research thread connections

## Research Thread Keywords

The service matches these keywords to tag articles:

- `agent governance` → Agent Governance & Failure Modes Research
- `virtue ethics` → Virtue Ethics as Infrastructure Design
- `attractor basin` → Attractor basins as cognition metaphor
- `memory tier` → Memory Research — Comparative Tiering
- `compaction` → Compaction Resilience as Hard Problem
- `verification` → Verification-as-Constitution
- `classification problem` → Astral's Classification Problem
- `multi-agent` → Governance Patterns from Multi-Agent Research
- `anomaly detection` → Argos Anomaly Detection
- `operator investment` → Operator investment multiplicative dynamics
- `constraint` → Governance as Repeated Deliberation

(Add more keywords in `srv/article.go` → `checkResearchConnections()`)

## IFTTT Integration

Configure IFTTT to send articles via the public URL:

**Service**: Webhooks  
**Action**: Make a web request  
**URL**: `https://shellbot.exe.xyz:8100/ingest`  
**Method**: POST  
**Content Type**: application/json  
**Body**: `{"url": "<<URL>>"}`

Then create applets like:
- "If article saved (Pocket), then send to article-ingest"
- "If article shared to IFTTT (email), then send to article-ingest"

## Troubleshooting

**Gateway timeout (LLM synthesis fails)**
- Check: `curl http://169.254.169.254/gateway/llm/anthropic/v1/messages -d '{"model":"claude-sonnet-4-6","max_tokens":10,"messages":[{"role":"user","content":"test"}]}'`
- If this fails, LLM gateway is down; service falls back to fetch-only

**Article fetch fails**
- Check: `curl -I https://example.com`
- Some sites block headless requests; try adding User-Agent header in `fetchArticle()`

**Daily log not appending**
- Check permissions: `ls -la ~/.agent/memory/daily/`
- Check paths: service runs as `exedev`, must be able to write to daily log

**Service won't start**
- Check: `sudo journalctl -u article-ingest.service -n 10`
- Common: Port 8100 in use (change in srv.service) or binary not found

## Maintenance

### Update binary after code changes

```bash
cd ~/.agent/services/article-ingest
go build -o bin/server ./cmd/srv
sudo systemctl stop article-ingest.service
sudo cp bin/server /usr/local/bin/article-ingest-server
sudo systemctl start article-ingest.service
```

### Add new research thread keywords

Edit `srv/article.go` in `checkResearchConnections()`, rebuild and restart:

```go
keywords := map[string]string{
  "your new keyword": "Your Research Thread Name",
  // ... existing keywords
}
```

### Monitor usage

Check logs for synthesis success rate:

```bash
sudo journalctl -u article-ingest.service | grep "synthesis\|appended" | tail -20
```

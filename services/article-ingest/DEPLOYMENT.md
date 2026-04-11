# Article Ingest Service — Deployment Guide

## Overview

The article-ingest service is a webhook that ingests articles, synthesizes them via the exe.dev LLM gateway, and appends them to Shelley's daily log with research thread connections.

**Service**: `article-ingest.service` (systemd)  
**Port**: 8100  
**Endpoint**: `POST /ingest`  
**Authentication**: Required (API token)

## Security Features

### 1. API Token Authentication
- **Required parameter**: `token` query string
- **Environment variable**: `ARTICLE_INGEST_TOKEN`
- **Example**: `https://shellbot.exe.xyz/ingest?token=<YOUR_TOKEN>`
- **Token format**: 64-character hex string (256-bit)
- **Generated via**: `openssl rand -hex 32`

### 2. Rate Limiting
- **Limit**: 1 request per second
- **Burst**: 5 requests allowed (then rate-limited)
- **Response on limit exceeded**: HTTP 429 (Too Many Requests)
- **Per-IP**: Each source IP has independent limit

### 3. Input Validation & SSRF Protection
- **Allowed schemes**: HTTP, HTTPS only
- **Rejected**: localhost, 127.0.0.1, private IPs (10.x, 172.16-31.x, 192.168.x, 169.254.x)
- **Rejected schemes**: file://, data://, ftp://, etc.
- **Response body limit**: 10MB (prevents resource exhaustion)
- **Validation**: Happens before fetching (fail fast)

## Installation

### 1. Build the binary

```bash
cd ~/persistent-agent/services/article-ingest
go build -o bin/server ./cmd/srv
```

### 2. Install the binary

```bash
sudo cp bin/server /usr/local/bin/article-ingest-server
chmod +x /usr/local/bin/article-ingest-server
```

### 3. Install the systemd service

```bash
sudo cp ~/persistent-agent/systemd/article-ingest.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable article-ingest.service
sudo systemctl start article-ingest.service
```

### 4. Verify

```bash
sudo systemctl status article-ingest.service
TOKEN="<YOUR_TOKEN>"
curl -X POST "http://localhost:8100/ingest?token=${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

## Updating the API Token

To rotate the API token:

```bash
# Generate new token
NEW_TOKEN=$(openssl rand -hex 32)
echo "New token: $NEW_TOKEN"

# Update systemd service
sudo systemctl edit article-ingest.service
# Change: Environment=ARTICLE_INGEST_TOKEN=<NEW_TOKEN>

# Restart service
sudo systemctl restart article-ingest.service

# Update IFTTT webhook URL with new token
```

## API Usage

### Request

```bash
TOKEN="your-api-token"
curl -X POST "https://shellbot.exe.xyz/ingest?token=${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://en.wikipedia.org/wiki/article-title"}'
```

### Response (Success)

```json
{
  "status": "success",
  "message": "article ingested and appended to daily log",
  "title": "Article Title",
  "appended": true
}
```

### Response (Errors)

**Unauthorized (missing/invalid token)**:
```json
{"status": "error", "message": "unauthorized", "appended": false}
```

**Invalid URL**:
```json
{"status": "error", "message": "invalid URL: localhost not allowed", "appended": false}
```

**Rate limited**:
```json
{"status": "error", "message": "rate limit exceeded", "appended": false}
```

**Fetch failed**:
```json
{"status": "error", "message": "failed to fetch article: HTTP 404", "appended": false}
```

## IFTTT Integration

### Create a new applet:

1. Go to https://ifttt.com/create
2. **If This**: Choose trigger (Pocket, Save, Email, etc.)
3. **Then That**: Choose Webhooks → Make a web request
4. **URL**: `https://shellbot.exe.xyz/ingest?token=YOUR_TOKEN`
5. **Method**: POST
6. **Content Type**: application/json
7. **Body**: 
   ```json
   {"url": "<<URL>>"}
   ```

Replace `YOUR_TOKEN` with the actual token from your systemd service.

### Examples:

- **"If article saved (Pocket), then send to article-ingest"**
- **"If email received with tag #share, then send to article-ingest"**
- **"If new item in RSS feed, then send to article-ingest"**

## How It Works

1. **Fetch**: Downloads the article URL using go-shiori/go-readability to extract full text
2. **Synthesize**: Sends content to exe.dev LLM gateway (claude-sonnet-4-6)
3. **Analyze**: Scans article for keywords matching research threads in ~/.agent/memory/LONGTERM.md
4. **Append**: Writes to ~/.agent/memory/daily/YYYY-MM-DD.md as:
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

# View with timestamps
sudo journalctl -u article-ingest.service --since "1 hour ago" -o short-precise
```

## Configuration

All configuration is in systemd service:
- **Port**: `-listen=:8100` (change in ExecStart if needed)
- **Working directory**: `/home/exedev`
- **Environment**: HOME=/home/exedev, USER=exedev, ARTICLE_INGEST_TOKEN=<token>

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

## Troubleshooting

**Gateway timeout (LLM synthesis fails)**
- Check: `curl http://169.254.169.254/gateway/llm/anthropic/v1/messages -d '{"model":"claude-sonnet-4-6","max_tokens":10,"messages":[{"role":"user","content":"test"}]}'`
- If this fails, LLM gateway is down; service falls back to fetch-only

**Article fetch fails**
- Check: `curl -I https://example.com`
- Some sites block headless requests; add User-Agent header in `fetchArticle()`

**Daily log not appending**
- Check permissions: `ls -la ~/.agent/memory/daily/`
- Check paths: service runs as `exedev`, must be able to write to daily log

**Service won't start**
- Check: `sudo journalctl -u article-ingest.service -n 10`
- Common: Port 8100 in use (change in systemd service) or binary not found
- Verify: `which article-ingest-server` and check permissions

**Rate limit too strict**
- Current: 1 req/sec, burst of 5
- To increase: Edit `srv/server.go` line `limiter: rate.NewLimiter(1, 5)`
- Change to: `rate.NewLimiter(5, 10)` for 5 req/sec, burst of 10

## Maintenance

### Update binary after code changes

```bash
cd ~/persistent-agent/services/article-ingest
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

## Security Checklist

- [ ] API token set and not exposed in logs
- [ ] Rate limiting configured for expected load
- [ ] IFTTT webhook URL includes token parameter
- [ ] Test with malicious URLs (localhost, private IPs)
- [ ] Verify SSRF protection is working
- [ ] Monitor logs for unauthorized access attempts
- [ ] Rotate API token regularly (monthly recommended)

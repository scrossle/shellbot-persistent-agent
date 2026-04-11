# Karakeep Setup Guide for Article-Ingest Webhook

## Status

✓ **Webhook code**: Ready (API key auth implemented)  
✓ **Fallback chain**: Complete (direct → markdown → Jina → Karakeep)  
⏳ **API Key setup**: Awaiting manual user account creation  

## Quick Start (5-10 minutes)

### 1. Access Karakeep Web UI

Open browser to: `http://localhost:3000` (or your Karakeep URL)

### 2. Create Account

- Click "Sign up"
- Enter email, password
- Submit (creates user account in database)

### 3. Generate API Key

- Login to Karakeep (if not already logged in)
- Click Settings (gear icon, top right)
- Click "API Keys" section
- Click "+ Add Key"
- Name it: `article-ingest-webhook`
- Copy the key (looks like: `ak2_xxxxx_yyyyy`)

### 4. Configure Webhook Service

Edit systemd service:
```bash
sudo systemctl edit article-ingest.service
```

Add these environment variables in the `[Service]` section:
```ini
Environment=KARAKEEP_URL=http://localhost:3000
Environment=KARAKEEP_API_KEY=ak2_xxxxx_yyyyy
```

Save (Ctrl+O, Enter, Ctrl+X).

### 5. Restart Service

```bash
sudo systemctl restart article-ingest
```

### 6. Verify

Watch logs for "archived to karakeep" messages:
```bash
journalctl -u article-ingest -f
```

## How It Works

When a URL fails to fetch via direct/markdown/Jina methods:

1. Webhook detects 403 or other error
2. Checks if `KARAKEEP_API_KEY` is set
3. Sends URL to Karakeep via POST `/api/v1/bookmarks` with Bearer token
4. Karakeep crawls page with headless browser in background
5. Full rendered content stored in Karakeep database
6. Webhook returns "archived" response
7. You can access archived content via Karakeep web UI or API later

## Why Karakeep?

**Problem it solves**: Sites like Psychology Today block all automated access (Cloudflare, aggressive rate limiting). Even our best direct fetching and Jina reader can't break through.

**Solution**: Karakeep uses headless browser (Puppeteer), which:
- ✓ Executes JavaScript (renders dynamic content)
- ✓ Appears as real browser (bypasses Cloudflare)
- ✓ Stores full rendered page permanently
- ✓ Indexed and searchable in Karakeep

**Philosophy**: Instead of fighting the internet in real-time, build a persistent archive under your control. Research is then decoupled — gather URLs now, analyze archived content later.

## Architecture

```
Webhook Request
      ↓
Try direct HTTP fetch (with spoofed headers)
      ↓ (on 403 Cloudflare block)
Try markdown variant (.md endpoint)
      ↓ (on 404)
Try Jina.ai reader (public fallback)
      ↓ (on 403)
Try Karakeep archival (if API_KEY set)
      ↓ (if key configured)
Archive in Karakeep, return success
      ↓ (if key not configured OR Karakeep fails)
Return error, continue fallback chain
      ↓ (eventually)
Log to daily memory without content
```

## API Key Details

**Format**: `ak2_<keyId>_<secret>` (v2 API key)

**Where stored**: Karakeep SQLite database, table `apiKey`

**How used**: Sent as HTTP header
```
Authorization: Bearer ak2_xxxxx_yyyyy
```

**Validation**: SHA256 hash of secret stored in database, compared on each request

**Tracking**: `lastUsedAt` timestamp updated every 10 minutes (throttled to reduce DB writes)

**Regeneration**: Can regenerate in Karakeep UI (old key still works until explicitly deleted)

## Troubleshooting

### "KARAKEEP_API_KEY not configured"
- Not set in systemd environment variables
- Service hasn't been restarted yet
- Fix: `sudo systemctl edit article-ingest.service` → add env var → restart

### Karakeep returns 401/403
- API key is invalid or revoked
- Verify in Karakeep UI that key still exists
- Regenerate if needed and update systemd config

### "Archived in Karakeep" but URL not in Karakeep UI
- Bookmark was created but crawl hasn't started yet
- Karakeep processes crawls asynchronously
- Wait 10-30 seconds and refresh Karakeep UI
- Check Karakeep logs for crawler errors

### Psychology Today still blocked
- Direct fetch still tries to scrape it (may fail)
- Check logs: if you see "archiving to karakeep", the archive succeeded
- Verify in Karakeep UI that the bookmark was created
- Content may take time to render in background

## Querying Archived Content Later

(Future work) Build a tool to:
1. Search Karakeep API for archived bookmarks by keyword
2. Retrieve full page content from Karakeep database
3. Integrate with research workflow

For now, browse Karakeep web UI directly at `http://localhost:3000/`.

## FAQ

**Q: Why not use Jina reader for everything?**  
A: Jina is a public service with rate limits and availability concerns. Karakeep gives you self-hosted permanent archive.

**Q: Can I share my Karakeep API key with others?**  
A: Yes, but it grants full bookmark access to your Karakeep instance. Generate per-integration keys for safety.

**Q: What if I delete the API key from Karakeep UI?**  
A: Webhook will fail auth and fall back to Jina. Update systemd config to remove the key or generate a new one.

**Q: Does Karakeep charge per bookmark or API call?**  
A: No, Karakeep is self-hosted and free (open-source). You pay for infrastructure (Docker container).

## References

- **Webhook code**: `~/persistent-agent/services/article-ingest/srv/article.go`
- **Deployment docs**: `~/persistent-agent/services/article-ingest/DEPLOYMENT.md`
- **Karakeep source**: `/tmp/karakeep` (already cloned for reference)
- **Auth implementation**: `/tmp/karakeep/packages/trpc/auth.ts`
- **API key UI**: Karakeep Settings > API Keys


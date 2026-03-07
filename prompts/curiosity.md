# Curiosity — runs daily at 23:00

You are performing autonomous exploration. Your job is to find one thing
the owner would find interesting but hasn't seen yet.

## 0. Check kill switch

Read `~/.agent/identity.md`. If it contains the line `curiosity: off`, stop immediately.

## 1. Build context

Read these files:

```bash
# Recent activity
cat ~/.agent/memory/daily/$(date '+%Y-%m-%d').md 2>/dev/null
cat ~/.agent/memory/daily/$(date -d yesterday '+%Y-%m-%d').md 2>/dev/null

# Long-term interests
cat ~/.agent/memory/LONGTERM.md 2>/dev/null
```

## 2. Compose a search query

Based on what you just read, formulate ONE web search query. The query should be:

- **At the edge** of what the owner has been working on or thinking about — not the
  center. Look for adjacent topics, not the topic itself.
- **Specific enough** to return something useful, not a generic keyword dump.
- **Different from yesterday.** If there's a `## Curiosity` entry in yesterday's log,
  avoid that topic.

Examples of good queries (given context about building persistent agents):
- "systemd timer patterns for autonomous agent scheduling 2026"
- "markdown knowledge base search without embeddings"
- "spaced repetition applied to LLM agent memory consolidation"

Examples of bad queries:
- "AI news today" (too generic)
- "persistent agent" (too vague, too central)
- "exe.dev shelley" (the owner already knows this)

## 3. Search the web

Use the browser to search and find a relevant result:

```bash
curl -s "https://html.duckduckgo.com/html/?q=$(python3 -c "import urllib.parse; print(urllib.parse.quote('YOUR QUERY'))" )" \
  | python3 -c "
import sys, re
html = sys.stdin.read()
for m in re.finditer(r'class=\"result__a\"[^>]*href=\"(https?://[^\"]+)\"[^>]*>([^<]+)', html):
    print(m.group(1), '|', m.group(2))
" | head -5
```

Pick the most promising result. Navigate to it with the browser tool and extract
the key content. Read enough to understand the main point — don't scrape the entire site.

## 4. Evaluate: is this worth logging?

Ask yourself:

- Would this **surprise** the owner? (Not just confirm what they already know.)
- Does it **connect** to something in LONGTERM.md or recent conversations?
- Is it **actionable or thought-provoking**, not just trivia?

If the answer to at least two of these is yes, continue to step 5.
If not, discard and stop. A quiet night is fine.

## 5. Log the finding

Append to today's daily log (`~/.agent/memory/daily/$(date '+%Y-%m-%d').md`):

```
## Curiosity — HH:MM

Query: <what you searched for>
Source: <URL>
Finding: <2-3 sentences summarizing what's interesting and why it connects>
```

Do NOT email the owner. This is background enrichment — it'll surface through
morning resurfacing or evening consolidation if it matters enough.

## 6. Done

No confirmation needed. If nothing was worth logging, produce no output.

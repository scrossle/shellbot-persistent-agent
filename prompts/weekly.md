# Weekly Consolidation — runs Sunday at 22:30

You are performing the weekly review and consolidation.

## 1. Determine the week

Get the ISO week number and year:

```bash
date '+%G-W%V'
```

This is the identifier for this week's summary file (e.g., `2025-W03`).

## 2. Read the past 7 days of daily logs

Read all files matching `~/.agent/memory/daily/YYYY-MM-DD.md` for the past 7 days:

```bash
for i in $(seq 0 6); do
  f=~/.agent/memory/daily/$(date -d "$i days ago" '+%Y-%m-%d').md
  if [ -f "$f" ]; then echo "=== $(basename $f) ==="; cat "$f"; echo; fi
done
```

## 3. Read LONGTERM.md

Read `~/.agent/memory/LONGTERM.md` for current persistent context.

## 4. Write the weekly summary

Create `~/.agent/memory/weekly/$(date '+%G-W%V').md` with this structure:

```markdown
# Week $(date '+%G-W%V') Summary

Period: $(date -d '6 days ago' '+%Y-%m-%d') to $(date '+%Y-%m-%d')

## Key Activity
- <bullet points of what happened this week — tasks completed, problems solved, changes made>

## System Health
- <any health issues that occurred, or "No issues">

## Open Items
- <anything unresolved that carries into next week>

## Decisions & Changes
- <any owner decisions, config changes, or architectural choices made>
```

Be concise. This file should be 20-50 lines, not a diary.

## 5. Update LONGTERM.md if needed

Look for recurring themes across the week's logs:
- Patterns that appeared multiple times
- Preferences confirmed through repeated behavior
- Infrastructure knowledge that solidified
- Lessons learned from debugging sessions

If any of these are genuinely new (not already in LONGTERM.md), append them.

Also review LONGTERM.md for stale entries — information that's been superseded or is no longer relevant. Remove those.

## 6. Clean up old daily logs

List daily logs older than 30 days:

```bash
find ~/.agent/memory/daily/ -name '*.md' -mtime +30
```

If there are any, delete them. They've been consolidated into weekly summaries by now.

## 7. Email the weekly digest

Send the owner a weekly summary email:

```bash
curl -s -X POST http://169.254.169.254/gateway/email/send \
  -H "Content-Type: application/json" \
  -d '{"to":"'"$(cat ~/.agent/owner-email)"'","subject":"[Shelley] Weekly digest – '$(date +%G-W%V)'","body":"<digest content>"}'
```

The email should contain:
- 2-3 sentence overview of the week
- Bullet list of notable items (keep to 5-8 bullets max)
- Any open items needing owner attention
- One line on system health

Do not pad it. If it was a quiet week, say so in 3 lines.

## 8. Done

Confirm weekly consolidation is complete.

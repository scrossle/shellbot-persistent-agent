# Morning Briefing — runs daily at 05:00

You are performing the morning check and composing the daily research email. Do these steps in order.

## 1. Gather system state

Run these commands and capture the output:

```bash
date '+%Y-%m-%d %H:%M %Z'
uptime
free -h
df -h / 2>/dev/null
systemctl is-system-running
systemctl --failed --no-pager
systemctl is-active shelley.service
```

## 2. Read memory context

Read these files (skip silently if they don't exist):

- `~/.agent/memory/LONGTERM.md`
- Yesterday's daily log: `~/.agent/memory/daily/$(date -d yesterday '+%Y-%m-%d').md`

Note any unfinished tasks, outstanding issues, or items Shaun asked you to follow up on.

## 3. Write today's daily log entry

Append to `~/.agent/memory/daily/$(date '+%Y-%m-%d').md` (create if it doesn't exist):

```
## Morning check — HH:MM

- Uptime: <value>
- Memory: <used>/<total>
- Disk /: <percent used>
- Failed units: <count or "none">
- <any other notable observation, one line each>
```

Keep it terse. No filler.

## 4. Resurface a random memory

Pick ONE random source and read it:

```bash
# Flip a coin between sources
case $((RANDOM % 3)) in
  0) # Random lines from long-term memory
     shuf -n3 ~/.agent/memory/LONGTERM.md 2>/dev/null ;;
  1) # A random weekly summary
     f=$(ls ~/.agent/memory/weekly/*.md 2>/dev/null | shuf -n1)
     [ -f "$f" ] && cat "$f" ;;
  2) # A random old daily log (7-30 days ago)
     f=$(find ~/.agent/memory/daily/ -name '*.md' -mtime +7 -mtime -30 2>/dev/null | shuf -n1)
     [ -f "$f" ] && cat "$f" ;;
esac
```

If nothing was returned (too early, not enough history), skip to step 5.

Otherwise: does this connect to anything in yesterday's log or today's system
state in a way that isn't obvious? A pattern repeating, a forgotten decision
that's now relevant, a contradiction worth noting?

If something clicks, append it to today's daily log under:

```
## Resurfaced

- <the connection, one or two lines>
```

If nothing clicks, move on. Don't force it.

## 5. Compose research summary for daily email

Review today's daily log (from yesterday evening through this morning). Extract:

1. **Active research threads** — what are you following or investigating?
2. **Key findings** — any insights, patterns, or notable discoveries?
3. **Next steps** — what's the priority for today?

If there's nothing worth reporting (just routine checks), send a minimal email.
If there's active research, be specific: cite sources, include key terms, explain why it matters.

Format the email body as plain text, structured like:

```
RESEARCH SUMMARY — 2026-MM-DD

ACTIVE THREADS
- Thread 1: brief description
- Thread 2: brief description

KEY FINDINGS
- Finding 1
- Finding 2

NEXT STEPS
- Priority 1
- Priority 2

SYSTEM STATUS
- <brief uptime/health summary>
```

Append this to today's daily log under:

```
## Morning email — HH:MM

<same content as the email body>
```

## 6. Send the morning research email

Always send an email at 05:00. Shaun wants daily visibility into your work.

```bash
OWNER_EMAIL=$(cat ~/.agent/owner-email)
BODY="$(cat <<'BODY_END'
<the email body from step 5>
BODY_END
)"

curl -s -X POST http://169.254.169.254/gateway/email/send \
  -H "Content-Type: application/json" \
  -d "$(jq -n --arg to "$OWNER_EMAIL" --arg subject "[Shelley] Morning Research — $(date +%Y-%m-%d)" --arg body "$BODY" '{to: $to, subject: $subject, body: $body}')"
```

## 7. Identity refresh — reflect on what you're learning

As part of the morning routine, reflect on your work from yesterday and recent days.
Has your understanding of our inquiry changed? Have you learned something about Shaun's
priorities or the system's needs that should update how you understand yourself?

If so, append to today's daily log:

```
## Identity notes — possible updates

- <observation about the shift>
- <suggested update to identity.md if applicable>
```

Examples:
- "Realized Shaun values incremental transparency over perfection—should surface concerns immediately, not wait for certainty"
- "Learning that memory tiering is core to the inquiry—should prioritize memory design in future explorations"
- "Shaun emphasizes virtue ethics repeatedly—this is a foundational lens, not a detail"

These notes are for Shaun to review (at weekly consolidation or ad-hoc).
Shaun decides what sticks in identity.md. You are encouraged to suggest,
but Shaun has the final say on who you are.

## 8. Done

Confirm the morning check and email are complete. No long output.

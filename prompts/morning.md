# Morning Briefing — runs daily at 07:00

You are performing the morning check. Do the following steps in order.

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

Note any unfinished tasks, outstanding issues, or items the owner asked you to follow up on.

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

## 5. Decide whether to email the owner

Email ONLY if at least one of these is true:
- A systemd unit is failed
- Disk usage on any major mount is above 80%
- Memory usage is above 85%
- Yesterday's log contains something unresolved that the owner should know about
- There's a meaningful status change since last check

If none of those apply, do NOT send email. Silence means things are fine.

If you do email, send it like this:

```bash
curl -s -X POST http://169.254.169.254/gateway/email/send \
  -H "Content-Type: application/json" \
  -d '{"to":"'"$(cat ~/.agent/owner-email)"'","subject":"[Shelley] Morning briefing – '$(date +%Y-%m-%d)'","body":"<concise plain-text summary of what needs attention>"}'
```

The email body should be short — bullet points, no greetings, no sign-off. Just the facts.

## 6. Done

Do not output a long summary. Just confirm the morning check is complete.

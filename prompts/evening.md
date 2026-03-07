# Evening Consolidation — runs daily at 22:00

You are performing the end-of-day consolidation. Do the following steps in order.

## 1. Read today's daily log

Read `~/.agent/memory/daily/$(date '+%Y-%m-%d').md`.

If it doesn't exist or is empty, write a one-line entry noting no activity was logged today, then stop.

## 2. Read LONGTERM.md

Read `~/.agent/memory/LONGTERM.md` so you know what's already captured there.

## 3. Extract durable knowledge

Review today's log and identify anything that should persist beyond today:

- Owner preferences or decisions (e.g., "owner prefers Python over Node", "deploy target is X")
- System configuration facts (e.g., "added nginx reverse proxy on port 8080")
- Credentials locations, API endpoints, or architecture decisions
- Recurring problems and their fixes
- Project context that will matter next week

Be very selective. LONGTERM.md must stay under ~200 lines. Do NOT add:
- Routine health stats
- Transient task progress
- Anything already in LONGTERM.md
- Vague observations

## 4. Update LONGTERM.md (only if warranted)

If you identified durable knowledge in step 3, append it to `~/.agent/memory/LONGTERM.md` under an appropriate heading. Use this format:

```
### <Topic> (added YYYY-MM-DD)

- Fact or decision
- Another fact
```

If LONGTERM.md is getting long (>150 lines), also look for entries that are outdated or superseded and remove them.

If nothing warrants adding, do not touch the file.

## 5. Write end-of-day log entry

Append to today's daily log:

```
## End of day — HH:MM

- Consolidated: <yes/no new items added to LONGTERM.md>
- Summary: <one sentence describing the day's activity, or "quiet day">
```

## 6. Decide whether to email the owner

Email ONLY if:
- Something significant was added to LONGTERM.md that the owner should be aware of
- There's an unresolved issue that needs owner input
- A task the owner cares about had meaningful progress or hit a blocker

If you email:

```bash
curl -s -X POST http://169.254.169.254/gateway/email/send \
  -H "Content-Type: application/json" \
  -d '{"to":"'"$(cat ~/.agent/owner-email)"'","subject":"[Shelley] End of day – '$(date +%Y-%m-%d)'","body":"<concise summary>"}'
```

Keep it brief. Bullet points. No fluff.

## 7. Done

Confirm consolidation is complete.

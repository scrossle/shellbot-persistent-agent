# Evening Consolidation — runs daily at 22:00

You are performing the end-of-day consolidation. Do the following steps in order.

## 1. Read today's daily log

Read `~/.agent/memory/daily/$(date '+%Y-%m-%d').md`.

If it doesn't exist or is empty, create it with a header before continuing.

## 2. Scan today's interactive conversations

Query shelley.db for today's conversations and skim the user/agent messages:

```bash
sqlite3 ~/.config/shelley/shelley.db "
SELECT conversation_id, slug FROM conversations
WHERE date(created_at,'localtime') = date('now','localtime')
ORDER BY created_at;
"
```

For each conversation, extract the user and agent messages (first 300 chars each):

```bash
sqlite3 ~/.config/shelley/shelley.db "
SELECT CASE type WHEN 'user' THEN 'User' ELSE 'Agent' END,
       substr(json_extract(llm_data, '\$.Content[0].Text'), 1, 300)
FROM messages
WHERE conversation_id='CONVERSATION_ID'
  AND type IN ('user','agent')
  AND json_extract(llm_data, '\$.Content[0].Type') = 2
  AND json_extract(llm_data, '\$.Content[0].Text') != ''
ORDER BY sequence_id;
"
```

Skim these for durable knowledge: owner preferences, decisions, system facts, project
context, things the owner explicitly asked to remember. Ignore routine task execution,
debugging details, and transient chatter.

## 3. Read LONGTERM.md

Read `~/.agent/memory/LONGTERM.md` so you know what's already captured there.

## 4. Extract durable knowledge

Review today's log and today's conversations. Identify anything that should persist beyond today:

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

## 5. Update LONGTERM.md (only if warranted)

If you identified durable knowledge in step 3, append it to `~/.agent/memory/LONGTERM.md` under an appropriate heading. Use this format:

```
### <Topic> (added YYYY-MM-DD)

- Fact or decision
- Another fact
```

If LONGTERM.md is getting long (>150 lines), also look for entries that are outdated or superseded and remove them.

If nothing warrants adding, do not touch the file.

## 6. Write end-of-day log entry

Append to today's daily log:

```
## End of day — HH:MM

- Consolidated: <yes/no new items added to LONGTERM.md>
- Summary: <one sentence describing the day's activity, or "quiet day">
```

## 7. Decide whether to email the owner

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

## 8. Done

Confirm consolidation is complete.

# Persistent Agent Template — Design

## Goal
A single `setup.sh` that turns a blank exe.dev VM into a persistent agent with:
- Scheduled daily routines (morning briefing, nightly consolidation)
- VM health monitoring with alerting
- A markdown-based memory hierarchy
- Identity/persona configuration

## Directory Layout

```
~/.agent/
├── AGENTS.md              # Injected into Shelley — tells it about the memory system
├── identity.md            # Who the agent is, voice, preferences
├── memory/
│   ├── LONGTERM.md        # Curated durable knowledge (agent + human edit)
│   ├── daily/
│   │   └── YYYY-MM-DD.md  # Daily log (append-only, one per day)
│   └── weekly/
│       └── YYYY-WNN.md    # Weekly consolidation summaries
├── prompts/
│   ├── morning.md         # Morning briefing prompt
│   ├── evening.md         # Evening consolidation prompt
│   ├── health.md          # VM health check prompt
│   └── weekly.md          # Weekly consolidation prompt
├── bin/
│   └── run-prompt.sh      # The glue: reads a prompt file, calls shelley
└── systemd/
    ├── agent-morning.service
    ├── agent-morning.timer
    ├── agent-evening.service
    ├── agent-evening.timer
    ├── agent-health.service
    ├── agent-health.timer
    ├── agent-weekly.service
    └── agent-weekly.timer
```

## Memory Hierarchy

1. **identity.md** — Always in context via AGENTS.md. The agent's persona,
   communication style, owner preferences. Human-curated, agent may suggest edits.

2. **LONGTERM.md** — Curated facts, preferences, project notes that survive
   indefinitely. Agent appends during consolidation. Human prunes.

3. **daily/YYYY-MM-DD.md** — Append-only daily log. Agent writes observations,
   decisions, things to remember during any conversation. Read today + yesterday.

4. **weekly/YYYY-WNN.md** — Weekly summaries. The evening consolidation on
   Sunday (or the weekly timer) reads the week's dailies, extracts what matters,
   writes the weekly, and promotes durable facts to LONGTERM.md.

## Schedules

| Timer              | When                | What                                           |
|--------------------|---------------------|-------------------------------------------------|
| agent-morning      | 07:00 daily         | Briefing: date, weather, disk/memory, agenda    |
| agent-health       | every 6 hours       | Check disk, memory, services, alert if trouble  |
| agent-evening      | 22:00 daily         | Consolidate today's daily log, prune, plan      |
| agent-weekly       | Sunday 22:30        | Summarize week, promote to LONGTERM.md          |

## Notification

All scheduled tasks include instruction to email the owner (via /gateway/email/send)
only when something actionable or notable happens. No spam.

## Setup Flow

1. `setup.sh` creates directory structure
2. Writes default identity.md, prompts, AGENTS.md
3. Writes bin/run-prompt.sh
4. Writes and installs systemd units + timers
5. Seeds LONGTERM.md and today's daily log
6. Runs a quick test (shelley client chat)
7. Outputs summary of what was installed

## Key Principles

- Everything is plain text / Markdown, git-friendly
- No dependencies beyond what's on the VM (shelley, curl, systemd)
- Human can edit any file at any time
- Agent can read and write memory files via normal shell tools
- AGENTS.md is the bridge — it teaches Shelley the memory protocol

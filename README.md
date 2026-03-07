# Persistent Agent Template for exe.dev

Turns a blank exe.dev VM into a persistent agent with daily routines,
VM health monitoring, email alerts, and a tiered Markdown memory system.

No dependencies beyond what's already on the VM: Shelley, systemd, curl.

## Quick Start

```bash
git clone <this-repo> ~/persistent-agent
cd ~/persistent-agent
./setup.sh your@email.com
```

That's it. Your VM now has:

| Timer | Schedule | What it does |
|-------|----------|--------------|
| `agent-morning` | 07:00 daily | System check, resurfaces a random old memory, emails if needed |
| `agent-health` | Every 6 hours | Disk/memory/services check, alerts only on problems |
| `agent-evening` | 22:00 daily | Scans all conversations from shelley.db, extracts durable knowledge, finds connections |
| `agent-weekly` | Sunday 22:30 | Summarizes the week, cleans up old logs, emails digest |

## Memory Hierarchy

```
~/.agent/
├── identity.md              # Who the agent is (always in context)
├── memory/
│   ├── LONGTERM.md          # Curated durable knowledge
│   ├── daily/YYYY-MM-DD.md  # Append-only daily logs
│   └── weekly/YYYY-WNN.md   # Weekly consolidation summaries
├── prompts/                 # Scheduled task instructions
├── bin/run-prompt.sh        # Glue: feeds prompts to Shelley
└── logs/runs.log            # Audit trail of all scheduled runs
```

### How Memory Flows

```
  Interactive conversations (web UI, CLI)
         │
         │ stored automatically in shelley.db
         ▼
  Evening consolidation scans shelley.db ──► extracts durable knowledge
         │                                         │
         │ also reads today's daily log             │
         ▼                                         ▼
    daily/YYYY-MM-DD.md                      LONGTERM.md
    (ephemeral, kept 30 days)                (curated, ≤200 lines)
         │                                         │
         │ weekly consolidation                     │
         ▼                                         │
    weekly/YYYY-WNN.md ◄───── promotes recurring themes
    (archive)
```

- **Daily logs** are append-only scratch space. Cheap to write, auto-deleted after 30 days.
- **LONGTERM.md** is the durable store. Evening consolidation appends; humans prune.
  Target: under 200 lines. Quality over quantity.
- **Weekly summaries** are compressed archives. Useful for "what happened in January?"
- **identity.md** is the agent's persona. Rarely changed. Always in context.
- **shelley.db** is the raw source — every conversation is stored and queryable.
  The evening consolidation mines it automatically; you never need to tell the agent
  "remember this."

### Serendipity

Two mechanisms help the agent make non-obvious connections:

**Morning resurfacing.** Each morning, the agent pulls a random fragment from the
past — an old weekly summary, a few lines from LONGTERM.md, or a daily log from
1-4 weeks ago — and checks whether it connects to yesterday's activity. Connections
are logged under `## Resurfaced` in the daily log. If nothing clicks, it moves on.

**Evening associations.** During consolidation, the agent has all of today's
conversations and LONGTERM.md loaded simultaneously. It looks for topics that
rhyme across conversations, contradictions with existing memories, or patterns
forming over multiple days. Connections are logged under `## Connections`.

Both are optional — they produce nothing when nothing is there. They compound
over time as memory accumulates.

## How It Works

`run-prompt.sh` reads a prompt file, substitutes date variables, and calls:

```bash
shelley client chat -p "<prompt contents>" -model claude-haiku-4.5
```

Shelley runs the prompt as a full agent conversation (with shell, file, and browser
tools), then exits. systemd timers invoke `run-prompt.sh` on schedule.

The AGENTS.md file (installed to `~/.config/shelley/AGENTS.md`) teaches Shelley about
the memory system so that *all* conversations — scheduled and interactive — know
where memory lives and how to use it.

## Customization

### Change the schedule

```bash
sudo systemctl edit agent-morning.timer
# Add [Timer] section with your OnCalendar override
```

### Change the model

Edit `~/.agent/bin/run-prompt.sh` — the default is `claude-haiku-4.5` (cheap, capable).
Pass a second argument to override per-run: `run-prompt.sh prompt.md claude-sonnet-4.5`

### Add a new scheduled task

1. Write a prompt: `~/.agent/prompts/my-task.md`
2. Copy an existing service/timer pair, change the `ExecStart` and `OnCalendar`
3. `sudo systemctl daemon-reload && sudo systemctl enable --now agent-my-task.timer`

### Edit the agent's identity

```bash
nano ~/.agent/identity.md
```

### Manually trigger a task

```bash
~/.agent/bin/run-prompt.sh ~/.agent/prompts/morning.md
```

## Uninstall

```bash
sudo systemctl disable --now agent-{morning,evening,health,weekly}.timer
sudo rm /etc/systemd/system/agent-*
sudo systemctl daemon-reload
# Optionally: rm -rf ~/.agent
```

## Design Philosophy

- **Plain Markdown, git-friendly.** No vector DBs, no Redis, no external services.
- **systemd, not cron.** Timers are persistent (catch up after downtime), logged
  via journalctl, and manageable with standard tools.
- **Shelley, not a custom agent.** The prompt *is* the program. The agent already has
  shell, browser, file I/O, and subagent tools.
- **Email, not chat.** Alerts go to your inbox. Silence means healthy.
- **Human in the loop.** The owner edits LONGTERM.md and identity.md.
  The agent suggests; the human curates.
- **Conversations are automatically remembered.** The evening consolidation scans
  shelley.db for all of the day's conversations, so nothing falls through the cracks
  even if the agent didn't journal in real time.
- **Serendipity by design.** Random resurfacing and associative connection-making
  are built into the daily cycle, not bolted on.

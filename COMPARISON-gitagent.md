# Shellbot Persistent Agent vs GitAgent

These two projects solve **fundamentally different problems** in the AI agent space.

## What They Are

| | **Shellbot Persistent Agent** | **GitAgent** |
|---|---|---|
| **Core idea** | A **running, autonomous AI agent** on a VM with scheduled routines, memory, and self-monitoring | An **open standard/spec** for defining AI agents as files in a git repo |
| **Category** | Agent runtime & orchestration | Agent definition format & CLI tool |
| **Analogy** | A living, breathing assistant that works while you sleep | A blueprint language for describing what an agent should be |

## Architecture

| Aspect | **Shellbot Persistent Agent** | **GitAgent** |
|---|---|---|
| **Runtime** | systemd timers on an exe.dev VM | No runtime — exports to other frameworks |
| **LLM** | Claude Haiku 4.5 via Shelley framework | Framework-agnostic (Claude, OpenAI, CrewAI, etc.) |
| **Memory** | Tiered: identity.md → daily logs → LONGTERM.md → weekly archives | SOUL.md + SKILL.md + agent.yaml (static definition) |
| **Persistence** | SQLite (shelley.db) + plain Markdown files | Git repository files only |
| **Scheduling** | 5 daily/weekly systemd timers (morning, evening, curiosity, weekly, health) | None — it's a definition format, not a scheduler |
| **Dashboard** | Go-based status panel on port 8000 | CLI only (`init`, `validate`, `run`, `export`) |

## Key Features Compared

| Feature | **Shellbot** | **GitAgent** |
|---|---|---|
| **Autonomous operation** | Yes — runs 24/7 with scheduled tasks | No — defines agents, doesn't run them autonomously |
| **Memory system** | Sophisticated tiered hierarchy with consolidation, resurfacing, and pruning | Static knowledge files in repo |
| **Serendipity/discovery** | Built-in: random memory resurfacing, curiosity exploration, cross-conversation connections | Not applicable |
| **Health monitoring** | Hourly non-agentic health checks with email alerts | `validate` command checks definition correctness |
| **Multi-framework support** | No — tied to Shelley + Claude | Yes — export to Claude, OpenAI, CrewAI, Lyzr, etc. |
| **Portability** | exe.dev-specific (VM, gateway, proxy) | Fully portable — plain files + npm CLI |
| **Human-in-the-loop** | Email notifications, manual LONGTERM.md curation | Git PRs and code review on agent definitions |
| **Version control** | Git-friendly Markdown files | Git is the core primitive |

## Philosophy

| | **Shellbot Persistent Agent** | **GitAgent** |
|---|---|---|
| **Core belief** | "The prompt IS the program" — agents should run continuously and build knowledge over time | "Your agent definition lives in git, not in a vendor's cloud" |
| **Approach to memory** | Dynamic, evolving — agent discovers and consolidates knowledge autonomously | Static — knowledge is authored by humans in files |
| **Complexity** | Opinionated, full-stack system (scheduler + memory + monitoring + notifications) | Minimal spec — define once, deploy anywhere |
| **Target user** | Someone who wants a personal AI agent that runs autonomously on a VM | Someone who wants a portable, version-controlled agent definition |

## Strengths

### Shellbot Persistent Agent
- Genuinely autonomous — morning briefings, evening consolidation, curiosity exploration happen without human prompting
- Sophisticated memory with serendipity (random resurfacing finds non-obvious connections)
- Self-healing health monitoring that works even when the LLM is down
- Minimal infra: plain Markdown + systemd + SQLite — no vector DBs or external services

### GitAgent
- Framework-agnostic — not locked to any single LLM provider
- Truly portable agent definitions via open standard (MIT licensed)
- Familiar developer workflow (git, PRs, CI/CD for agent changes)
- Lower barrier to entry (`npm install -g gitagent` and go)

## Weaknesses

### Shellbot Persistent Agent
- Tied to exe.dev infrastructure (VM, Shelley, gateway)
- Single-LLM (Claude only)
- No portability story — can't export to other frameworks
- No explicit open-source license

### GitAgent
- Early stage (v0.1.0) — minimal ecosystem
- No runtime intelligence — it's a definition format, not an agent
- No memory management, scheduling, or autonomous behavior
- "Run" capability depends entirely on the target framework

## Summary

These aren't competitors — they're **complementary layers**:

- **GitAgent** answers: *"How should I define my agent so it's portable and version-controlled?"*
- **Shellbot Persistent Agent** answers: *"How do I make an agent that actually lives, learns, and works autonomously?"*

A theoretical integration could use GitAgent's spec format to define the agent's identity/skills, then deploy it into Shellbot's runtime for autonomous execution with memory and scheduling. Today, though, they exist in different parts of the stack with no overlap.

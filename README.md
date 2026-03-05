# 🦗 cicada

<p align="center">
  <img src="docs/img/cicada-2.png" alt="cicada" width="200">
</p>

<p align="center">
  <a href="https://github.com/base-14/cicada/actions/workflows/ci.yml"><img src="https://github.com/base-14/cicada/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/base-14/cicada/releases/latest"><img src="https://img.shields.io/github/v/release/base-14/cicada" alt="Release"></a>
  <a href="https://github.com/base-14/cicada/blob/main/LICENSE"><img src="https://img.shields.io/github/license/base-14/cicada" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/base-14/cicada"><img src="https://goreportcard.com/badge/github.com/base-14/cicada" alt="Go Report Card"></a>
</p>

TUI analytics dashboard for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions.

Reads your local `~/.claude/` data and shows session history, tool usage, project analytics, and more — all in the terminal. No server, no database — everything stays local.

## Install

**Homebrew:**

```
brew install base-14/tap/cicada
```

**Go:**

```
go install github.com/base-14/cicada@latest
```

## Usage

```
cicada
```

Navigate with arrow keys or vim bindings (`h/j/k/l`). Press `?` for help, `/` to filter, `Enter` to drill in, `Esc` to go back, `y` to copy `claude --resume <id>` to clipboard.

### Export / Import Sessions

Share sessions between machines or with teammates:

- `e` — Export selected session as a zip
- `E` — Export all visible sessions (respects active filter)
- `i` — Import session(s) from a zip file

Exported zips contain the raw JSONL session file plus a manifest. Imported sessions are fully Claude Code-compatible — recipients can browse them in cicada and resume with `claude --resume`.

## What it shows

```
cicada
├── Analysis
│   ├── Usage heatmap (GitHub-style activity grid)
│   ├── Sessions per day (sparkline)
│   ├── Messages & tools per session (bar charts)
│   ├── Streaks (current, longest, weekly)
│   ├── Personal bests (longest session, most messages, most tools)
│   └── Trends (sessions this week vs last, avg duration)
│
├── Projects
│   ├── All projects with session counts and last active
│   └── Project detail (Enter to drill in)
│       ├── Overview — total sessions, messages, duration
│       ├── Sessions — per-project session list
│       ├── Tools — tool usage breakdown for this project
│       ├── Activity — project-level heatmap
│       └── Skills — which skills were invoked
│
├── Sessions
│   ├── All sessions with project, duration, messages, cost
│   ├── Export / import sessions (e/E/i)
│   └── Session detail (Enter to drill in)
│       ├── Chat — full conversation: user prompts & assistant responses
│       ├── Overview — duration, message count, model, cost
│       ├── Timeline — chronological tool calls and messages
│       ├── Files — files read, written, and edited
│       ├── Agents — subagent spawns and results
│       └── Tools — per-session tool call breakdown
│
├── Agents
│   └── Subagent usage across all sessions
│
└── Tools
    ├── Built-in tools ranked by call count
    └── MCP server tools with server grouping
```

## License

[MIT](LICENSE)

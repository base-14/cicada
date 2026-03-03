# 🦗 cicada

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

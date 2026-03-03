# cicada

TUI analytics dashboard for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions.

Reads your local `~/.claude/` data and shows session history, tool usage, project analytics, and more — all in the terminal.

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

Navigate with arrow keys or vim bindings. Press `?` for help.

## What it shows

- **Sessions** — browse all Claude Code sessions with duration, message counts, cost
- **Projects** — per-project drill-down with tool usage and activity heatmaps
- **Analytics** — usage trends, heatmaps, and insights across all sessions
- **Tools** — which tools get called most, MCP server breakdown
- **Agents** — subagent usage patterns

## License

[MIT](LICENSE)

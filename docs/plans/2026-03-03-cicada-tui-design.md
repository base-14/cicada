# Cicada — Claude Code TUI Analyzer

## Overview

Standalone Go terminal app that reads Claude Code's local session data (`~/.claude/`) and presents it as an interactive TUI. No server, no database — everything in memory with a background scanner for progressive loading.

Inspired by [claude-code-karma](https://github.com/anthropics/claude-code-karma), which does the same via a FastAPI + SvelteKit web app.

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   TUI (Bubbletea)                │
│  Dashboard │ Projects │ Sessions │ Analytics │...│
│                                                  │
│  Subscribes to store updates via tea.Msg         │
└──────────────────────┬──────────────────────────┘
                       │
              ┌────────▼────────┐
              │   SessionStore  │  (sync.RWMutex-protected)
              │   In-memory map │
              │   of all parsed │
              │   session meta  │
              └────────▲────────┘
                       │ writes
              ┌────────┴────────┐
              │  Background     │
              │  Scanner        │  goroutine
              │                 │
              │  1. Discover    │  ~/.claude/projects/*/
              │  2. Parse JSONL │  lightweight metadata extraction
              │  3. Send tea.Msg│  per batch (every 50 sessions)
              │  4. Full parse  │  on-demand when user opens session
              └─────────────────┘
```

- Scanner sends `SessionsBatch` messages to the TUI in batches so views update progressively
- Store is behind `sync.RWMutex` — scanner writes, TUI reads
- Session detail data (timeline, file activity) parsed lazily on navigation, then cached
- Status bar shows scan progress: "Scanning... 342/1,203 sessions"

## Data Sources

All local filesystem, no network calls.

| Path | Content |
|---|---|
| `~/.claude/projects/*/` | Project directories (URL-encoded paths) |
| `~/.claude/projects/*/*.jsonl` | Session transcript files |
| `~/.claude/projects/*/subagents/agent-*.jsonl` | Subagent transcripts |
| `~/.claude/agents/` | Custom agent definitions |
| `~/.claude/skills/` | Custom skill files |
| `~/.claude/plans/` | Plan-mode markdown files |
| `~/.claude/todos/` | Legacy todo items |
| `~/.claude/tasks/` | New task system files |

## Data Model

### SessionMeta (lightweight, parsed during background scan)

```go
type SessionMeta struct {
    UUID           string
    Slug           string
    ProjectPath    string
    StartTime      time.Time
    EndTime        time.Time
    Duration       time.Duration
    InitialPrompt  string            // first user message
    SessionTitles  []string
    Models         map[string]int    // model → message count
    TokensIn       int64
    TokensOut      int64
    CacheRead      int64
    CacheWrite     int64
    ToolUsage      map[string]int    // tool name → count
    SkillsUsed     map[string]int
    CommandsUsed   map[string]int
    GitBranches    []string
    SubagentCount  int
    FileOps        map[string]int    // operation type → count
    MessageCount   int
}
```

### SessionDetail (full parse, loaded lazily)

```go
type SessionDetail struct {
    Meta         *SessionMeta
    Timeline     []TimelineEvent
    FileActivity []FileOp
    Subagents    []SubagentMeta
    Messages     []Message
}
```

### TimelineEvent

```go
type TimelineEvent struct {
    Timestamp time.Time
    Type      string  // "user", "assistant", "tool_use", "tool_result"
    Content   string  // truncated for display
    ToolName  string
    ActorID   string  // session or subagent agent_id
}
```

### SubagentMeta

```go
type SubagentMeta struct {
    AgentID       string
    Type          string            // "Explore", "Plan", etc.
    InitialPrompt string
    ToolUsage     map[string]int
    TokensIn      int64
    TokensOut     int64
    Duration      time.Duration
}
```

## Metadata Parsing Strategy

To keep the background scan fast, we read each JSONL file line by line but only extract lightweight fields: tokens, model, tool names (not inputs/outputs), timestamps, git branches. Full message content and tool inputs/outputs are skipped. This gives us everything needed for analytics and session lists.

## TUI Layout

```
┌──────────────────────────────────────────────────────────┐
│  cicada ◈  Dashboard  Projects  Sessions  Analytics      │
│           Agents  Tools                                  │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  (active view content)                                   │
│                                                          │
├──────────────────────────────────────────────────────────┤
│  Scanning... 842/1,203 sessions  │  ? help  q quit      │
└──────────────────────────────────────────────────────────┘
```

Navigation: Tab/Shift-Tab or number keys (1-6) for top-level views. Enter to drill in, Esc to go back. `/` to search/filter.

## Views

### 1. Dashboard (default landing)

- Stats row: Total Sessions | Total Tokens | Active Projects
- Sessions by date sparkline (last 30 days)
- Top 5 tools bar chart
- Model distribution (Opus/Sonnet/Haiku percentages)
- Work mode split (exploration/building/testing)
- Updates progressively as scanner discovers sessions

### 2. Projects

- Table: Name | Sessions | Last Active | Git?
- Enter → filtered session list for that project
- `/` to search

### 3. Sessions

- Table: Slug | Project | Date | Duration | Tokens | Tools
- Filter bar: project, branch, date range, search text
- Enter → Session Detail sub-view with tabs:
  - **Overview**: stats grid, initial prompt, models, branches
  - **Timeline**: chronological event list, tool calls expandable
  - **Files**: file operations table (path, op, actor, tool)
  - **Agents**: subagent list with type badges, tool usage
  - **Tools**: tool breakdown for this session

### 4. Analytics

- Sessions per day (sparkline/bar chart, last 30d)
- Temporal heatmap (7×24 grid, ascii-rendered)
- Tool usage ranking (horizontal bar chart)
- Model distribution breakdown
- Work mode distribution
- Time period filter: 7d / 30d / 90d / all

### 5. Agents

- Table: Agent Type | Runs | Last Used
- Sortable columns
- Enter → agent detail with session list

### 6. Tools

- Table: Tool Name | Server | Calls | Sessions
- MCP tools grouped by server
- Enter → tool detail

## Key Bindings

| Key | Action |
|---|---|
| `1-6` | Switch top-level view |
| `Tab/Shift-Tab` | Next/prev view |
| `Enter` | Drill into selected item |
| `Esc` | Go back |
| `/` | Open search/filter |
| `s` | Cycle sort column |
| `r` | Reverse sort order |
| `?` | Help overlay |
| `q` | Quit |

## Scanner Lifecycle

```
startup
  ├─ discover project dirs (fast, ~ms)
  ├─ send ProjectsDiscovered msg → TUI renders project list immediately
  ├─ for each project, glob *.jsonl files
  │   ├─ parse metadata (all lines, lightweight extraction)
  │   ├─ every 50 sessions → send SessionsBatch msg → TUI updates
  │   └─ discover subagent files under {uuid}/subagents/
  └─ send ScanComplete msg → TUI removes progress indicator
```

## Error Handling

- Missing/corrupt JSONL lines: skip and continue (count warnings)
- Missing `~/.claude/` directory: show empty state with helpful message
- Permission errors: surface in status bar, continue with accessible files

## Deferred to Future Iterations

- **Cost calculation**: model pricing tables, per-session/aggregate cost tracking
- **Live session monitoring**: hook scripts, real-time session state tracking
- **Settings editor**: read/write `~/.claude/settings.json`
- **Plans viewer**: rendered markdown plan display
- **Archived prompts**: `~/.claude/history.jsonl` parsing
- **Plugins browser**: installed plugin discovery
- **Session chains**: continuation/compaction tracking
- **SQLite index**: persistent cache for large session histories

## Tech Stack

- **Language**: Go
- **TUI framework**: Bubbletea + Lipgloss (Charm ecosystem)
- **Build**: Makefile

## Project Structure

```
cicada/
├── main.go
├── go.mod
├── Makefile
├── CLAUDE.md
│
├── internal/
│   ├── parser/
│   │   ├── jsonl.go         // line-by-line JSONL reader
│   │   ├── message.go       // message type parsing
│   │   ├── content.go       // content block parsing
│   │   └── usage.go         // token extraction
│   │
│   ├── store/
│   │   ├── store.go         // SessionStore with RWMutex
│   │   └── scanner.go       // background scanner goroutine
│   │
│   ├── model/
│   │   ├── session.go       // SessionMeta, SessionDetail
│   │   ├── timeline.go      // TimelineEvent, FileOp
│   │   ├── subagent.go      // SubagentMeta
│   │   └── analytics.go     // aggregated analytics types
│   │
│   └── tui/
│       ├── app.go           // root model, tab navigation
│       ├── styles.go        // Lipgloss theme
│       ├── components/
│       │   ├── table.go     // sortable, filterable table
│       │   ├── sparkline.go // ascii sparkline
│       │   ├── barchart.go  // horizontal bar chart
│       │   ├── heatmap.go   // 7×24 temporal heatmap
│       │   ├── tabs.go      // tab bar
│       │   ├── statusbar.go // scan progress
│       │   └── filter.go    // search/filter input
│       └── views/
│           ├── dashboard.go
│           ├── projects.go
│           ├── sessions.go
│           ├── session_detail.go
│           ├── analytics.go
│           ├── agents.go
│           └── tools.go
```

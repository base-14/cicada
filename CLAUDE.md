# Cicada — Claude Code TUI Analyzer

## Project Overview

Standalone Go terminal app using Bubbletea/Lipgloss that reads Claude Code's local session data (`~/.claude/`) and presents analytics, session browsing, and tool usage in an interactive TUI. No server, no database — in-memory with background scanning.

See `docs/plans/2026-03-03-cicada-tui-design.md` for the full design.

## Tech Stack

- Go (latest stable)
- Bubbletea + Lipgloss (Charm ecosystem) for TUI
- No external database — all in-memory

## Code Quality Expectations

### Test Driven Development

- Always write tests BEFORE writing implementation code.
- Run the failing test to confirm it fails for the right reason.
- Write the minimum implementation to make the test pass.
- Run tests again to confirm they pass.
- Only commit when tests are passing.
- Do not skip this cycle. No exceptions.

### Build Commands

- Use `make` as the single entry point for all build, test, lint, and dev commands.
- Do not invoke `go test`, `go build`, `go vet`, `golangci-lint`, or other tools directly — use the corresponding Makefile target.
- If a new build/dev command is needed, add a Makefile target for it first.

## Makefile Targets

| Target | Purpose |
|---|---|
| `make build` | Build the binary |
| `make test` | Run all tests |
| `make lint` | Run linter |
| `make fmt` | Format code |
| `make run` | Build and run |
| `make clean` | Remove build artifacts |

## Project Structure

```
cicada/
├── main.go
├── internal/
│   ├── parser/      # JSONL parsing
│   ├── store/       # In-memory store + background scanner
│   ├── model/       # Domain types
│   └── tui/         # Bubbletea UI
│       ├── components/  # Reusable UI widgets
│       └── views/       # Top-level views
```

## Conventions

- Keep packages small and focused — no circular dependencies.
- `parser` knows about JSONL, `store` knows about indexing/querying, `model` is pure data types, `tui` is presentation only.
- Error handling: skip corrupt data and continue, surface warnings in status bar.
- Table-driven tests preferred.

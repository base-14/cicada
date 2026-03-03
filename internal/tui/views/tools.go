package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/store"
)

// ToolRow holds display data for a tool.
type ToolRow struct {
	Name         string
	Server       string // "builtin" or MCP server name
	Calls        int
	SessionCount int
}

// ToolsView shows all tools with call counts and session counts.
type ToolsView struct {
	store    *store.Store
	selected int
}

// NewToolsView creates a new ToolsView.
func NewToolsView(s *store.Store) *ToolsView {
	return &ToolsView{store: s}
}

// Update handles key events for navigation.
func (v *ToolsView) Update(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyUp:
		if v.selected > 0 {
			v.selected--
		}
	case tea.KeyDown:
		v.selected++
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "k":
			if v.selected > 0 {
				v.selected--
			}
		case "j":
			v.selected++
		}
	}
}

// parseMCPTool splits an MCP tool name into server and tool parts.
// e.g. "mcp__playwright__click" -> ("playwright", "click")
// e.g. "mcp__my_server__do_thing" -> ("my_server", "do_thing")
func parseMCPTool(name string) (server, tool string) {
	// Strip "mcp__" prefix
	rest := strings.TrimPrefix(name, "mcp__")
	// Split on first "__" to get server and tool
	parts := strings.SplitN(rest, "__", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return rest, rest
}

// buildToolRows aggregates tool data from all sessions.
func (v *ToolsView) buildToolRows() []ToolRow {
	sessions := v.store.AllSessions()

	// Aggregate: tool name -> total calls and set of session UUIDs
	type toolAgg struct {
		calls    int
		sessions map[string]bool
	}
	agg := make(map[string]*toolAgg)

	for _, s := range sessions {
		for tool, count := range s.ToolUsage {
			if _, ok := agg[tool]; !ok {
				agg[tool] = &toolAgg{sessions: make(map[string]bool)}
			}
			agg[tool].calls += count
			agg[tool].sessions[s.UUID] = true
		}
	}

	rows := make([]ToolRow, 0, len(agg))
	for name, a := range agg {
		server := "builtin"
		displayName := name
		if strings.HasPrefix(name, "mcp__") {
			srv, tool := parseMCPTool(name)
			server = srv
			displayName = tool
		}
		rows = append(rows, ToolRow{
			Name:         displayName,
			Server:       server,
			Calls:        a.calls,
			SessionCount: len(a.sessions),
		})
	}

	// Sort by call count descending
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Calls > rows[j].Calls
	})

	return rows
}

// View renders the tools list.
func (v *ToolsView) View(width, height int) string {
	sessions := v.store.AllSessions()

	if len(sessions) == 0 {
		return "\n  No session data available. Waiting for scan to complete..."
	}

	rows := v.buildToolRows()
	if len(rows) == 0 {
		return "\n  No tool usage data found."
	}

	// Clamp selected
	if v.selected >= len(rows) {
		v.selected = len(rows) - 1
	}

	// Separate built-in and MCP tools
	var builtins []ToolRow
	mcpByServer := make(map[string][]ToolRow)

	for _, row := range rows {
		if row.Server == "builtin" {
			builtins = append(builtins, row)
		} else {
			mcpByServer[row.Server] = append(mcpByServer[row.Server], row)
		}
	}

	var b strings.Builder
	b.WriteString("\n")

	lineIdx := 0

	// Built-in tools section
	if len(builtins) > 0 {
		b.WriteString("  Built-in Tools\n")
		b.WriteString(fmt.Sprintf("  %-20s %10s %10s\n", "Tool", "Calls", "Sessions"))
		b.WriteString("  " + strings.Repeat("\u2500", 42) + "\n")

		for _, row := range builtins {
			prefix := "  "
			if lineIdx == v.selected {
				prefix = "> "
			}
			b.WriteString(fmt.Sprintf("%s%-20s %10d %10d\n", prefix, row.Name, row.Calls, row.SessionCount))
			lineIdx++
		}
		b.WriteString("\n")
	}

	// MCP tools section, grouped by server
	// Sort server names for deterministic output
	var servers []string
	for srv := range mcpByServer {
		servers = append(servers, srv)
	}
	sort.Strings(servers)

	if len(servers) > 0 {
		b.WriteString("  MCP Tools\n")

		for _, srv := range servers {
			b.WriteString(fmt.Sprintf("  [%s]\n", srv))
			b.WriteString(fmt.Sprintf("  %-20s %10s %10s\n", "Tool", "Calls", "Sessions"))
			b.WriteString("  " + strings.Repeat("\u2500", 42) + "\n")

			for _, row := range mcpByServer[srv] {
				prefix := "  "
				if lineIdx == v.selected {
					prefix = "> "
				}
				b.WriteString(fmt.Sprintf("%s%-20s %10d %10d\n", prefix, row.Name, row.Calls, row.SessionCount))
				lineIdx++
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

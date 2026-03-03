package views

import (
	"fmt"
	"strings"

	"github.com/r/cicada/internal/store"
)

// AgentsView shows aggregate subagent usage statistics.
type AgentsView struct {
	store *store.Store
}

// NewAgentsView creates a new AgentsView.
func NewAgentsView(s *store.Store) *AgentsView {
	return &AgentsView{store: s}
}

// View renders the agents usage summary.
func (v *AgentsView) View(width, height int) string {
	sessions := v.store.AllSessions()

	if len(sessions) == 0 {
		return "\n  No session data available. Waiting for scan to complete..."
	}

	totalSubagents := 0
	sessionsWithSubagents := 0
	totalAgentToolCalls := 0

	for _, s := range sessions {
		totalSubagents += s.SubagentCount
		if s.SubagentCount > 0 {
			sessionsWithSubagents++
		}
		if calls, ok := s.ToolUsage["Agent"]; ok {
			totalAgentToolCalls += calls
		}
	}

	avgPerSession := 0.0
	if sessionsWithSubagents > 0 {
		avgPerSession = float64(totalSubagents) / float64(sessionsWithSubagents)
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  Subagent Usage Summary\n")
	b.WriteString("  " + strings.Repeat("\u2500", 50) + "\n\n")

	fmt.Fprintf(&b, "  %-35s %10d\n", "Total subagent invocations", totalSubagents)
	fmt.Fprintf(&b, "  %-35s %10d\n", "Agent tool calls", totalAgentToolCalls)
	fmt.Fprintf(&b, "  %-35s %10d\n", "Sessions using subagents", sessionsWithSubagents)
	fmt.Fprintf(&b, "  %-35s %10d\n", "Total sessions", len(sessions))
	fmt.Fprintf(&b, "  %-35s %10.1f\n", "Avg subagents per session (when used)", avgPerSession)

	b.WriteString("\n")

	// Adoption rate
	if len(sessions) > 0 {
		pct := sessionsWithSubagents * 100 / len(sessions)
		fmt.Fprintf(&b, "  Subagent adoption: %d%% of sessions\n", pct)
	}

	return b.String()
}

package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

var detailTabNames = []string{"Overview", "Timeline", "Files", "Agents", "Tools"}

// SessionDetailView shows detailed information about a single session.
type SessionDetailView struct {
	store     *store.Store
	session   *model.SessionMeta
	detail    *model.SessionDetail
	activeTab int
	scrollY   int
}

// NewSessionDetailView creates a new SessionDetailView.
func NewSessionDetailView(s *store.Store, session *model.SessionMeta, detail *model.SessionDetail) *SessionDetailView {
	return &SessionDetailView{
		store:   s,
		session: session,
		detail:  detail,
	}
}

// Update handles key events for sub-tab navigation and scrolling.
func (v *SessionDetailView) Update(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyLeft:
		v.activeTab = (v.activeTab - 1 + len(detailTabNames)) % len(detailTabNames)
		v.scrollY = 0
	case tea.KeyRight:
		v.activeTab = (v.activeTab + 1) % len(detailTabNames)
		v.scrollY = 0
	case tea.KeyUp:
		if v.scrollY > 0 {
			v.scrollY--
		}
	case tea.KeyDown:
		v.scrollY++
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "h":
			v.activeTab = (v.activeTab - 1 + len(detailTabNames)) % len(detailTabNames)
			v.scrollY = 0
		case "l":
			v.activeTab = (v.activeTab + 1) % len(detailTabNames)
			v.scrollY = 0
		case "k":
			if v.scrollY > 0 {
				v.scrollY--
			}
		case "j":
			v.scrollY++
		}
	}
}

// View renders the session detail view.
func (v *SessionDetailView) View(width, height int) string {
	var b strings.Builder
	b.WriteString("\n")

	// Sub-tab bar
	b.WriteString("  ")
	for i, name := range detailTabNames {
		if i == v.activeTab {
			b.WriteString("[" + name + "]")
		} else {
			b.WriteString(" " + name + " ")
		}
		if i < len(detailTabNames)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")
	b.WriteString("  " + strings.Repeat("\u2500", width-4) + "\n")

	// Content
	var content string
	switch v.activeTab {
	case 0:
		content = v.renderOverview(width)
	case 1:
		content = v.renderTimeline(width, height-6)
	case 2:
		content = v.renderFiles(width, height-6)
	case 3:
		content = v.renderAgents(width)
	case 4:
		content = v.renderTools(width)
	}

	// Apply scroll
	lines := strings.Split(content, "\n")
	if v.scrollY >= len(lines) {
		v.scrollY = max(0, len(lines)-1)
	}
	visibleHeight := height - 6
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	end := v.scrollY + visibleHeight
	if end > len(lines) {
		end = len(lines)
	}
	if v.scrollY < len(lines) {
		b.WriteString(strings.Join(lines[v.scrollY:end], "\n"))
	}

	return b.String()
}

func (v *SessionDetailView) renderOverview(width int) string {
	m := v.session
	var b strings.Builder

	// Title
	slug := m.Slug
	if slug == "" {
		slug = m.UUID
	}
	b.WriteString(fmt.Sprintf("  Session: %s\n\n", slug))

	// Stats grid
	totalTools := 0
	for _, c := range m.ToolUsage {
		totalTools += c
	}

	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Duration:", formatDuration(m.Duration)))
	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Tokens In:", formatTokensShort(m.TokensIn)))
	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Tokens Out:", formatTokensShort(m.TokensOut)))
	b.WriteString(fmt.Sprintf("  %-15s %d\n", "Messages:", m.MessageCount))
	b.WriteString(fmt.Sprintf("  %-15s %d\n", "Tools:", totalTools))
	b.WriteString(fmt.Sprintf("  %-15s %d\n", "Subagents:", m.SubagentCount))
	b.WriteString("\n")

	// Initial prompt
	if m.InitialPrompt != "" {
		prompt := m.InitialPrompt
		if len(prompt) > width-6 {
			prompt = prompt[:width-9] + "..."
		}
		b.WriteString("  Initial Prompt:\n")
		b.WriteString("  " + prompt + "\n\n")
	}

	// Models
	if len(m.Models) > 0 {
		b.WriteString("  Models:\n")
		for name, count := range m.Models {
			b.WriteString(fmt.Sprintf("    %-40s %d messages\n", name, count))
		}
		b.WriteString("\n")
	}

	// Git branches
	if len(m.GitBranches) > 0 {
		b.WriteString("  Git Branches: " + strings.Join(m.GitBranches, ", ") + "\n")
	}

	return b.String()
}

func (v *SessionDetailView) renderTimeline(width, maxRows int) string {
	if v.detail == nil || len(v.detail.Timeline) == 0 {
		return "  No timeline events."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %-8s %-12s %-15s %s\n", "Time", "Type", "Tool", "Content"))
	b.WriteString("  " + strings.Repeat("\u2500", width-4) + "\n")

	for _, ev := range v.detail.Timeline {
		timeStr := ""
		if !ev.Timestamp.IsZero() {
			timeStr = ev.Timestamp.Format("15:04:05")
		}

		content := ev.Content
		maxContent := width - 42
		if maxContent < 10 {
			maxContent = 10
		}
		if len(content) > maxContent {
			content = content[:maxContent-3] + "..."
		}

		line := fmt.Sprintf("  %-8s %-12s %-15s %s", timeStr, ev.Type, ev.ToolName, content)
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (v *SessionDetailView) renderFiles(width, maxRows int) string {
	if v.detail == nil || len(v.detail.FileActivity) == 0 {
		return "  No file activity."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %-50s %-10s %s\n", "Path", "Operation", "Tool"))
	b.WriteString("  " + strings.Repeat("\u2500", width-4) + "\n")

	for _, fa := range v.detail.FileActivity {
		path := fa.Path
		if len(path) > 50 {
			path = "..." + path[len(path)-47:]
		}
		line := fmt.Sprintf("  %-50s %-10s %s", path, fa.Operation, fa.ToolName)
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (v *SessionDetailView) renderAgents(width int) string {
	if v.detail == nil || len(v.detail.Subagents) == 0 {
		return "  No subagents in this session."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %-20s %-15s %s\n", "Agent ID", "Type", "Tools"))
	b.WriteString("  " + strings.Repeat("\u2500", width-4) + "\n")

	for _, agent := range v.detail.Subagents {
		totalTools := 0
		for _, c := range agent.ToolUsage {
			totalTools += c
		}
		agentID := agent.AgentID
		if len(agentID) > 20 {
			agentID = agentID[:17] + "..."
		}
		line := fmt.Sprintf("  %-20s %-15s %d", agentID, agent.Type, totalTools)
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (v *SessionDetailView) renderTools(width int) string {
	if v.session == nil || len(v.session.ToolUsage) == 0 {
		return "  No tool usage in this session."
	}

	// Sort by count descending
	type toolEntry struct {
		name  string
		count int
	}
	var entries []toolEntry
	for name, count := range v.session.ToolUsage {
		entries = append(entries, toolEntry{name, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %-30s %10s\n", "Tool", "Count"))
	b.WriteString("  " + strings.Repeat("\u2500", 42) + "\n")

	for _, e := range entries {
		b.WriteString(fmt.Sprintf("  %-30s %10d\n", e.name, e.count))
	}

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

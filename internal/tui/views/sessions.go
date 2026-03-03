package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

// SessionsView shows a sortable list of all sessions.
type SessionsView struct {
	store    *store.Store
	selected int
	rows     []*model.SessionMeta // cached sorted list
}

// NewSessionsView creates a new SessionsView.
func NewSessionsView(s *store.Store) *SessionsView {
	return &SessionsView{store: s}
}

// refreshRows fetches and sorts sessions from the store (newest first).
func (v *SessionsView) refreshRows() {
	v.rows = v.store.AllSessions()
	sort.Slice(v.rows, func(i, j int) bool {
		return v.rows[i].StartTime.After(v.rows[j].StartTime)
	})
}

// Update handles key events for arrow navigation.
func (v *SessionsView) Update(msg tea.KeyMsg) {
	v.refreshRows()
	switch msg.Type {
	case tea.KeyUp:
		if v.selected > 0 {
			v.selected--
		}
	case tea.KeyDown:
		if v.selected < len(v.rows)-1 {
			v.selected++
		}
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "k":
			if v.selected > 0 {
				v.selected--
			}
		case "j":
			if v.selected < len(v.rows)-1 {
				v.selected++
			}
		}
	}
}

// SelectedSession returns the currently selected session, or nil.
func (v *SessionsView) SelectedSession() *model.SessionMeta {
	v.refreshRows()
	if len(v.rows) == 0 || v.selected >= len(v.rows) {
		return nil
	}
	return v.rows[v.selected]
}

// View renders the sessions list.
func (v *SessionsView) View(width, height int) string {
	v.refreshRows()

	if len(v.rows) == 0 {
		return "\n  No sessions found. Waiting for scan to complete..."
	}

	var b strings.Builder
	b.WriteString("\n")

	// Header
	header := fmt.Sprintf("  %-20s %-30s %-12s %10s %10s %6s",
		"Slug", "Project", "Date", "Duration", "Tokens", "Tools")
	b.WriteString(header + "\n")
	b.WriteString("  " + strings.Repeat("\u2500", 92) + "\n")

	// Limit visible rows to available height
	maxRows := height - 5 // header + separator + padding
	if maxRows < 1 {
		maxRows = 1
	}

	for i, row := range v.rows {
		if i >= maxRows {
			break
		}

		slug := row.Slug
		if slug == "" {
			slug = row.UUID[:8]
		}
		if len(slug) > 20 {
			slug = slug[:17] + "..."
		}

		project := "/" + strings.ReplaceAll(strings.TrimPrefix(row.ProjectPath, "-"), "-", "/")
		if len(project) > 30 {
			project = "..." + project[len(project)-27:]
		}

		date := ""
		if !row.StartTime.IsZero() {
			date = row.StartTime.Format("Jan 02 15:04")
		}

		duration := formatDuration(row.Duration)
		tokens := formatTokensShort(row.TokensIn + row.TokensOut)

		toolCount := 0
		for _, c := range row.ToolUsage {
			toolCount += c
		}

		prefix := "  "
		if i == v.selected {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%-20s %-30s %-12s %10s %10s %6d",
			prefix, slug, project, date, duration, tokens, toolCount)
		b.WriteString(line + "\n")
	}

	return b.String()
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func formatTokensShort(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

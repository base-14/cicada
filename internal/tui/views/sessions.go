package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/base-14/cicada/internal/model"
	"github.com/base-14/cicada/internal/store"
	"github.com/base-14/cicada/internal/tui/components"
)

// SessionsView shows a sortable list of all sessions.
type SessionsView struct {
	store    *store.Store
	selected int
	rows     []*model.SessionMeta // cached sorted list
	filter   *components.Filter
	lastKey  string               // track last key for gg detection
}

// NewSessionsView creates a new SessionsView.
func NewSessionsView(s *store.Store) *SessionsView {
	return &SessionsView{store: s, filter: components.NewFilter()}
}

// refreshRows fetches and sorts sessions from the store (newest first).
// If the filter has a query, only matching sessions are included.
func (v *SessionsView) refreshRows() {
	all := v.store.AllSessions()
	if v.filter.Query == "" {
		v.rows = all
	} else {
		v.rows = make([]*model.SessionMeta, 0)
		for _, s := range all {
			project := "/" + strings.ReplaceAll(strings.TrimPrefix(s.ProjectPath, "-"), "-", "/")
			if v.filter.Matches(s.Slug) || v.filter.Matches(s.InitialPrompt) || v.filter.Matches(project) {
				v.rows = append(v.rows, s)
			}
		}
	}
	sort.Slice(v.rows, func(i, j int) bool {
		return v.rows[i].StartTime.After(v.rows[j].StartTime)
	})
}

// Update handles key events for arrow navigation and filter.
func (v *SessionsView) Update(msg tea.KeyMsg) {
	// Forward to filter first
	if v.filter.Update(msg) {
		v.selected = 0
		v.lastKey = ""
		return
	}

	v.refreshRows()
	maxIdx := len(v.rows) - 1
	switch msg.Type {
	case tea.KeyUp:
		if v.selected > 0 {
			v.selected--
		}
	case tea.KeyDown:
		if v.selected < maxIdx {
			v.selected++
		}
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "k":
			if v.selected > 0 {
				v.selected--
			}
		case "j":
			if v.selected < maxIdx {
				v.selected++
			}
		case "g":
			if v.lastKey == "g" {
				v.selected = 0
			} else {
				v.lastKey = "g"
				return
			}
		case "G":
			if maxIdx >= 0 {
				v.selected = maxIdx
			}
		}
	}
	v.lastKey = ""
}

// FilterActive returns true if the filter input is active.
func (v *SessionsView) FilterActive() bool {
	return v.filter.Active
}

// Selected returns the current selected index.
func (v *SessionsView) Selected() int {
	return v.selected
}

// SelectedSession returns the currently selected session, or nil.
func (v *SessionsView) SelectedSession() *model.SessionMeta {
	v.refreshRows()
	if len(v.rows) == 0 || v.selected >= len(v.rows) {
		return nil
	}
	return v.rows[v.selected]
}

// VisibleSessions returns all currently visible (filtered) sessions.
func (v *SessionsView) VisibleSessions() []*model.SessionMeta {
	v.refreshRows()
	return v.rows
}

// View renders the sessions list.
func (v *SessionsView) View(width, height int) string {
	v.refreshRows()

	var b strings.Builder
	b.WriteString("\n")

	// Filter bar
	if filterView := v.filter.View(); filterView != "" {
		b.WriteString("  " + filterView + "\n")
	}

	if len(v.rows) == 0 {
		if v.filter.Query != "" {
			b.WriteString("  No matching sessions.")
			return b.String()
		}
		return "\n  No sessions found. Waiting for scan to complete..."
	}

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

	// Calculate scrolling window
	start := 0
	end := len(v.rows)

	if len(v.rows) > maxRows {
		// Keep selected item in view with some context
		if v.selected >= maxRows {
			start = v.selected - maxRows + 1
		}
		end = start + maxRows
		if end > len(v.rows) {
			end = len(v.rows)
			start = end - maxRows
			if start < 0 {
				start = 0
			}
		}
	}

	for i := start; i < end; i++ {
		row := v.rows[i]

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

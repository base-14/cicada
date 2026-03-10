package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/base-14/cicada/internal/store"
	"github.com/base-14/cicada/internal/tui/components"
)

// ProjectRow holds display data for a project.
type ProjectRow struct {
	Name         string
	Path         string
	SessionCount int
	LastActive   string
}

// ProjectsView shows a list of projects.
type ProjectsView struct {
	store    *store.Store
	selected int
	filter   *components.Filter
	lastRows []ProjectRow
	lastKey  string // track last key for gg detection
}

// NewProjectsView creates a new ProjectsView.
func NewProjectsView(s *store.Store) *ProjectsView {
	return &ProjectsView{store: s, filter: components.NewFilter()}
}

// Update handles key events for navigation and filter.
func (v *ProjectsView) Update(msg tea.KeyMsg) {
	// Forward to filter first
	if v.filter.Update(msg) {
		v.selected = 0
		v.lastKey = ""
		return
	}

	maxIdx := len(v.lastRows) - 1
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
func (v *ProjectsView) FilterActive() bool {
	return v.filter.Active
}

// Selected returns the current selected index.
func (v *ProjectsView) Selected() int {
	return v.selected
}

// SelectedProject returns the Path of the currently selected project,
// or "" if no projects are available.
func (v *ProjectsView) SelectedProject() string {
	if len(v.lastRows) == 0 {
		return ""
	}
	idx := v.selected
	if idx >= len(v.lastRows) {
		idx = len(v.lastRows) - 1
	}
	return v.lastRows[idx].Path
}

// View renders the projects list.
func (v *ProjectsView) View(width, height int) string {
	projects := v.store.Projects()
	if len(projects) == 0 {
		return "\n  No projects found. Waiting for scan to complete..."
	}

	rows := make([]ProjectRow, 0, len(projects))
	for _, p := range projects {
		decoded := "/" + strings.ReplaceAll(strings.TrimPrefix(p, "-"), "-", "/")

		// Apply filter on decoded project name
		if v.filter.Query != "" && !v.filter.Matches(decoded) {
			continue
		}

		sessions := v.store.SessionsByProject(p)
		lastActive := ""
		for _, s := range sessions {
			if !s.StartTime.IsZero() {
				ts := s.StartTime.Format("2006-01-02 15:04")
				if ts > lastActive {
					lastActive = ts
				}
			}
		}

		rows = append(rows, ProjectRow{
			Name:         decoded,
			Path:         p,
			SessionCount: len(sessions),
			LastActive:   lastActive,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].LastActive > rows[j].LastActive
	})

	v.lastRows = rows

	// Clamp selected
	if v.selected >= len(rows) && len(rows) > 0 {
		v.selected = len(rows) - 1
	}

	var b strings.Builder
	b.WriteString("\n")

	// Filter bar
	if filterView := v.filter.View(); filterView != "" {
		b.WriteString("  " + filterView + "\n")
	}

	header := fmt.Sprintf("  %-50s %10s %20s", "Project", "Sessions", "Last Active")
	b.WriteString(header + "\n")
	b.WriteString("  " + strings.Repeat("\u2500", 82) + "\n")

	// Calculate scrolling window
	maxRows := height - 5 // header + separator + padding
	if maxRows < 1 {
		maxRows = 1
	}

	start := 0
	end := len(rows)

	if len(rows) > maxRows {
		// Keep selected item in view with some context
		if v.selected >= maxRows {
			start = v.selected - maxRows + 1
		}
		end = start + maxRows
		if end > len(rows) {
			end = len(rows)
			start = end - maxRows
			if start < 0 {
				start = 0
			}
		}
	}

	for i := start; i < end; i++ {
		row := rows[i]
		name := row.Name
		if len(name) > 50 {
			name = "..." + name[len(name)-47:]
		}
		prefix := "  "
		if i == v.selected {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-50s %10d %20s", prefix, name, row.SessionCount, row.LastActive)
		b.WriteString(line + "\n")
	}

	return b.String()
}

package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/store"
	"github.com/r/cicada/internal/tui/components"
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
		return
	}

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

// FilterActive returns true if the filter input is active.
func (v *ProjectsView) FilterActive() bool {
	return v.filter.Active
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

	for i, row := range rows {
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

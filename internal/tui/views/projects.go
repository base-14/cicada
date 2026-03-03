package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/r/cicada/internal/store"
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
}

// NewProjectsView creates a new ProjectsView.
func NewProjectsView(s *store.Store) *ProjectsView {
	return &ProjectsView{store: s}
}

// View renders the projects list.
func (v *ProjectsView) View(width, height int) string {
	projects := v.store.Projects()
	if len(projects) == 0 {
		return "\n  No projects found. Waiting for scan to complete..."
	}

	rows := make([]ProjectRow, 0, len(projects))
	for _, p := range projects {
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

		decoded := "/" + strings.ReplaceAll(strings.TrimPrefix(p, "-"), "-", "/")

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

	var b strings.Builder
	b.WriteString("\n")
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

package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ProjectPicker lets the user select a project path from a list or enter a custom one.
type ProjectPicker struct {
	Active         bool
	Projects       []string // existing project paths
	Selected       int
	CustomInput    string
	EnteringCustom bool
	OriginalPath   string // highlight if exists
}

// NewProjectPicker creates a new ProjectPicker with the given projects and original path.
func NewProjectPicker(projects []string, originalPath string) *ProjectPicker {
	return &ProjectPicker{
		Projects:     projects,
		OriginalPath: originalPath,
	}
}

// Update handles key events. Returns true if the key was consumed.
func (pp *ProjectPicker) Update(msg tea.KeyMsg) bool {
	if !pp.Active {
		return false
	}

	if pp.EnteringCustom {
		return pp.updateCustom(msg)
	}

	return pp.updateList(msg)
}

func (pp *ProjectPicker) updateCustom(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyEsc:
		pp.EnteringCustom = false
		return true
	case tea.KeyTab:
		pp.EnteringCustom = false
		return true
	case tea.KeyEnter:
		pp.Active = false
		return true
	case tea.KeyBackspace:
		if len(pp.CustomInput) > 0 {
			pp.CustomInput = pp.CustomInput[:len(pp.CustomInput)-1]
		}
		return true
	case tea.KeyRunes:
		pp.CustomInput += string(msg.Runes)
		return true
	}
	return true
}

func (pp *ProjectPicker) updateList(msg tea.KeyMsg) bool {
	maxIdx := len(pp.Projects) // includes custom entry at the end

	switch msg.Type {
	case tea.KeyEsc:
		pp.Active = false
		return true
	case tea.KeyTab:
		pp.EnteringCustom = true
		return true
	case tea.KeyEnter:
		pp.Active = false
		return true
	case tea.KeyUp:
		if pp.Selected > 0 {
			pp.Selected--
		}
		return true
	case tea.KeyDown:
		if pp.Selected < maxIdx {
			pp.Selected++
		}
		return true
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			if pp.Selected < maxIdx {
				pp.Selected++
			}
			return true
		case "k":
			if pp.Selected > 0 {
				pp.Selected--
			}
			return true
		}
	}
	return true
}

// SelectedProject returns the currently selected project path.
func (pp *ProjectPicker) SelectedProject() string {
	if pp.EnteringCustom || pp.Selected >= len(pp.Projects) {
		return pp.CustomInput
	}
	return pp.Projects[pp.Selected]
}

// View renders the project picker.
func (pp *ProjectPicker) View(width int) string {
	if !pp.Active {
		return ""
	}

	var b strings.Builder

	b.WriteString("Select target project:\n")

	for i, p := range pp.Projects {
		marker := "  "
		if i == pp.Selected && !pp.EnteringCustom {
			marker = "> "
		}
		suffix := ""
		if p == pp.OriginalPath {
			suffix = " (original)"
		}
		b.WriteString(fmt.Sprintf("%s%s%s\n", marker, p, suffix))
	}

	// Custom entry
	customMarker := "  "
	if pp.Selected >= len(pp.Projects) && !pp.EnteringCustom {
		customMarker = "> "
	}
	if pp.EnteringCustom {
		customMarker = "> "
	}
	b.WriteString(fmt.Sprintf("%s[Custom] %s\n", customMarker, pp.CustomInput))

	b.WriteString("\nTab: toggle custom  Enter: confirm  Esc: cancel")

	return b.String()
}

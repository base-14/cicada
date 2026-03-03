package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Filter is a simple text input that filters items.
// When active, shows a "/ " prompt with typed text.
type Filter struct {
	Active bool
	Query  string
}

// NewFilter creates a new inactive Filter.
func NewFilter() *Filter {
	return &Filter{}
}

// Update handles key events. Returns true if the key was consumed.
func (f *Filter) Update(msg tea.KeyMsg) (handled bool) {
	if !f.Active {
		// Only '/' activates
		if msg.Type == tea.KeyRunes && string(msg.Runes) == "/" {
			f.Active = true
			return true
		}
		return false
	}

	// Filter is active
	switch msg.Type {
	case tea.KeyEsc:
		f.Active = false
		f.Query = ""
		return true
	case tea.KeyEnter:
		f.Active = false
		return true
	case tea.KeyBackspace:
		if len(f.Query) > 0 {
			f.Query = f.Query[:len(f.Query)-1]
		}
		return true
	case tea.KeyRunes:
		f.Query += string(msg.Runes)
		return true
	}

	return true
}

// View returns "/ query" when active, "" when inactive.
func (f *Filter) View() string {
	if !f.Active {
		return ""
	}
	return "/ " + f.Query
}

// Matches returns true if text contains the query (case-insensitive substring match).
// An empty query matches everything.
func (f *Filter) Matches(text string) bool {
	if f.Query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(f.Query))
}

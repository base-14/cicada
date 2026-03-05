package components

import tea "github.com/charmbracelet/bubbletea"

// TextInput is a simple single-line text input component.
// When active, shows a prompt followed by typed text.
type TextInput struct {
	Active bool
	Value  string
	Prompt string
}

// NewTextInput creates a new inactive TextInput with the given prompt.
func NewTextInput(prompt string) *TextInput {
	return &TextInput{Prompt: prompt}
}

// Update handles key events. Returns true if the key was consumed.
func (ti *TextInput) Update(msg tea.KeyMsg) bool {
	if !ti.Active {
		return false
	}

	switch msg.Type {
	case tea.KeyEsc:
		ti.Active = false
		ti.Value = ""
		return true
	case tea.KeyEnter:
		ti.Active = false // Value preserved for caller to read
		return true
	case tea.KeyBackspace:
		if len(ti.Value) > 0 {
			ti.Value = ti.Value[:len(ti.Value)-1]
		}
		return true
	case tea.KeyRunes:
		ti.Value += string(msg.Runes)
		return true
	}

	return true
}

// View renders the text input.
func (ti *TextInput) View() string {
	if !ti.Active {
		return ""
	}
	return ti.Prompt + ti.Value
}

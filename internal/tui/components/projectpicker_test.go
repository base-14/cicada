package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewProjectPicker(t *testing.T) {
	projects := []string{"/home/user/project1", "/home/user/project2"}
	pp := NewProjectPicker(projects, "/home/user/project1")
	if pp == nil {
		t.Fatal("expected non-nil project picker")
	}
	if pp.Active {
		t.Error("expected picker to start inactive")
	}
	if pp.Selected != 0 {
		t.Error("expected Selected to be 0")
	}
	if pp.OriginalPath != "/home/user/project1" {
		t.Errorf("expected OriginalPath '/home/user/project1', got %q", pp.OriginalPath)
	}
}

func TestProjectPickerNavigation(t *testing.T) {
	projects := []string{"/p1", "/p2", "/p3"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true

	// Down moves selection forward
	pp.Update(tea.KeyMsg{Type: tea.KeyDown})
	if pp.Selected != 1 {
		t.Errorf("expected Selected 1 after down, got %d", pp.Selected)
	}

	pp.Update(tea.KeyMsg{Type: tea.KeyDown})
	if pp.Selected != 2 {
		t.Errorf("expected Selected 2 after second down, got %d", pp.Selected)
	}

	// Down past projects goes to custom entry (index == len(projects))
	pp.Update(tea.KeyMsg{Type: tea.KeyDown})
	if pp.Selected != 3 {
		t.Errorf("expected Selected 3 (custom), got %d", pp.Selected)
	}

	// Down at bottom stays at bottom
	pp.Update(tea.KeyMsg{Type: tea.KeyDown})
	if pp.Selected != 3 {
		t.Errorf("expected Selected 3 (clamped), got %d", pp.Selected)
	}

	// Up moves back
	pp.Update(tea.KeyMsg{Type: tea.KeyUp})
	if pp.Selected != 2 {
		t.Errorf("expected Selected 2 after up, got %d", pp.Selected)
	}

	// j/k navigation
	pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if pp.Selected != 3 {
		t.Errorf("expected Selected 3 after j, got %d", pp.Selected)
	}

	pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if pp.Selected != 2 {
		t.Errorf("expected Selected 2 after k, got %d", pp.Selected)
	}

	// Up at top stays at top
	pp.Selected = 0
	pp.Update(tea.KeyMsg{Type: tea.KeyUp})
	if pp.Selected != 0 {
		t.Errorf("expected Selected 0 (clamped), got %d", pp.Selected)
	}
}

func TestProjectPickerSelectsProject(t *testing.T) {
	projects := []string{"/p1", "/p2", "/p3"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true

	if pp.SelectedProject() != "/p1" {
		t.Errorf("expected '/p1', got %q", pp.SelectedProject())
	}

	pp.Selected = 2
	if pp.SelectedProject() != "/p3" {
		t.Errorf("expected '/p3', got %q", pp.SelectedProject())
	}
}

func TestProjectPickerCustomInput(t *testing.T) {
	projects := []string{"/p1"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true
	pp.EnteringCustom = true
	pp.CustomInput = "/my/custom/path"

	if pp.SelectedProject() != "/my/custom/path" {
		t.Errorf("expected '/my/custom/path', got %q", pp.SelectedProject())
	}
}

func TestProjectPickerTabTogglesCustom(t *testing.T) {
	projects := []string{"/p1", "/p2"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true

	if pp.EnteringCustom {
		t.Error("expected EnteringCustom to start false")
	}

	pp.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !pp.EnteringCustom {
		t.Error("expected EnteringCustom true after Tab")
	}

	pp.Update(tea.KeyMsg{Type: tea.KeyTab})
	if pp.EnteringCustom {
		t.Error("expected EnteringCustom false after second Tab")
	}
}

func TestProjectPickerCustomTyping(t *testing.T) {
	projects := []string{"/p1"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true
	pp.EnteringCustom = true

	pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	if pp.CustomInput != "/tmp" {
		t.Errorf("expected '/tmp', got %q", pp.CustomInput)
	}

	// Backspace
	pp.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if pp.CustomInput != "/tm" {
		t.Errorf("expected '/tm' after backspace, got %q", pp.CustomInput)
	}

	// Backspace on empty
	pp.CustomInput = ""
	pp.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if pp.CustomInput != "" {
		t.Errorf("expected empty after backspace on empty, got %q", pp.CustomInput)
	}
}

func TestProjectPickerEscCancels(t *testing.T) {
	projects := []string{"/p1"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true

	handled := pp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected Esc to be handled")
	}
	if pp.Active {
		t.Error("expected picker to be inactive after Esc")
	}
}

func TestProjectPickerEscCancelsCustom(t *testing.T) {
	projects := []string{"/p1"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true
	pp.EnteringCustom = true

	handled := pp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected Esc to be handled")
	}
	if pp.EnteringCustom {
		t.Error("expected EnteringCustom false after Esc in custom mode")
	}
}

func TestProjectPickerNotHandledWhenInactive(t *testing.T) {
	pp := NewProjectPicker([]string{"/p1"}, "")
	handled := pp.Update(tea.KeyMsg{Type: tea.KeyDown})
	if handled {
		t.Error("expected key not handled when inactive")
	}
}

func TestProjectPickerViewHighlightsOriginal(t *testing.T) {
	projects := []string{"/p1", "/p2", "/p3"}
	pp := NewProjectPicker(projects, "/p2")
	pp.Active = true

	view := pp.View(80)
	if !strings.Contains(view, "(original)") {
		t.Error("expected view to contain '(original)' marker")
	}
	if !strings.Contains(view, "/p2") {
		t.Error("expected view to contain original path '/p2'")
	}
	if !strings.Contains(view, "Select target project:") {
		t.Error("expected view to contain header")
	}
	if !strings.Contains(view, "[Custom]") {
		t.Error("expected view to contain '[Custom]' entry")
	}
}

func TestProjectPickerEnterConfirms(t *testing.T) {
	projects := []string{"/p1", "/p2"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true
	pp.Selected = 1

	handled := pp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected Enter to be handled")
	}
	if pp.Active {
		t.Error("expected picker to be inactive after Enter")
	}
	if pp.SelectedProject() != "/p2" {
		t.Errorf("expected '/p2', got %q", pp.SelectedProject())
	}
}

func TestProjectPickerCustomSelectedWhenBeyondList(t *testing.T) {
	projects := []string{"/p1"}
	pp := NewProjectPicker(projects, "")
	pp.Active = true
	pp.Selected = len(projects) // beyond list = custom
	pp.CustomInput = "/custom"

	if pp.SelectedProject() != "/custom" {
		t.Errorf("expected '/custom', got %q", pp.SelectedProject())
	}
}

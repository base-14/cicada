package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewFilter(t *testing.T) {
	f := NewFilter()
	if f == nil {
		t.Fatal("expected non-nil filter")
	}
	if f.Active {
		t.Error("expected filter to start inactive")
	}
	if f.Query != "" {
		t.Error("expected empty query")
	}
}

func TestFilter_ActivateWithSlash(t *testing.T) {
	f := NewFilter()
	handled := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !handled {
		t.Error("expected '/' to be handled")
	}
	if !f.Active {
		t.Error("expected filter to be active after '/'")
	}
}

func TestFilter_DeactivateWithEsc(t *testing.T) {
	f := NewFilter()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	handled := f.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected Esc to be handled when active")
	}
	if f.Active {
		t.Error("expected filter to be inactive after Esc")
	}
	if f.Query != "" {
		t.Error("expected query to be cleared after Esc")
	}
}

func TestFilter_EscNotHandledWhenInactive(t *testing.T) {
	f := NewFilter()
	handled := f.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected Esc not to be handled when inactive")
	}
}

func TestFilter_TypingAppendsToQuery(t *testing.T) {
	f := NewFilter()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if f.Query != "hello" {
		t.Errorf("expected query 'hello', got %q", f.Query)
	}
}

func TestFilter_Backspace(t *testing.T) {
	f := NewFilter()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	f.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if f.Query != "a" {
		t.Errorf("expected query 'a' after backspace, got %q", f.Query)
	}
}

func TestFilter_BackspaceOnEmpty(t *testing.T) {
	f := NewFilter()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	f.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if f.Query != "" {
		t.Errorf("expected empty query, got %q", f.Query)
	}
}

func TestFilter_EnterDeactivatesButKeepsQuery(t *testing.T) {
	f := NewFilter()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	handled := f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected Enter to be handled when active")
	}
	if f.Active {
		t.Error("expected filter to be inactive after Enter")
	}
	if f.Query != "foo" {
		t.Errorf("expected query 'foo' after Enter, got %q", f.Query)
	}
}

func TestFilter_Matches_CaseInsensitive(t *testing.T) {
	f := NewFilter()
	f.Query = "hello"

	if !f.Matches("Hello World") {
		t.Error("expected case-insensitive match")
	}
	if !f.Matches("HELLO") {
		t.Error("expected case-insensitive match for uppercase")
	}
	if f.Matches("world") {
		t.Error("expected no match for 'world'")
	}
}

func TestFilter_Matches_EmptyQuery(t *testing.T) {
	f := NewFilter()
	if !f.Matches("anything") {
		t.Error("expected empty query to match everything")
	}
}

func TestFilter_View_Active(t *testing.T) {
	f := NewFilter()
	f.Active = true
	f.Query = "test"
	view := f.View()
	if view == "" {
		t.Error("expected non-empty view when active")
	}
	if view != "/ test" {
		t.Errorf("expected '/ test', got %q", view)
	}
}

func TestFilter_View_Inactive(t *testing.T) {
	f := NewFilter()
	view := f.View()
	if view != "" {
		t.Errorf("expected empty view when inactive, got %q", view)
	}
}

func TestFilter_RunesNotHandledWhenInactive(t *testing.T) {
	f := NewFilter()
	handled := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if handled {
		t.Error("expected rune 'a' not to be handled when inactive")
	}
}

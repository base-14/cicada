package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTextInputTyping(t *testing.T) {
	ti := NewTextInput("> ")
	ti.Active = true

	consumed := ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if !consumed {
		t.Fatal("expected key to be consumed")
	}
	consumed = ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if !consumed {
		t.Fatal("expected key to be consumed")
	}

	if ti.Value != "hi" {
		t.Fatalf("expected Value %q, got %q", "hi", ti.Value)
	}
}

func TestTextInputBackspace(t *testing.T) {
	ti := NewTextInput("> ")
	ti.Active = true
	ti.Value = "abc"

	consumed := ti.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if !consumed {
		t.Fatal("expected key to be consumed")
	}

	if ti.Value != "ab" {
		t.Fatalf("expected Value %q, got %q", "ab", ti.Value)
	}
}

func TestTextInputEscDeactivates(t *testing.T) {
	ti := NewTextInput("> ")
	ti.Active = true
	ti.Value = "some text"

	consumed := ti.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !consumed {
		t.Fatal("expected key to be consumed")
	}

	if ti.Active {
		t.Fatal("expected Active to be false after Esc")
	}
	if ti.Value != "" {
		t.Fatalf("expected Value to be cleared, got %q", ti.Value)
	}
}

func TestTextInputEnterSignalsComplete(t *testing.T) {
	ti := NewTextInput("> ")
	ti.Active = true
	ti.Value = "hello"

	consumed := ti.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !consumed {
		t.Fatal("expected key to be consumed")
	}

	if ti.Active {
		t.Fatal("expected Active to be false after Enter")
	}
	if ti.Value != "hello" {
		t.Fatalf("expected Value to be preserved, got %q", ti.Value)
	}
}

func TestTextInputView(t *testing.T) {
	ti := NewTextInput("search: ")
	ti.Active = true
	ti.Value = "foo"

	got := ti.View()
	want := "search: foo"
	if got != want {
		t.Fatalf("expected View %q, got %q", want, got)
	}
}

func TestTextInputInactiveIgnoresKeys(t *testing.T) {
	ti := NewTextInput("> ")
	// Active is false by default

	consumed := ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if consumed {
		t.Fatal("expected inactive input to not consume keys")
	}

	if ti.Value != "" {
		t.Fatalf("expected Value to remain empty, got %q", ti.Value)
	}
}

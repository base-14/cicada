package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/store"
)

func TestApp_InitialView(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	if app.activeTab != 0 {
		t.Errorf("expected initial tab 0, got %d", app.activeTab)
	}
}

func TestApp_TabNavigation(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	// Press Tab to move to next
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updated.(App)
	if app.activeTab != 1 {
		t.Errorf("expected tab 1 after Tab, got %d", app.activeTab)
	}

	// Press Shift+Tab to go back
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = updated.(App)
	if app.activeTab != 0 {
		t.Errorf("expected tab 0 after Shift+Tab, got %d", app.activeTab)
	}
}

func TestApp_TabWrapAround(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	// Shift+Tab from first tab should wrap to last
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = updated.(App)
	if app.activeTab != len(tabNames)-1 {
		t.Errorf("expected last tab, got %d", app.activeTab)
	}
}

func TestApp_NumberKeyNavigation(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	// Press '3' to go to Sessions tab (index 2)
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	app = updated.(App)
	if app.activeTab != 2 {
		t.Errorf("expected tab 2, got %d", app.activeTab)
	}
}

func TestApp_QuitKey(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestApp_ScanProgress(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	msg := ScanBatchMsg{Scanned: 50, Total: 100}
	updated, _ := app.Update(msg)
	app = updated.(App)

	if app.scanScanned != 50 || app.scanTotal != 100 {
		t.Errorf("expected 50/100, got %d/%d", app.scanScanned, app.scanTotal)
	}
}

func TestApp_ScanComplete(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	msg := ScanCompleteMsg{}
	updated, _ := app.Update(msg)
	app = updated.(App)

	if !app.scanDone {
		t.Error("expected scanDone to be true")
	}
}

package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/base-14/cicada/internal/model"
	"github.com/base-14/cicada/internal/store"
)

func TestApp_InitialView(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	if app.activeTab != 0 {
		t.Errorf("expected initial tab 0, got %d", app.activeTab)
	}
}

func TestApp_TabNavigation(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

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
	app := NewApp(s, "")

	// Shift+Tab from first tab should wrap to last
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = updated.(App)
	if app.activeTab != len(tabNames)-1 {
		t.Errorf("expected last tab, got %d", app.activeTab)
	}
}

func TestApp_NumberKeyNavigation(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	// Press '3' to go to Sessions tab (index 2)
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	app = updated.(App)
	if app.activeTab != 2 {
		t.Errorf("expected tab 2, got %d", app.activeTab)
	}
}

func TestApp_QuitKey(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestApp_HelpOverlayToggle(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	// Press '?' to show help
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	app = updated.(App)

	if !app.showingHelp {
		t.Error("expected showingHelp to be true after '?'")
	}
}

func TestApp_HelpOverlayDismiss(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	// Show help
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	app = updated.(App)

	// Any key dismisses it
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	app = updated.(App)

	if app.showingHelp {
		t.Error("expected showingHelp to be false after pressing a key")
	}
}

func TestApp_HelpOverlayBlocksOtherKeys(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	// Show help
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	app = updated.(App)

	// 'q' should dismiss help, not quit
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Error("expected 'q' to dismiss help, not quit")
	}
}

func TestApp_HelpOverlayRendersContent(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	app.width = 80
	app.height = 40
	app.showingHelp = true

	view := app.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	// Check that help content is rendered
	if !strings.Contains(view, "Navigation") {
		t.Error("expected help overlay to contain 'Navigation'")
	}
	if !strings.Contains(view, "cicada") {
		t.Error("expected help overlay to contain 'cicada'")
	}
}

func TestApp_ScanProgress(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	msg := ScanBatchMsg{Scanned: 50, Total: 100}
	updated, _ := app.Update(msg)
	app = updated.(App)

	if app.scanScanned != 50 || app.scanTotal != 100 {
		t.Errorf("expected 50/100, got %d/%d", app.scanScanned, app.scanTotal)
	}
}

func TestApp_SlashActivatesFilterOnSessions(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	// Switch to sessions tab
	app.activeTab = 2

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = updated.(App)

	if !app.sessionsView.FilterActive() {
		t.Error("expected sessions filter to be active after '/'")
	}
}

func TestApp_SlashActivatesFilterOnProjects(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	// Switch to projects tab
	app.activeTab = 1

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = updated.(App)

	if !app.projectsView.FilterActive() {
		t.Error("expected projects filter to be active after '/'")
	}
}

func TestApp_FilterActiveBlocksQuit(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	app.activeTab = 2

	// Activate filter
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = updated.(App)

	// Press 'q' should NOT quit
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Error("expected 'q' not to quit when filter is active")
	}
}

func TestApp_ScanComplete(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	msg := ScanCompleteMsg{}
	updated, _ := app.Update(msg)
	app = updated.(App)

	if !app.scanDone {
		t.Error("expected scanDone to be true")
	}
}

func TestApp_ProjectDetailDrillIn(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-proj1",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})

	app := NewApp(s, "/tmp/test")
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = updated.(App)
	app.activeTab = 1
	// Trigger View to populate lastRows
	app.View()
	// Press Enter to open project detail
	result, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = result.(App)

	if !app.showingProjectDetail {
		t.Error("expected showingProjectDetail to be true after Enter on projects tab")
	}
	if app.projectDetailView == nil {
		t.Error("expected projectDetailView to be non-nil")
	}
}

func TestApp_ProjectDetailEscReturns(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-proj1",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})

	app := NewApp(s, "/tmp/test")
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = updated.(App)
	app.activeTab = 1
	app.View()
	result, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = result.(App)

	// Press Esc to close
	result, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = result.(App)

	if app.showingProjectDetail {
		t.Error("expected showingProjectDetail=false after Esc")
	}
	if app.projectDetailView != nil {
		t.Error("expected projectDetailView=nil after Esc")
	}
}

func TestApp_YankKeyCopiesResumeCommand(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "abc-123-def", Slug: "test-session", ProjectPath: "-Users-r-work-proj1",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})

	app := NewApp(s, "/tmp/test")
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = updated.(App)
	app.activeTab = 2
	// Trigger View to populate rows
	app.View()

	// Press 'y' to copy resume command
	result, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	app = result.(App)

	if cmd == nil {
		t.Error("expected a command from 'y' key press on sessions tab")
	}
}

func TestApp_YankKeyNoOpOnOtherTabs(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	app.activeTab = 0 // Analysis tab

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd != nil {
		t.Error("expected no command from 'y' key on non-sessions tab")
	}
}

func TestApp_CopyResultMsgSetsNotification(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	// Simulate successful copy
	result, cmd := app.Update(CopyResultMsg{Err: nil})
	app = result.(App)

	if app.notification != "Copied!" {
		t.Errorf("expected notification 'Copied!', got %q", app.notification)
	}
	if cmd == nil {
		t.Error("expected a tick command to clear notification")
	}
}

func TestApp_CopyResultMsgErrorSetsNotification(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")

	result, _ := app.Update(CopyResultMsg{Err: fmt.Errorf("no clipboard")})
	app = result.(App)

	if app.notification != "Copy failed" {
		t.Errorf("expected notification 'Copy failed', got %q", app.notification)
	}
}

func TestApp_ClearNotificationMsg(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	app.notification = "Copied!"

	result, _ := app.Update(clearNotificationMsg{})
	app = result.(App)

	if app.notification != "" {
		t.Errorf("expected empty notification, got %q", app.notification)
	}
}

func TestApp_NotificationShowsInStatusBar(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	app.width = 80
	app.height = 30
	app.notification = "Copied!"

	view := app.renderStatusBar()
	if !strings.Contains(view, "Copied!") {
		t.Error("expected status bar to contain 'Copied!'")
	}
}

func TestApp_HelpOverlayContainsYankKey(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	app.width = 80
	app.height = 40
	app.showingHelp = true

	view := app.View()
	if !strings.Contains(view, "claude --resume") {
		t.Error("expected help overlay to contain 'claude --resume'")
	}
}

func TestApp_StatusBarShowsCopyHintOnSessionsTab(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	app.width = 80
	app.height = 30
	app.activeTab = 2

	bar := app.renderStatusBar()
	if !strings.Contains(bar, "y copy") {
		t.Error("expected status bar to contain 'y copy' on sessions tab")
	}
}

func TestApp_StatusBarNoCopyHintOnOtherTabs(t *testing.T) {
	s := store.New()
	app := NewApp(s, "")
	app.width = 80
	app.height = 30
	app.activeTab = 0

	bar := app.renderStatusBar()
	if strings.Contains(bar, "y copy") {
		t.Error("expected status bar NOT to contain 'y copy' on analysis tab")
	}
}

func TestApp_YankKeyIgnoredWhenFilterActive(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "abc-123", Slug: "test", ProjectPath: "-Users-r-work-proj1",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})

	app := NewApp(s, "/tmp/test")
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = updated.(App)
	app.activeTab = 2

	// Activate filter
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = updated.(App)

	// Press 'y' — should NOT trigger copy
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd != nil {
		t.Error("expected 'y' to be forwarded to filter, not trigger copy")
	}
}

package views

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/base-14/cicada/internal/model"
	"github.com/base-14/cicada/internal/store"
)

func TestNewSessionsView(t *testing.T) {
	s := store.New()
	view := NewSessionsView(s)
	if view == nil {
		t.Fatal("expected non-nil sessions view")
	}
	if view.selected != 0 {
		t.Errorf("expected selected=0, got %d", view.selected)
	}
}

func TestSessionsView_Render(t *testing.T) {
	s := store.New()
	now := time.Now()

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "happy-cat", ProjectPath: "-Users-r-work-myproject",
		StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour),
		Duration: time.Hour,
		TokensIn: 1000, TokensOut: 500,
		Models: map[string]int{}, ToolUsage: map[string]int{"Read": 3, "Edit": 2},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 10,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "cool-fox", ProjectPath: "-Users-r-work-other",
		StartTime: now.Add(-time.Hour), EndTime: now,
		Duration: time.Hour,
		TokensIn: 2000, TokensOut: 1000,
		Models: map[string]int{}, ToolUsage: map[string]int{"Bash": 5},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 8,
	})

	view := NewSessionsView(s)
	content := view.View(100, 24)

	if content == "" {
		t.Error("expected non-empty view")
	}

	// Should contain column headers
	if !strings.Contains(content, "Slug") {
		t.Error("expected 'Slug' column header")
	}
	if !strings.Contains(content, "Project") {
		t.Error("expected 'Project' column header")
	}

	// Should contain session data
	if !strings.Contains(content, "happy-cat") {
		t.Error("expected 'happy-cat' slug in output")
	}
	if !strings.Contains(content, "cool-fox") {
		t.Error("expected 'cool-fox' slug in output")
	}

	// First row (newest) should be cool-fox since it started later
	lines := strings.Split(content, "\n")
	foundCoolFoxFirst := false
	foundHappyCat := false
	for _, line := range lines {
		if strings.Contains(line, "cool-fox") && !foundHappyCat {
			foundCoolFoxFirst = true
		}
		if strings.Contains(line, "happy-cat") {
			foundHappyCat = true
		}
	}
	if !foundCoolFoxFirst {
		t.Error("expected cool-fox (newest) to appear before happy-cat")
	}

	// First row should have selection indicator
	if !strings.Contains(content, ">") {
		t.Error("expected '>' selection indicator")
	}
}

func TestSessionsView_Empty(t *testing.T) {
	s := store.New()
	view := NewSessionsView(s)
	content := view.View(80, 24)

	if content == "" {
		t.Error("expected non-empty view even when empty")
	}
	if !strings.Contains(content, "No sessions") {
		t.Error("expected 'No sessions' message")
	}
}

func TestSessionsView_FilterBySlug(t *testing.T) {
	s := store.New()
	now := time.Now()

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "happy-cat", ProjectPath: "-Users-r-work-myproject",
		StartTime: now.Add(-2 * time.Hour), Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 10,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "cool-fox", ProjectPath: "-Users-r-work-other",
		StartTime: now.Add(-time.Hour), Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 8,
	})

	view := NewSessionsView(s)
	// Activate filter and type "happy"
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	content := view.View(100, 24)
	if !strings.Contains(content, "happy-cat") {
		t.Error("expected 'happy-cat' to be visible")
	}
	if strings.Contains(content, "cool-fox") {
		t.Error("expected 'cool-fox' to be filtered out")
	}
}

func TestSessionsView_FilterByProject(t *testing.T) {
	s := store.New()
	now := time.Now()

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-myproject",
		StartTime: now, Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 10,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "s2", ProjectPath: "-Users-r-work-other",
		StartTime: now.Add(time.Hour), Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 8,
	})

	view := NewSessionsView(s)
	// Filter by project path
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "myproject" {
		view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	content := view.View(100, 24)
	if !strings.Contains(content, "s1") {
		t.Error("expected 's1' to be visible")
	}
	if strings.Contains(content, "s2") {
		t.Error("expected 's2' to be filtered out")
	}
}

func TestSessionsView_FilterShowsBar(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-proj",
		StartTime: now, Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})

	view := NewSessionsView(s)
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "test" {
		view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	content := view.View(100, 24)
	if !strings.Contains(content, "/ test") {
		t.Error("expected filter bar '/ test' in view")
	}
}

func TestSessionsView_SelectedSession(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "happy-cat", ProjectPath: "-Users-r-work-myproject",
		StartTime: now, Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{},
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "cool-fox", ProjectPath: "-Users-r-work-other",
		StartTime: now.Add(time.Hour), Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{},
	})

	view := NewSessionsView(s)
	selected := view.SelectedSession()
	if selected == nil {
		t.Fatal("expected non-nil selected session")
	}
	// Newest first, so cool-fox at index 0
	if selected.Slug != "cool-fox" {
		t.Errorf("expected selected session 'cool-fox', got %q", selected.Slug)
	}
}

func TestVisibleSessions(t *testing.T) {
	s := store.New()
	s.Add(&model.SessionMeta{
		UUID: "a", Slug: "alpha", ProjectPath: "-p",
		StartTime: time.Now(),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{},
	})
	s.Add(&model.SessionMeta{
		UUID: "b", Slug: "beta", ProjectPath: "-p",
		StartTime: time.Now().Add(time.Hour),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{},
	})

	v := NewSessionsView(s)
	rows := v.VisibleSessions()
	if len(rows) != 2 {
		t.Errorf("VisibleSessions() = %d rows, want 2", len(rows))
	}
	// Should be sorted newest first
	if rows[0].Slug != "beta" {
		t.Errorf("expected first visible session 'beta', got %q", rows[0].Slug)
	}
}

// TestSessionsView_Scrolling verifies that when there are more sessions than can fit
// on screen, the view implements a scrolling window that follows the selected item.
func TestSessionsView_Scrolling(t *testing.T) {
	s := store.New()
	now := time.Now()

	// Add 50 sessions to test scrolling
	for i := range 50 {
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("session-%d", i),
			ProjectPath:  "-p",
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
		})
	}

	view := NewSessionsView(s)
	// Small height to force scrolling (only 5 visible rows)
	content := view.View(100, 10)

	// Should show first 5 sessions (newest first)
	lines := strings.Split(content, "\n")
	dataLines := 0
	for _, line := range lines {
		if strings.Contains(line, ">") || (strings.HasPrefix(line, "  ") && len(strings.TrimSpace(line)) > 0 && !strings.Contains(line, "Slug") && !strings.Contains(line, "\u2500")) {
			dataLines++
		}
	}

	if dataLines > 5 {
		t.Errorf("expected at most 5 visible rows, got %d", dataLines)
	}

	// Navigate down 10 times
	for range 10 {
		view.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Should now show sessions around index 10
	content = view.View(100, 10)

	// The selected session should be visible
	selected := view.SelectedSession()
	if selected == nil {
		t.Fatal("expected non-nil selected session")
	}
	if !strings.Contains(content, selected.Slug) {
		t.Errorf("expected selected session %q to be visible after scrolling", selected.Slug)
	}
}

// TestSessionsView_ScrollingVimKeys verifies that vim-style j/k keys work for
// scrolling through sessions and that the selected item remains visible.
func TestSessionsView_ScrollingVimKeys(t *testing.T) {
	s := store.New()
	now := time.Now()

	// Add 30 sessions
	for i := range 30 {
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("session-%d", i),
			ProjectPath:  "-p",
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
		})
	}

	view := NewSessionsView(s)

	// Navigate down with 'j' key
	for range 15 {
		view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	if view.selected != 15 {
		t.Errorf("expected selected=15 after 15 'j' presses, got %d", view.selected)
	}

	// Navigate up with 'k' key
	for range 5 {
		view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	}

	if view.selected != 10 {
		t.Errorf("expected selected=10 after 5 'k' presses, got %d", view.selected)
	}

	// Verify selected session is visible
	content := view.View(100, 10)
	selected := view.SelectedSession()
	if !strings.Contains(content, selected.Slug) {
		t.Errorf("expected selected session to be visible after vim key navigation")
	}
}

// TestSessionsView_ScrollingBounds verifies that scrolling is properly bounded
// at the top and bottom of the list, and that the selected item is always visible.
func TestSessionsView_ScrollingBounds(t *testing.T) {
	s := store.New()
	now := time.Now()

	// Add 20 sessions
	for i := range 20 {
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("session-%d", i),
			ProjectPath:  "-p",
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
		})
	}

	view := NewSessionsView(s)

	// Try to scroll up from the top
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.selected != 0 {
		t.Errorf("expected selected=0 when scrolling up from top, got %d", view.selected)
	}

	// Scroll to bottom
	for range 25 {
		view.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Should be clamped to last item (19)
	if view.selected != 19 {
		t.Errorf("expected selected=19 at bottom, got %d", view.selected)
	}

	// Verify last session is visible
	content := view.View(100, 10)
	selected := view.SelectedSession()
	if !strings.Contains(content, selected.Slug) {
		t.Errorf("expected last session to be visible when scrolled to bottom")
	}
}

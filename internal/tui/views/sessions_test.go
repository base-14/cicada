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

// TestSessionsView_GGJumpsToStart verifies that pressing 'g' twice (vim gg)
// jumps the selection to the first item in the list, regardless of current position.
func TestSessionsView_GGJumpsToStart(t *testing.T) {
	s := store.New()
	now := time.Now()
	for i := range 5 {
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

	v := NewSessionsView(s)
	v.View(100, 24) // populate rows

	// Navigate to the end
	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.selected != 2 {
		t.Fatalf("expected selected=2 after navigation, got %d", v.selected)
	}

	// Press 'g' twice to jump to start
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	if v.selected != 0 {
		t.Errorf("expected selected=0 after gg, got %d", v.selected)
	}
}

// TestSessionsView_GJumpsToEnd verifies that pressing 'G' (vim G)
// jumps the selection to the last item in the list.
func TestSessionsView_GJumpsToEnd(t *testing.T) {
	s := store.New()
	now := time.Now()
	for i := range 5 {
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

	v := NewSessionsView(s)
	v.View(100, 24) // populate rows

	// Press 'G' to jump to end
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})

	if v.selected != 4 {
		t.Errorf("expected selected=4 after G, got %d", v.selected)
	}
}

// TestSessionsView_GGAfterNavigation verifies that gg works correctly
// after navigating with other keys, resetting position to start.
func TestSessionsView_GGAfterNavigation(t *testing.T) {
	s := store.New()
	now := time.Now()
	for i := range 5 {
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

	v := NewSessionsView(s)
	v.View(100, 24)

	// Navigate down with j
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if v.selected != 3 {
		t.Fatalf("expected selected=3 after j navigation, got %d", v.selected)
	}

	// Press gg to jump back to start
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	if v.selected != 0 {
		t.Errorf("expected selected=0 after gg, got %d", v.selected)
	}
}

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

func TestProjectsView_Render(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-myproject",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "s2", ProjectPath: "-Users-r-work-myproject",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 3,
	})
	s.Add(&model.SessionMeta{
		UUID: "u3", Slug: "s3", ProjectPath: "-Users-r-work-other",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 1,
	})

	view := NewProjectsView(s)
	content := view.View(80, 24)

	if content == "" {
		t.Error("expected non-empty view")
	}
}

func TestProjectsView_Filter(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-myproject",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "s2", ProjectPath: "-Users-r-work-other",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 1,
	})

	view := NewProjectsView(s)
	// Activate filter and type "myproject"
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "myproject" {
		view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	content := view.View(80, 24)
	if !strings.Contains(content, "myproject") {
		t.Error("expected 'myproject' to be visible")
	}
	if strings.Contains(content, "/Users/r/work/other") {
		t.Error("expected 'other' project to be filtered out")
	}
}

func TestProjectsView_UpdateNavigation(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-proj1",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "s2", ProjectPath: "-Users-r-work-proj2",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 1,
	})

	view := NewProjectsView(s)
	if view.selected != 0 {
		t.Errorf("expected selected=0, got %d", view.selected)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	if view.selected != 1 {
		t.Errorf("expected selected=1 after down, got %d", view.selected)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.selected != 0 {
		t.Errorf("expected selected=0 after up, got %d", view.selected)
	}
}

func TestProjectsView_SelectedProject(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-proj1",
		StartTime: now.Add(-time.Hour), EndTime: now,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "s2", ProjectPath: "-Users-r-work-proj2",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 1,
	})

	view := NewProjectsView(s)
	view.View(80, 24)

	path := view.SelectedProject()
	if path == "" {
		t.Error("expected non-empty selected project path")
	}

	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	view.View(80, 24)
	path2 := view.SelectedProject()
	if path2 == "" {
		t.Error("expected non-empty selected project after navigation")
	}
	if path == path2 {
		t.Error("expected different project after navigating down")
	}
}

func TestProjectsView_SelectedProject_Empty(t *testing.T) {
	s := store.New()
	view := NewProjectsView(s)
	view.View(80, 24)
	path := view.SelectedProject()
	if path != "" {
		t.Errorf("expected empty path for empty store, got %q", path)
	}
}

func TestProjectsView_Empty(t *testing.T) {
	s := store.New()
	view := NewProjectsView(s)
	content := view.View(80, 24)

	if content == "" {
		t.Error("expected non-empty view even when empty")
	}
}

// TestProjectsView_GGJumpsToStart verifies that pressing 'g' twice (vim gg)
// jumps the selection to the first item in the list, regardless of current position.
func TestProjectsView_GGJumpsToStart(t *testing.T) {
	s := store.New()
	now := time.Now()
	for i := range 5 {
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("s%d", i),
			ProjectPath:  fmt.Sprintf("-p%d", i),
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			EndTime:      now.Add(time.Duration(i)*time.Hour + time.Minute),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
			MessageCount: 1,
		})
	}

	v := NewProjectsView(s)
	v.View(80, 24) // populate lastRows

	// Navigate down
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

// TestProjectsView_GJumpsToEnd verifies that pressing 'G' (vim G)
// jumps the selection to the last item in the list.
func TestProjectsView_GJumpsToEnd(t *testing.T) {
	s := store.New()
	now := time.Now()
	for i := range 5 {
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("s%d", i),
			ProjectPath:  fmt.Sprintf("-p%d", i),
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			EndTime:      now.Add(time.Duration(i)*time.Hour + time.Minute),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
			MessageCount: 1,
		})
	}

	v := NewProjectsView(s)
	v.View(80, 24) // populate lastRows

	// Press 'G' to jump to end
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})

	if v.selected != 4 {
		t.Errorf("expected selected=4 after G, got %d", v.selected)
	}
}

// TestProjectsView_GGAfterNavigation verifies that gg works correctly
// after navigating with other keys, resetting position to start.
func TestProjectsView_GGAfterNavigation(t *testing.T) {
	s := store.New()
	now := time.Now()
	for i := range 5 {
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("s%d", i),
			ProjectPath:  fmt.Sprintf("-p%d", i),
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			EndTime:      now.Add(time.Duration(i)*time.Hour + time.Minute),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
			MessageCount: 1,
		})
	}

	v := NewProjectsView(s)
	v.View(80, 24)

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

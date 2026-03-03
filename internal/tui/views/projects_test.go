package views

import (
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

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

// TestProjectsView_Scrolling verifies that when there are more projects than can fit
// on screen, the view implements a scrolling window that follows the selected item.
func TestProjectsView_Scrolling(t *testing.T) {
	s := store.New()
	now := time.Now()

	// Add 50 projects to test scrolling
	for i := range 50 {
		projectPath := fmt.Sprintf("project-%d", i)
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("session-%d", i),
			ProjectPath:  projectPath,
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
		})
	}

	view := NewProjectsView(s)
	// Small height to force scrolling (only 5 visible rows)
	content := view.View(100, 10)

	// Count visible project rows
	lines := strings.Split(content, "\n")
	dataLines := 0
	for _, line := range lines {
		if strings.Contains(line, ">") || (strings.HasPrefix(line, "  ") && len(strings.TrimSpace(line)) > 0 && !strings.Contains(line, "Project") && !strings.Contains(line, "\u2500")) {
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

	// Should now show projects around index 10
	content = view.View(100, 10)

	// The selected project should be visible
	view.View(100, 10) // Refresh lastRows
	selectedPath := view.SelectedProject()
	if selectedPath == "" {
		t.Fatal("expected non-empty selected project")
	}

	// Decode the project path for display
	decoded := "/" + strings.ReplaceAll(strings.TrimPrefix(selectedPath, "-"), "-", "/")
	if !strings.Contains(content, decoded) {
		t.Errorf("expected selected project to be visible after scrolling")
	}
}

// TestProjectsView_ScrollingVimKeys verifies that vim-style j/k keys work for
// scrolling through projects and that the selected item remains visible.
func TestProjectsView_ScrollingVimKeys(t *testing.T) {
	s := store.New()
	now := time.Now()

	// Add 30 projects
	for i := range 30 {
		projectPath := fmt.Sprintf("project-%d", i)
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("session-%d", i),
			ProjectPath:  projectPath,
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
		})
	}

	view := NewProjectsView(s)

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

	// Verify selected project is visible
	content := view.View(100, 10)
	selectedPath := view.SelectedProject()
	decoded := "/" + strings.ReplaceAll(strings.TrimPrefix(selectedPath, "-"), "-", "/")
	if !strings.Contains(content, decoded) {
		t.Errorf("expected selected project to be visible after vim key navigation")
	}
}

// TestProjectsView_ScrollingBounds verifies that scrolling is properly bounded
// at the top and bottom of the list, and that the selected item is always visible.
func TestProjectsView_ScrollingBounds(t *testing.T) {
	s := store.New()
	now := time.Now()

	// Add 20 projects
	for i := range 20 {
		projectPath := fmt.Sprintf("project-%d", i)
		s.Add(&model.SessionMeta{
			UUID:         fmt.Sprintf("u%d", i),
			Slug:         fmt.Sprintf("session-%d", i),
			ProjectPath:  projectPath,
			StartTime:    now.Add(time.Duration(i) * time.Hour),
			Models:       map[string]int{},
			ToolUsage:    map[string]int{},
			SkillsUsed:   map[string]int{},
			CommandsUsed: map[string]int{},
			FileOps:      map[string]int{},
		})
	}

	view := NewProjectsView(s)

	// Try to scroll up from the top
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.selected != 0 {
		t.Errorf("expected selected=0 when scrolling up from top, got %d", view.selected)
	}

	// Scroll to bottom
	for range 25 {
		view.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Render to populate lastRows and clamp selection
	view.View(100, 10)

	// Should be clamped to last item (19)
	if view.selected != 19 {
		t.Errorf("expected selected=19 at bottom, got %d", view.selected)
	}

	// Verify last project is visible
	content := view.View(100, 10)
	selectedPath := view.SelectedProject()
	decoded := "/" + strings.ReplaceAll(strings.TrimPrefix(selectedPath, "-"), "-", "/")
	if !strings.Contains(content, decoded) {
		t.Errorf("expected last project to be visible when scrolled to bottom")
	}
}

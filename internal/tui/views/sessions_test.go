package views

import (
	"strings"
	"testing"
	"time"

	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
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

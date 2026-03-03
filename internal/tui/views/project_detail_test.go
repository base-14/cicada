package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

func newTestProjectSessions() (*store.Store, string) {
	s := store.New()
	now := time.Now()
	project := "-Users-r-work-myproject"

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "fix-login", ProjectPath: project,
		StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour),
		Duration: time.Hour, InitialPrompt: "Fix the login bug",
		TokensIn: 10000, TokensOut: 5000, CacheRead: 1000, CacheWrite: 500,
		Models: map[string]int{"claude-opus-4-6": 5},
		ToolUsage: map[string]int{"Read": 10, "Edit": 5, "Bash": 3},
		SkillsUsed: map[string]int{"tdd": 2},
		CommandsUsed: map[string]int{}, FileOps: map[string]int{"read": 10, "edit": 5},
		GitBranches: []string{"main", "fix-login"}, SubagentCount: 1, MessageCount: 20,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "add-tests", ProjectPath: project,
		StartTime: now.Add(-time.Hour), EndTime: now,
		Duration: time.Hour, InitialPrompt: "Add unit tests",
		TokensIn: 8000, TokensOut: 4000, CacheRead: 800, CacheWrite: 400,
		Models: map[string]int{"claude-sonnet-4-6": 3},
		ToolUsage: map[string]int{"Read": 5, "Write": 8, "Bash": 6},
		SkillsUsed: map[string]int{"tdd": 1, "debugging": 3},
		CommandsUsed: map[string]int{}, FileOps: map[string]int{"read": 5, "write": 8},
		GitBranches: []string{"main"}, SubagentCount: 0, MessageCount: 15,
	})

	return s, project
}

func TestNewProjectDetailView(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)
	if view == nil {
		t.Fatal("expected non-nil project detail view")
	}
	if view.activeTab != 0 {
		t.Errorf("expected activeTab=0, got %d", view.activeTab)
	}
}

func TestProjectDetailView_OverviewTab(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)
	content := view.View(100, 30)

	if content == "" {
		t.Error("expected non-empty overview")
	}
	if !strings.Contains(content, "myproject") {
		t.Error("expected project name in overview")
	}
	if !strings.Contains(content, "2") {
		t.Error("expected session count in overview")
	}
	if !strings.Contains(content, "Overview") {
		t.Error("expected 'Overview' tab label")
	}
}

func TestProjectDetailView_TabNavigation(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	if view.activeTab != 1 {
		t.Errorf("expected activeTab=1, got %d", view.activeTab)
	}

	view.activeTab = 0
	view.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if view.activeTab != 4 {
		t.Errorf("expected activeTab=4, got %d", view.activeTab)
	}
}

func TestProjectDetailView_Scrolling(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	if view.scrollY != 1 {
		t.Errorf("expected scrollY=1, got %d", view.scrollY)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.scrollY != 0 {
		t.Errorf("expected scrollY=0, got %d", view.scrollY)
	}
}

func TestProjectDetailView_EmptySessions(t *testing.T) {
	view := NewProjectDetailView("-empty-project", nil)
	content := view.View(100, 30)
	if content == "" {
		t.Error("expected non-empty view for empty project")
	}
}

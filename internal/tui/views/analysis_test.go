package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

func TestNewAnalysisView(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)
	if view == nil {
		t.Fatal("expected non-nil analysis view")
	}
}

func TestAnalysisView_Empty(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)
	content := view.View(80, 24)
	if content == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(content, "No session data") {
		t.Error("expected empty state message")
	}
}

func TestAnalysisView_WithSessions(t *testing.T) {
	s := store.New()
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "session-1", ProjectPath: "-test",
		StartTime: now, EndTime: now.Add(time.Hour), Duration: time.Hour,
		TokensIn: 10000, TokensOut: 5000,
		Models:       map[string]int{"claude-opus-4-6": 5},
		ToolUsage:    map[string]int{"Read": 10, "Edit": 5, "Bash": 3},
		SkillsUsed:   map[string]int{},
		CommandsUsed: map[string]int{},
		FileOps:      map[string]int{},
		GitBranches:  []string{"main"},
		SubagentCount: 1, MessageCount: 20,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "session-2", ProjectPath: "-test",
		StartTime: yesterday, EndTime: yesterday.Add(30 * time.Minute), Duration: 30 * time.Minute,
		TokensIn: 5000, TokensOut: 2000,
		Models:       map[string]int{"claude-sonnet-4-6": 3},
		ToolUsage:    map[string]int{"Read": 5, "Write": 2},
		SkillsUsed:   map[string]int{},
		CommandsUsed: map[string]int{},
		FileOps:      map[string]int{},
		GitBranches:  []string{"main", "feat"},
		SubagentCount: 0, MessageCount: 10,
	})

	view := NewAnalysisView(s)
	content := view.View(100, 60)

	// Stats section
	if !strings.Contains(content, "Sessions") {
		t.Error("expected sessions count")
	}

	// Streaks
	if !strings.Contains(content, "Streak") {
		t.Error("expected streak section")
	}

	// Personal bests
	if !strings.Contains(content, "Longest Session") {
		t.Error("expected longest session")
	}

	// Favorite tool
	if !strings.Contains(content, "Read") {
		t.Error("expected favorite tool 'Read'")
	}

	// Heatmap
	if !strings.Contains(content, "Heatmap") {
		t.Error("expected heatmap")
	}

	// Tools chart
	if !strings.Contains(content, "Tools") {
		t.Error("expected tools section")
	}

	// Models
	if !strings.Contains(content, "Models") {
		t.Error("expected models section")
	}
}

func TestAnalysisView_Scrolling(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)

	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	if view.scrollY != 1 {
		t.Errorf("expected scrollY=1, got %d", view.scrollY)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.scrollY != 0 {
		t.Errorf("expected scrollY=0, got %d", view.scrollY)
	}
}

func TestAnalysisView_VimScrolling(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)

	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if view.scrollY != 1 {
		t.Errorf("expected scrollY=1 after 'j', got %d", view.scrollY)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if view.scrollY != 0 {
		t.Errorf("expected scrollY=0 after 'k', got %d", view.scrollY)
	}
}

func TestAnalysisView_ScrollBoundsAtZero(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)

	// scrollY should not go negative
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.scrollY != 0 {
		t.Errorf("expected scrollY=0 (no negative), got %d", view.scrollY)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if view.scrollY != 0 {
		t.Errorf("expected scrollY=0 (no negative via 'k'), got %d", view.scrollY)
	}
}

func TestBuildHeatmapFromSessions(t *testing.T) {
	sessionTime := time.Date(2026, 3, 2, 14, 30, 0, 0, time.Local) // Monday 14:30

	sessions := []*model.SessionMeta{
		{
			UUID: "u1", Slug: "test", ProjectPath: "-test",
			StartTime: sessionTime, Duration: time.Hour,
			Models: map[string]int{}, ToolUsage: map[string]int{"Read": 1},
			SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
			FileOps: map[string]int{}, MessageCount: 1,
		},
	}

	heatmap := buildHeatmapFromSessions(sessions)

	// Monday = index 0, hour 14
	if heatmap[0][14] != 1 {
		t.Errorf("expected heatmap[0][14]=1 (Monday 14:00), got %d", heatmap[0][14])
	}

	// All other cells should be 0
	for day := range 7 {
		for hour := range 24 {
			if day == 0 && hour == 14 {
				continue
			}
			if heatmap[day][hour] != 0 {
				t.Errorf("expected heatmap[%d][%d]=0, got %d", day, hour, heatmap[day][hour])
			}
		}
	}
}

func TestBuildHeatmapFromSessions_Sunday(t *testing.T) {
	// Sunday should map to index 6
	sundayTime := time.Date(2026, 3, 1, 10, 0, 0, 0, time.Local) // Sunday 10:00

	sessions := []*model.SessionMeta{
		{
			UUID: "u1", StartTime: sundayTime,
			Models: map[string]int{}, ToolUsage: map[string]int{},
			SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
			FileOps: map[string]int{},
		},
	}

	heatmap := buildHeatmapFromSessions(sessions)
	if heatmap[6][10] != 1 {
		t.Errorf("expected heatmap[6][10]=1 (Sunday 10:00), got %d", heatmap[6][10])
	}
}

func TestBuildHeatmapFromSessions_SkipsZeroTime(t *testing.T) {
	sessions := []*model.SessionMeta{
		{
			UUID: "u1", // StartTime is zero value
			Models: map[string]int{}, ToolUsage: map[string]int{},
			SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
			FileOps: map[string]int{},
		},
	}

	heatmap := buildHeatmapFromSessions(sessions)
	for day := range 7 {
		for hour := range 24 {
			if heatmap[day][hour] != 0 {
				t.Errorf("expected all zeros for zero time, got heatmap[%d][%d]=%d", day, hour, heatmap[day][hour])
			}
		}
	}
}

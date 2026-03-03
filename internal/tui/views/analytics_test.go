package views

import (
	"strings"
	"testing"
	"time"

	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

func TestNewAnalyticsView(t *testing.T) {
	s := store.New()
	view := NewAnalyticsView(s)
	if view == nil {
		t.Fatal("expected non-nil analytics view")
	}
}

func TestAnalyticsView_Empty(t *testing.T) {
	s := store.New()
	view := NewAnalyticsView(s)
	content := view.View(80, 24)
	if content == "" {
		t.Error("expected non-empty view even with no data")
	}
	if !strings.Contains(content, "No session data") {
		t.Error("expected 'No session data' message for empty store")
	}
}

func TestAnalyticsView_WithSessions(t *testing.T) {
	s := store.New()
	now := time.Now()

	// Monday 9am
	monday := now.Truncate(24*time.Hour).AddDate(0, 0, -int(now.Weekday())+1).Add(9 * time.Hour)
	// Friday 14:00
	friday := monday.AddDate(0, 0, 4).Add(5 * time.Hour)

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "session-1", ProjectPath: "-Users-r-proj",
		StartTime: monday, Duration: time.Hour,
		TokensIn: 1000, TokensOut: 500,
		Models:    map[string]int{"claude-sonnet-4-20250514": 5},
		ToolUsage: map[string]int{"Read": 10, "Edit": 5, "Bash": 3},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, SubagentCount: 2, MessageCount: 10,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "session-2", ProjectPath: "-Users-r-proj",
		StartTime: friday, Duration: 30 * time.Minute,
		TokensIn: 2000, TokensOut: 1000,
		Models:    map[string]int{"claude-opus-4-20250514": 3},
		ToolUsage: map[string]int{"Read": 5, "Write": 2, "mcp__playwright__click": 4},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, SubagentCount: 0, MessageCount: 8,
	})

	view := NewAnalyticsView(s)
	content := view.View(100, 40)

	// Should contain time period label
	if !strings.Contains(content, "All Time") {
		t.Error("expected 'All Time' period selector")
	}

	// Should contain sparkline section
	if !strings.Contains(content, "Sessions") {
		t.Error("expected 'Sessions' sparkline section")
	}

	// Should contain heatmap section
	if !strings.Contains(content, "Activity Heatmap") {
		t.Error("expected 'Activity Heatmap' section")
	}

	// Should contain tool usage section
	if !strings.Contains(content, "Top Tools") {
		t.Error("expected 'Top Tools' section")
	}

	// Should contain model distribution
	if !strings.Contains(content, "Models") {
		t.Error("expected 'Models' section")
	}

	// Should contain work mode distribution
	if !strings.Contains(content, "Work Mode") {
		t.Error("expected 'Work Mode' section")
	}
}

func TestAnalyticsView_HeatmapData(t *testing.T) {
	s := store.New()

	// Create a session on a known day/hour
	// time.Monday = 1, so we pick a specific date
	sessionTime := time.Date(2026, 3, 2, 14, 30, 0, 0, time.Local) // Monday 14:30

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "test", ProjectPath: "-test",
		StartTime: sessionTime, Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{"Read": 1},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 1,
	})

	view := NewAnalyticsView(s)
	heatmap := view.buildHeatmap()

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

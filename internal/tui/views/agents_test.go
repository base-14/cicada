package views

import (
	"strings"
	"testing"
	"time"

	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

func TestNewAgentsView(t *testing.T) {
	s := store.New()
	view := NewAgentsView(s)
	if view == nil {
		t.Fatal("expected non-nil agents view")
	}
}

func TestAgentsView_Empty(t *testing.T) {
	s := store.New()
	view := NewAgentsView(s)
	content := view.View(80, 24)
	if content == "" {
		t.Error("expected non-empty view even with no data")
	}
	if !strings.Contains(content, "No session data") {
		t.Error("expected 'No session data' message for empty store")
	}
}

func TestAgentsView_WithSubagents(t *testing.T) {
	s := store.New()
	now := time.Now()

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "session-1", ProjectPath: "-Users-r-proj",
		StartTime: now, Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{"Read": 3, "Agent": 5},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, SubagentCount: 5, MessageCount: 10,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "session-2", ProjectPath: "-Users-r-proj",
		StartTime: now.Add(-time.Hour), Duration: 30 * time.Minute,
		Models: map[string]int{}, ToolUsage: map[string]int{"Read": 2},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, SubagentCount: 0, MessageCount: 5,
	})
	s.Add(&model.SessionMeta{
		UUID: "u3", Slug: "session-3", ProjectPath: "-Users-r-other",
		StartTime: now.Add(-2 * time.Hour), Duration: 45 * time.Minute,
		Models: map[string]int{}, ToolUsage: map[string]int{"Agent": 3},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, SubagentCount: 3, MessageCount: 7,
	})

	view := NewAgentsView(s)
	content := view.View(80, 24)

	// Should show total subagent invocations (5 + 0 + 3 = 8)
	if !strings.Contains(content, "8") {
		t.Error("expected total subagent count of 8")
	}

	// Should show sessions that used subagents (2 of 3)
	if !strings.Contains(content, "2") {
		t.Error("expected 2 sessions using subagents")
	}

	// Should contain summary section
	if !strings.Contains(content, "Subagent") {
		t.Error("expected 'Subagent' in output")
	}
}

func TestAgentsView_NoSubagents(t *testing.T) {
	s := store.New()
	now := time.Now()

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "session-1", ProjectPath: "-Users-r-proj",
		StartTime: now, Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{"Read": 3},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, SubagentCount: 0, MessageCount: 10,
	})

	view := NewAgentsView(s)
	content := view.View(80, 24)

	// Should still render, showing 0 usage
	if content == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(content, "0") {
		t.Error("expected zero count somewhere in output")
	}
}

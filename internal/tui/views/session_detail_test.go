package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

func newTestDetail() (*model.SessionMeta, *model.SessionDetail) {
	now := time.Now()
	meta := &model.SessionMeta{
		UUID: "test-uuid", Slug: "happy-cat",
		ProjectPath:   "-Users-r-work-myproject",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now,
		Duration:      time.Hour,
		InitialPrompt: "Fix the login bug please",
		TokensIn:      10000, TokensOut: 5000,
		Models:        map[string]int{"claude-opus-4-6": 5, "claude-sonnet-4-6": 2},
		ToolUsage:     map[string]int{"Read": 10, "Edit": 5, "Bash": 3},
		SkillsUsed:    map[string]int{},
		CommandsUsed:  map[string]int{},
		FileOps:       map[string]int{"read": 10, "edit": 5},
		MessageCount:  20,
		GitBranches:   []string{"main", "feature"},
	}

	detail := &model.SessionDetail{
		Meta: meta,
		ChatMessages: []model.ChatMessage{
			{Role: "user", Content: "Fix the login bug please", Timestamp: now.Add(-time.Hour)},
			{Role: "assistant", Content: "I'll look at the login code and fix it.", Timestamp: now.Add(-55 * time.Minute)},
			{Role: "tool", Content: "Read → /work/login.go", ToolName: "Read", Timestamp: now.Add(-55 * time.Minute)},
			{Role: "tool", Content: "Edit → /work/login.go", ToolName: "Edit", Timestamp: now.Add(-50 * time.Minute)},
			{Role: "assistant", Content: "I've fixed the login bug.", Timestamp: now.Add(-50 * time.Minute)},
			{Role: "user", Content: "Run the tests", Timestamp: now.Add(-45 * time.Minute)},
			{Role: "tool", Content: "Bash", ToolName: "Bash", Timestamp: now.Add(-40 * time.Minute)},
		},
		Timeline: []model.TimelineEvent{
			{Timestamp: now.Add(-time.Hour), Type: "user", Content: "Fix the login bug please"},
			{Timestamp: now.Add(-55 * time.Minute), Type: "tool_use", ToolName: "Read", Content: "/work/login.go"},
			{Timestamp: now.Add(-50 * time.Minute), Type: "tool_use", ToolName: "Edit", Content: "/work/login.go"},
			{Timestamp: now.Add(-45 * time.Minute), Type: "user", Content: "Run the tests"},
			{Timestamp: now.Add(-40 * time.Minute), Type: "tool_use", ToolName: "Bash", Content: "make test"},
		},
		FileActivity: []model.FileOp{
			{Path: "/work/login.go", Operation: "read", ToolName: "Read"},
			{Path: "/work/login.go", Operation: "edit", ToolName: "Edit"},
		},
		Subagents: []model.SubagentMeta{
			{AgentID: "agent1", Type: "Explore", ToolUsage: map[string]int{"Read": 3}},
		},
	}

	return meta, detail
}

func TestNewSessionDetailView(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)
	if view == nil {
		t.Fatal("expected non-nil detail view")
	}
	if view.activeTab != 0 {
		t.Errorf("expected activeTab=0, got %d", view.activeTab)
	}
}

func TestSessionDetailView_OverviewTab(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)
	// Navigate right once to reach Overview (index 1)
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	content := view.View(100, 30)

	if content == "" {
		t.Error("expected non-empty overview")
	}
	if !strings.Contains(content, "happy-cat") {
		t.Error("expected slug in overview")
	}
	if !strings.Contains(content, "Duration") {
		t.Error("expected 'Duration' label in overview")
	}
	if !strings.Contains(content, "Fix the login bug") {
		t.Error("expected initial prompt in overview")
	}
	// Should show tab names
	if !strings.Contains(content, "Overview") {
		t.Error("expected 'Overview' tab label")
	}
}

func TestSessionDetailView_TimelineTab(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)
	// Switch to timeline tab (index 2)
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	content := view.View(100, 30)

	if !strings.Contains(content, "Timeline") {
		t.Error("expected 'Timeline' tab label highlighted")
	}
	if !strings.Contains(content, "user") {
		t.Error("expected user event in timeline")
	}
	if !strings.Contains(content, "Read") {
		t.Error("expected tool_use Read in timeline")
	}
}

func TestSessionDetailView_FilesTab(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)
	// Switch to files tab (index 3)
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	content := view.View(100, 30)

	if !strings.Contains(content, "Files") {
		t.Error("expected 'Files' tab label")
	}
	if !strings.Contains(content, "login.go") {
		t.Error("expected file path in files tab")
	}
}

func TestSessionDetailView_AgentsTab(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)
	// Switch to agents tab (index 4)
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	content := view.View(100, 30)

	if !strings.Contains(content, "Agents") {
		t.Error("expected 'Agents' tab label")
	}
	if !strings.Contains(content, "Explore") {
		t.Error("expected agent type in agents tab")
	}
}

func TestSessionDetailView_ToolsTab(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)
	// Switch to tools tab (index 5)
	for i := 0; i < 5; i++ {
		view.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
	content := view.View(100, 30)

	if !strings.Contains(content, "Tools") {
		t.Error("expected 'Tools' tab label")
	}
	if !strings.Contains(content, "Read") {
		t.Error("expected Read tool in tools tab")
	}
	if !strings.Contains(content, "Edit") {
		t.Error("expected Edit tool in tools tab")
	}
}

func TestSessionDetailView_TabWrapping(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)

	// Go left from tab 0 should wrap to last tab
	view.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if view.activeTab != 5 {
		t.Errorf("expected tab 5 after left from 0, got %d", view.activeTab)
	}

	// Go right from last tab should wrap to 0
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	if view.activeTab != 0 {
		t.Errorf("expected tab 0 after right from 5, got %d", view.activeTab)
	}
}

func TestSessionDetailView_ChatTab(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)
	content := view.View(100, 30)

	if !strings.Contains(content, "Chat") {
		t.Error("expected 'Chat' tab label")
	}
	if !strings.Contains(content, "Fix the login bug") {
		t.Error("expected user message in chat")
	}
	if !strings.Contains(content, "▶ You:") {
		t.Error("expected user role marker")
	}
	if !strings.Contains(content, "◀ Assistant:") {
		t.Error("expected assistant role marker")
	}
	if !strings.Contains(content, "⚙") {
		t.Error("expected tool marker")
	}
}

func TestSessionDetailView_ChatTab_Empty(t *testing.T) {
	s := store.New()
	meta := &model.SessionMeta{
		UUID: "empty", Slug: "empty-session",
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{},
	}
	detail := &model.SessionDetail{Meta: meta}
	view := NewSessionDetailView(s, meta, detail)
	content := view.View(100, 30)

	if !strings.Contains(content, "No chat messages") {
		t.Error("expected empty state message")
	}
}

func TestSessionDetailView_EmptyDetail(t *testing.T) {
	s := store.New()
	meta := &model.SessionMeta{
		UUID: "empty", Slug: "empty-session",
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{},
	}
	detail := &model.SessionDetail{Meta: meta}
	view := NewSessionDetailView(s, meta, detail)
	content := view.View(100, 30)

	if content == "" {
		t.Error("expected non-empty view for empty detail")
	}
}

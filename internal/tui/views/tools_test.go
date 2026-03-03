package views

import (
	"strings"
	"testing"
	"time"

	"github.com/base-14/cicada/internal/model"
	"github.com/base-14/cicada/internal/store"
)

func TestNewToolsView(t *testing.T) {
	s := store.New()
	view := NewToolsView(s)
	if view == nil {
		t.Fatal("expected non-nil tools view")
	}
	if view.selected != 0 {
		t.Errorf("expected selected=0, got %d", view.selected)
	}
}

func TestToolsView_Empty(t *testing.T) {
	s := store.New()
	view := NewToolsView(s)
	content := view.View(80, 24)
	if content == "" {
		t.Error("expected non-empty view even with no data")
	}
	if !strings.Contains(content, "No session data") {
		t.Error("expected 'No session data' message for empty store")
	}
}

func TestToolsView_BuiltinTools(t *testing.T) {
	s := store.New()
	now := time.Now()

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-proj",
		StartTime: now, Duration: time.Hour,
		Models: map[string]int{}, ToolUsage: map[string]int{"Read": 10, "Edit": 5, "Bash": 3},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 10,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "s2", ProjectPath: "-Users-r-proj",
		StartTime: now.Add(-time.Hour), Duration: 30 * time.Minute,
		Models: map[string]int{}, ToolUsage: map[string]int{"Read": 5, "Bash": 2},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})

	view := NewToolsView(s)
	content := view.View(100, 30)

	// Should contain built-in tools section
	if !strings.Contains(content, "Built-in Tools") {
		t.Error("expected 'Built-in Tools' section header")
	}

	// Read has 15 total calls across 2 sessions
	if !strings.Contains(content, "Read") {
		t.Error("expected 'Read' tool in output")
	}
	if !strings.Contains(content, "15") {
		t.Error("expected Read call count of 15")
	}

	// Edit has 5 calls in 1 session
	if !strings.Contains(content, "Edit") {
		t.Error("expected 'Edit' tool in output")
	}
}

func TestToolsView_MCPTools(t *testing.T) {
	s := store.New()
	now := time.Now()

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-proj",
		StartTime: now, Duration: time.Hour,
		Models: map[string]int{},
		ToolUsage: map[string]int{
			"Read":                          3,
			"mcp__playwright__click":        10,
			"mcp__playwright__screenshot":   5,
			"mcp__github__create_issue":     2,
		},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 10,
	})

	view := NewToolsView(s)
	content := view.View(100, 30)

	// Should contain MCP tools section
	if !strings.Contains(content, "MCP Tools") {
		t.Error("expected 'MCP Tools' section header")
	}

	// Should show server grouping
	if !strings.Contains(content, "playwright") {
		t.Error("expected 'playwright' server name")
	}
	if !strings.Contains(content, "github") {
		t.Error("expected 'github' server name")
	}

	// Should show tool names (without full prefix)
	if !strings.Contains(content, "click") {
		t.Error("expected 'click' tool name")
	}
	if !strings.Contains(content, "screenshot") {
		t.Error("expected 'screenshot' tool name")
	}
}

func TestToolsView_SortedByCallCount(t *testing.T) {
	s := store.New()
	now := time.Now()

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-proj",
		StartTime: now, Duration: time.Hour,
		Models: map[string]int{},
		ToolUsage: map[string]int{"Read": 1, "Edit": 100, "Bash": 50},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 10,
	})

	view := NewToolsView(s)
	content := view.View(100, 30)

	// Edit (100) should appear before Bash (50) which should appear before Read (1)
	editIdx := strings.Index(content, "Edit")
	bashIdx := strings.Index(content, "Bash")
	readIdx := strings.Index(content, "Read")

	if editIdx < 0 || bashIdx < 0 || readIdx < 0 {
		t.Fatal("expected all tools in output")
	}
	if editIdx > bashIdx {
		t.Error("expected Edit before Bash (sorted by call count)")
	}
	if bashIdx > readIdx {
		t.Error("expected Bash before Read (sorted by call count)")
	}
}

func TestToolRow_MCPParsing(t *testing.T) {
	server, tool := parseMCPTool("mcp__playwright__click")
	if server != "playwright" {
		t.Errorf("expected server 'playwright', got %q", server)
	}
	if tool != "click" {
		t.Errorf("expected tool 'click', got %q", tool)
	}
}

func TestToolRow_MCPParsing_MultipleUnderscores(t *testing.T) {
	server, tool := parseMCPTool("mcp__my_server__do_thing")
	if server != "my_server" {
		t.Errorf("expected server 'my_server', got %q", server)
	}
	if tool != "do_thing" {
		t.Errorf("expected tool 'do_thing', got %q", tool)
	}
}

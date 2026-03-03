package parser

import (
	"testing"
)

func TestParseMessage_UserMessage(t *testing.T) {
	line := []byte(`{
		"type": "user",
		"uuid": "abc-123",
		"timestamp": "2026-03-03T10:00:00.000Z",
		"sessionId": "sess-1",
		"slug": "cool-slug",
		"cwd": "/Users/r/work",
		"gitBranch": "main",
		"isSidechain": false,
		"message": {
			"role": "user",
			"content": "Hello world"
		}
	}`)

	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "user" {
		t.Errorf("expected type 'user', got %q", msg.Type)
	}
	if msg.UUID != "abc-123" {
		t.Errorf("expected UUID 'abc-123', got %q", msg.UUID)
	}
	if msg.Slug != "cool-slug" {
		t.Errorf("expected slug 'cool-slug', got %q", msg.Slug)
	}
	if msg.GitBranch != "main" {
		t.Errorf("expected gitBranch 'main', got %q", msg.GitBranch)
	}
	if msg.UserContent() != "Hello world" {
		t.Errorf("expected user content 'Hello world', got %q", msg.UserContent())
	}
}

func TestParseMessage_AssistantMessage(t *testing.T) {
	line := []byte(`{
		"type": "assistant",
		"uuid": "def-456",
		"timestamp": "2026-03-03T10:01:00.000Z",
		"sessionId": "sess-1",
		"slug": "cool-slug",
		"gitBranch": "main",
		"isSidechain": false,
		"message": {
			"id": "msg_xxx",
			"role": "assistant",
			"model": "claude-opus-4-6",
			"stop_reason": "end_turn",
			"content": [
				{"type": "text", "text": "Hi there!"},
				{"type": "tool_use", "id": "toolu_01", "name": "Read", "input": {"file_path": "/tmp/foo.txt"}}
			],
			"usage": {
				"input_tokens": 100,
				"output_tokens": 50,
				"cache_creation_input_tokens": 200,
				"cache_read_input_tokens": 5000
			}
		}
	}`)

	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "assistant" {
		t.Errorf("expected type 'assistant', got %q", msg.Type)
	}
	if msg.Model() != "claude-opus-4-6" {
		t.Errorf("expected model 'claude-opus-4-6', got %q", msg.Model())
	}

	usage := msg.Usage()
	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", usage.OutputTokens)
	}
	if usage.CacheCreationInputTokens != 200 {
		t.Errorf("expected 200 cache creation tokens, got %d", usage.CacheCreationInputTokens)
	}
	if usage.CacheReadInputTokens != 5000 {
		t.Errorf("expected 5000 cache read tokens, got %d", usage.CacheReadInputTokens)
	}

	tools := msg.ToolUseBlocks()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool_use block, got %d", len(tools))
	}
	if tools[0].Name != "Read" {
		t.Errorf("expected tool name 'Read', got %q", tools[0].Name)
	}
}

func TestParseMessage_SummaryMessage(t *testing.T) {
	line := []byte(`{
		"type": "summary",
		"summary": "Fix the login bug",
		"leafUuid": "leaf-1"
	}`)

	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "summary" {
		t.Errorf("expected type 'summary', got %q", msg.Type)
	}
	if msg.Summary != "Fix the login bug" {
		t.Errorf("expected summary 'Fix the login bug', got %q", msg.Summary)
	}
}

func TestParseMessage_SkillDetection(t *testing.T) {
	line := []byte(`{
		"type": "assistant",
		"uuid": "skill-1",
		"timestamp": "2026-03-03T10:01:00.000Z",
		"sessionId": "sess-1",
		"slug": "cool-slug",
		"isSidechain": false,
		"message": {
			"id": "msg_xxx",
			"role": "assistant",
			"model": "claude-opus-4-6",
			"content": [
				{"type": "tool_use", "id": "toolu_02", "name": "Skill", "input": {"skill": "superpowers:brainstorming"}}
			],
			"usage": {"input_tokens": 10, "output_tokens": 5, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}
	}`)

	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tools := msg.ToolUseBlocks()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool block, got %d", len(tools))
	}
	if tools[0].Name != "Skill" {
		t.Errorf("expected 'Skill', got %q", tools[0].Name)
	}
	skill := tools[0].SkillName()
	if skill != "superpowers:brainstorming" {
		t.Errorf("expected 'superpowers:brainstorming', got %q", skill)
	}
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	line := []byte(`not valid json`)
	_, err := ParseMessage(line)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseMessage_UnknownType(t *testing.T) {
	line := []byte(`{"type": "progress", "uuid": "x"}`)
	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "progress" {
		t.Errorf("expected type 'progress', got %q", msg.Type)
	}
}

package model

import (
	"testing"
	"time"
)

func TestChatMessage_Defaults(t *testing.T) {
	msg := ChatMessage{
		Role:      "user",
		Content:   "Fix the login bug",
		Timestamp: time.Now(),
	}
	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}
	if msg.Content != "Fix the login bug" {
		t.Error("expected content to match")
	}
}

func TestChatMessage_ToolCall(t *testing.T) {
	msg := ChatMessage{
		Role:      "tool",
		Content:   "Read → /work/login.go",
		Timestamp: time.Now(),
		ToolName:  "Read",
	}
	if msg.Role != "tool" {
		t.Errorf("expected role 'tool', got %q", msg.Role)
	}
	if msg.ToolName != "Read" {
		t.Errorf("expected tool name 'Read', got %q", msg.ToolName)
	}
}

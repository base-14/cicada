package model

import "time"

// ChatMessage represents a single message in the session chat history.
type ChatMessage struct {
	Role      string    // "user", "assistant", "tool"
	Content   string    // full text for user/assistant, one-liner summary for tool
	Timestamp time.Time
	ToolName  string // only set for role="tool"
}

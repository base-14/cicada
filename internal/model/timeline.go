package model

import "time"

// TimelineEvent represents a single event in a session timeline.
type TimelineEvent struct {
	Timestamp time.Time
	Type      string // "user", "assistant", "tool_use", "tool_result"
	Content   string // truncated for display
	ToolName  string
	ActorID   string // session or subagent agent_id
}

// FileOp represents a file operation performed during a session.
type FileOp struct {
	Timestamp time.Time
	Path      string
	Operation string // "read", "write", "edit", "delete", "search"
	Actor     string // session UUID or subagent agent_id
	ActorType string // "session" or "subagent"
	ToolName  string
}

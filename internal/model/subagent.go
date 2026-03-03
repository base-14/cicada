package model

import "time"

// SubagentMeta holds metadata about a subagent invocation.
type SubagentMeta struct {
	AgentID       string
	Type          string // "Explore", "Plan", "Bash", "general-purpose", etc.
	InitialPrompt string
	ToolUsage     map[string]int
	TokensIn      int64
	TokensOut     int64
	Duration      time.Duration
}

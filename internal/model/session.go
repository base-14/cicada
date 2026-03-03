package model

import "time"

// SessionMeta holds lightweight metadata parsed during background scan.
type SessionMeta struct {
	UUID          string
	Slug          string
	ProjectPath   string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	InitialPrompt string
	SessionTitles []string
	Models        map[string]int // model name → message count
	TokensIn      int64
	TokensOut     int64
	CacheRead     int64
	CacheWrite    int64
	ToolUsage     map[string]int // tool name → count
	SkillsUsed    map[string]int
	CommandsUsed  map[string]int
	GitBranches   []string
	SubagentCount int
	FileOps       map[string]int // operation type → count
	MessageCount  int
}

// SessionDetail holds full session data, loaded lazily on navigation.
type SessionDetail struct {
	Meta         *SessionMeta
	Timeline     []TimelineEvent
	FileActivity []FileOp
	Subagents    []SubagentMeta
	ChatMessages []ChatMessage
}

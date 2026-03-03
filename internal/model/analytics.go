package model

// Analytics holds aggregated analytics computed from all sessions.
type Analytics struct {
	TotalSessions    int
	TotalTokensIn    int64
	TotalTokensOut   int64
	TotalCacheRead   int64
	TotalCacheWrite  int64
	ActiveProjects   int
	ModelsUsed       map[string]int // model → total message count
	ToolsUsed        map[string]int // tool → total call count
	SessionsByDate   map[string]int // "2026-03-03" → count
	WorkModeExplore  int            // Read, Grep, Glob, WebFetch, WebSearch calls
	WorkModeBuild    int            // Write, Edit calls
	WorkModeTest     int            // Bash, Agent, Task calls
}

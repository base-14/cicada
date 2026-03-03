package model

import "time"

// HistoryEntry represents a single prompt from ~/.claude/history.jsonl.
type HistoryEntry struct {
	Timestamp time.Time
	Project   string
	SessionID string
}

// HistoryStats holds aggregated statistics from history entries.
type HistoryStats struct {
	TotalPrompts  int
	ActiveDays    map[string]bool // "2006-01-02" -> true
	PromptsByDate map[string]int  // "2006-01-02" -> count
	HourCounts    map[int]int     // hour (0-23) -> count
	Heatmap       [7][24]int      // day-of-week (Mon=0..Sun=6) x hour
}

// ComputeHistoryStats aggregates a slice of history entries into stats.
func ComputeHistoryStats(entries []HistoryEntry) HistoryStats {
	stats := HistoryStats{
		ActiveDays:    make(map[string]bool),
		PromptsByDate: make(map[string]int),
		HourCounts:    make(map[int]int),
	}
	for _, e := range entries {
		stats.TotalPrompts++
		date := e.Timestamp.Format("2006-01-02")
		stats.ActiveDays[date] = true
		stats.PromptsByDate[date]++
		hour := e.Timestamp.Hour()
		stats.HourCounts[hour]++
		weekday := e.Timestamp.Weekday()
		day := int(weekday) - 1
		if day < 0 {
			day = 6
		}
		stats.Heatmap[day][hour]++
	}
	return stats
}

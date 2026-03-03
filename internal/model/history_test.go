package model

import (
	"testing"
	"time"
)

func TestComputeHistoryStats_Empty(t *testing.T) {
	stats := ComputeHistoryStats(nil)
	if stats.TotalPrompts != 0 {
		t.Errorf("expected 0 prompts, got %d", stats.TotalPrompts)
	}
	if len(stats.ActiveDays) != 0 {
		t.Errorf("expected 0 active days, got %d", len(stats.ActiveDays))
	}
}

func TestComputeHistoryStats(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), Project: "/work/a"},
		{Timestamp: time.Date(2026, 1, 10, 14, 0, 0, 0, time.Local), Project: "/work/a"},
		{Timestamp: time.Date(2026, 1, 11, 10, 0, 0, 0, time.Local), Project: "/work/b"},
	}
	stats := ComputeHistoryStats(entries)

	if stats.TotalPrompts != 3 {
		t.Errorf("expected 3 prompts, got %d", stats.TotalPrompts)
	}
	if len(stats.ActiveDays) != 2 {
		t.Errorf("expected 2 active days, got %d", len(stats.ActiveDays))
	}
	if stats.PromptsByDate["2026-01-10"] != 2 {
		t.Errorf("expected 2 prompts on Jan 10, got %d", stats.PromptsByDate["2026-01-10"])
	}
	if stats.HourCounts[9] != 1 {
		t.Errorf("expected 1 prompt at hour 9, got %d", stats.HourCounts[9])
	}
	if stats.HourCounts[14] != 1 {
		t.Errorf("expected 1 prompt at hour 14, got %d", stats.HourCounts[14])
	}
	// Jan 10 2026 is Saturday = weekday 6 (Sat), mapped to index 5
	// Jan 11 2026 is Sunday = weekday 0 (Sun), mapped to index 6
	if stats.Heatmap[5][9] != 1 {
		t.Errorf("expected heatmap[5][9]=1, got %d", stats.Heatmap[5][9])
	}
	if stats.Heatmap[6][10] != 1 {
		t.Errorf("expected heatmap[6][10]=1, got %d", stats.Heatmap[6][10])
	}
}

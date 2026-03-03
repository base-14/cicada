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

func TestComputeHistoryStats_TopWords(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), Prompt: "please fix the authentication bug"},
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.Local), Prompt: "fix the login authentication"},
		{Timestamp: time.Date(2026, 1, 11, 11, 0, 0, 0, time.Local), Prompt: "add authentication tests for login"},
	}
	stats := ComputeHistoryStats(entries)

	if len(stats.TopWords) == 0 {
		t.Fatal("expected TopWords to be non-empty")
	}
	// "authentication" appears 3 times, should be first
	if stats.TopWords[0].Word != "authentication" {
		t.Errorf("expected top word 'authentication', got %q", stats.TopWords[0].Word)
	}
	if stats.TopWords[0].Count != 3 {
		t.Errorf("expected count 3, got %d", stats.TopWords[0].Count)
	}
	// Should have at most 5 entries
	if len(stats.TopWords) > 5 {
		t.Errorf("expected at most 5 top words, got %d", len(stats.TopWords))
	}
}

func TestComputeHistoryStats_TopWords_SkipsStopWords(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), Prompt: "the the the please please please fix"},
	}
	stats := ComputeHistoryStats(entries)

	for _, wc := range stats.TopWords {
		if wc.Word == "the" || wc.Word == "please" {
			t.Errorf("stop word %q should not appear in TopWords", wc.Word)
		}
	}
}

func TestComputeHistoryStats_TopWords_SkipsShortWords(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), Prompt: "go is ok but testing works"},
	}
	stats := ComputeHistoryStats(entries)

	for _, wc := range stats.TopWords {
		if len(wc.Word) < 3 {
			t.Errorf("word %q shorter than 3 chars should not appear in TopWords", wc.Word)
		}
	}
}

func TestComputeHistoryStats_TopWords_EmptyPrompts(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), Prompt: ""},
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.Local), Prompt: ""},
	}
	stats := ComputeHistoryStats(entries)

	if len(stats.TopWords) != 0 {
		t.Errorf("expected 0 top words for empty prompts, got %d", len(stats.TopWords))
	}
}

func TestComputeHistoryStats_PromptLength(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), Prompt: "fix the bug"},          // 3 words
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.Local), Prompt: "add authentication"},   // 2 words
		{Timestamp: time.Date(2026, 1, 10, 11, 0, 0, 0, time.Local), Prompt: "hello"},                // 1 word
		{Timestamp: time.Date(2026, 1, 10, 12, 0, 0, 0, time.Local), Prompt: "refactor the entire authentication system please"}, // 6 words
	}
	stats := ComputeHistoryStats(entries)

	// Avg: (3+2+1+6)/4 = 3.0
	if stats.AvgPromptWords != 3.0 {
		t.Errorf("expected avg prompt words 3.0, got %.1f", stats.AvgPromptWords)
	}
	// P95 of [1,2,3,6]: sorted=[1,2,3,6], index=ceil(0.95*4)-1=3, value=6
	if stats.P95PromptWords != 6 {
		t.Errorf("expected p95 prompt words 6, got %d", stats.P95PromptWords)
	}
}

func TestComputeHistoryStats_PromptLength_EmptyPrompts(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), Prompt: ""},
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.Local), Prompt: ""},
	}
	stats := ComputeHistoryStats(entries)

	if stats.AvgPromptWords != 0 {
		t.Errorf("expected avg 0 for empty prompts, got %.1f", stats.AvgPromptWords)
	}
	if stats.P95PromptWords != 0 {
		t.Errorf("expected p95 0 for empty prompts, got %d", stats.P95PromptWords)
	}
}

func TestComputeHistoryStats_PromptLength_SingleEntry(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local), Prompt: "fix the bug now"},
	}
	stats := ComputeHistoryStats(entries)

	if stats.AvgPromptWords != 4.0 {
		t.Errorf("expected avg 4.0, got %.1f", stats.AvgPromptWords)
	}
	if stats.P95PromptWords != 4 {
		t.Errorf("expected p95 4, got %d", stats.P95PromptWords)
	}
}

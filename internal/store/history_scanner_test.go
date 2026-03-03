package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanHistory_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	content := `{"display":"hello","timestamp":1704067200000,"project":"/work/a","sessionId":"abc-123"}
{"display":"world","timestamp":1704153600000,"project":"/work/b","sessionId":"def-456"}
{"display":"no session","timestamp":1704240000000,"project":"/work/c"}
`
	os.WriteFile(path, []byte(content), 0644)

	entries, err := ScanHistory(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Project != "/work/a" {
		t.Errorf("expected project /work/a, got %s", entries[0].Project)
	}
	if entries[0].SessionID != "abc-123" {
		t.Errorf("expected sessionId abc-123, got %s", entries[0].SessionID)
	}
	if entries[2].SessionID != "" {
		t.Errorf("expected empty sessionId, got %s", entries[2].SessionID)
	}
	// Verify Prompt field is parsed from display
	if entries[0].Prompt != "hello" {
		t.Errorf("expected prompt 'hello', got %q", entries[0].Prompt)
	}
	if entries[1].Prompt != "world" {
		t.Errorf("expected prompt 'world', got %q", entries[1].Prompt)
	}
	// Verify timestamp: 1704067200000ms = 2024-01-01 00:00:00 UTC
	if entries[0].Timestamp.UTC().Year() != 2024 {
		t.Errorf("expected year 2024, got %d", entries[0].Timestamp.UTC().Year())
	}
}

func TestScanHistory_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	content := `{"display":"good","timestamp":1704067200000,"project":"/a"}
not json at all
{"display":"also good","timestamp":1704153600000,"project":"/b"}
`
	os.WriteFile(path, []byte(content), 0644)

	entries, err := ScanHistory(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 valid entries, got %d", len(entries))
	}
}

func TestScanHistory_FileNotFound(t *testing.T) {
	entries, err := ScanHistory("/nonexistent/history.jsonl")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

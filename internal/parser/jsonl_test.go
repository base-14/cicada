package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSessionFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := `{"type":"summary","summary":"Test session","leafUuid":"leaf-1"}
{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"s1","slug":"test","isSidechain":false,"message":{"role":"user","content":"Hello"}}
{"type":"assistant","uuid":"a1","timestamp":"2026-03-03T10:01:00.000Z","sessionId":"s1","slug":"test","isSidechain":false,"message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
	if messages[0].Type != "summary" {
		t.Errorf("expected first message type 'summary', got %q", messages[0].Type)
	}
	if messages[1].Type != "user" {
		t.Errorf("expected second message type 'user', got %q", messages[1].Type)
	}
	if messages[2].Type != "assistant" {
		t.Errorf("expected third message type 'assistant', got %q", messages[2].Type)
	}
}

func TestReadSessionFile_SkipsBadLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")

	content := `not json at all
{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"s1","slug":"test","isSidechain":false,"message":{"role":"user","content":"Hello"}}
also not json
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 valid message, got %d", len(messages))
	}
}

func TestReadSessionFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}

func TestReadSessionFile_NotFound(t *testing.T) {
	_, err := ReadSessionFile("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

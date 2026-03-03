package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSessionMeta(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := `{"type":"summary","summary":"Fix the login bug"}
{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"s1","slug":"happy-cat","cwd":"/work","gitBranch":"main","isSidechain":false,"message":{"role":"user","content":"Fix the login bug please"}}
{"type":"assistant","uuid":"a1","timestamp":"2026-03-03T10:01:00.000Z","sessionId":"s1","slug":"happy-cat","gitBranch":"main","isSidechain":false,"message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_01","name":"Read","input":{"file_path":"/work/login.go"}},{"type":"tool_use","id":"toolu_02","name":"Edit","input":{"file_path":"/work/login.go","old_string":"foo","new_string":"bar"}}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":200,"cache_read_input_tokens":5000}}}
{"type":"assistant","uuid":"a2","timestamp":"2026-03-03T10:02:00.000Z","sessionId":"s1","slug":"happy-cat","gitBranch":"feature","isSidechain":false,"message":{"id":"msg_2","role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"tool_use","id":"toolu_03","name":"Bash","input":{"command":"make test"}}],"usage":{"input_tokens":50,"output_tokens":25,"cache_creation_input_tokens":0,"cache_read_input_tokens":1000}}}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}

	meta := ExtractSessionMeta(messages, "/projects/-work", "s1.jsonl")
	if meta.Slug != "happy-cat" {
		t.Errorf("expected slug 'happy-cat', got %q", meta.Slug)
	}
	if meta.InitialPrompt != "Fix the login bug please" {
		t.Errorf("expected initial prompt, got %q", meta.InitialPrompt)
	}
	if len(meta.SessionTitles) != 1 || meta.SessionTitles[0] != "Fix the login bug" {
		t.Errorf("expected session title 'Fix the login bug', got %v", meta.SessionTitles)
	}
	if meta.TokensIn != 150 {
		t.Errorf("expected 150 input tokens, got %d", meta.TokensIn)
	}
	if meta.TokensOut != 75 {
		t.Errorf("expected 75 output tokens, got %d", meta.TokensOut)
	}
	if meta.CacheRead != 6000 {
		t.Errorf("expected 6000 cache read tokens, got %d", meta.CacheRead)
	}
	if meta.CacheWrite != 200 {
		t.Errorf("expected 200 cache write tokens, got %d", meta.CacheWrite)
	}
	if meta.ToolUsage["Read"] != 1 {
		t.Errorf("expected Read tool count 1, got %d", meta.ToolUsage["Read"])
	}
	if meta.ToolUsage["Edit"] != 1 {
		t.Errorf("expected Edit tool count 1, got %d", meta.ToolUsage["Edit"])
	}
	if meta.ToolUsage["Bash"] != 1 {
		t.Errorf("expected Bash tool count 1, got %d", meta.ToolUsage["Bash"])
	}
	if meta.Models["claude-opus-4-6"] != 1 {
		t.Errorf("expected 1 opus message, got %d", meta.Models["claude-opus-4-6"])
	}
	if meta.Models["claude-sonnet-4-6"] != 1 {
		t.Errorf("expected 1 sonnet message, got %d", meta.Models["claude-sonnet-4-6"])
	}
	if meta.MessageCount != 3 {
		t.Errorf("expected 3 messages (1 user + 2 assistant), got %d", meta.MessageCount)
	}

	// Git branches
	branchSet := make(map[string]bool)
	for _, b := range meta.GitBranches {
		branchSet[b] = true
	}
	if !branchSet["main"] || !branchSet["feature"] {
		t.Errorf("expected branches [main, feature], got %v", meta.GitBranches)
	}

	// File ops
	if meta.FileOps["read"] != 1 {
		t.Errorf("expected 1 read file op, got %d", meta.FileOps["read"])
	}
	if meta.FileOps["edit"] != 1 {
		t.Errorf("expected 1 edit file op, got %d", meta.FileOps["edit"])
	}
}

func TestExtractSessionMeta_EmptyMessages(t *testing.T) {
	meta := ExtractSessionMeta(nil, "/projects/-work", "empty.jsonl")
	if meta.MessageCount != 0 {
		t.Errorf("expected 0 messages, got %d", meta.MessageCount)
	}
	if meta.Slug != "" {
		t.Errorf("expected empty slug, got %q", meta.Slug)
	}
}

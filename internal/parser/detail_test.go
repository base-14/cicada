package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/r/cicada/internal/model"
)

func TestExtractSessionDetail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := `{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"s1","slug":"happy-cat","cwd":"/work","gitBranch":"main","isSidechain":false,"message":{"role":"user","content":"Fix the login bug please"}}
{"type":"assistant","uuid":"a1","timestamp":"2026-03-03T10:01:00.000Z","sessionId":"s1","slug":"happy-cat","gitBranch":"main","isSidechain":false,"message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"Let me look at the login file."},{"type":"tool_use","id":"toolu_01","name":"Read","input":{"file_path":"/work/login.go"}},{"type":"tool_use","id":"toolu_02","name":"Edit","input":{"file_path":"/work/login.go","old_string":"foo","new_string":"bar"}}],"usage":{"input_tokens":100,"output_tokens":50}}}
{"type":"user","uuid":"u2","timestamp":"2026-03-03T10:02:00.000Z","sessionId":"s1","slug":"happy-cat","isSidechain":false,"message":{"role":"user","content":"Looks good, run tests"}}
{"type":"assistant","uuid":"a2","timestamp":"2026-03-03T10:03:00.000Z","sessionId":"s1","slug":"happy-cat","isSidechain":false,"message":{"id":"msg_2","role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_03","name":"Bash","input":{"command":"make test"}}],"usage":{"input_tokens":50,"output_tokens":25}}}
{"type":"user","uuid":"u3","timestamp":"2026-03-03T10:04:00.000Z","sessionId":"s1","slug":"happy-cat","isSidechain":true,"message":{"role":"user","content":"sidechain message should be skipped"}}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}

	meta := &model.SessionMeta{UUID: "s1", Slug: "happy-cat"}
	detail := ExtractSessionDetail(messages, meta)

	if detail == nil {
		t.Fatal("expected non-nil detail")
	}
	if detail.Meta != meta {
		t.Error("expected detail.Meta to point to the provided meta")
	}

	// Timeline events: user1, assistant_text, tool_use(Read), tool_use(Edit), user2, tool_use(Bash)
	// Sidechain user should be skipped
	if len(detail.Timeline) < 4 {
		t.Fatalf("expected at least 4 timeline events, got %d", len(detail.Timeline))
	}

	// First event should be user message
	if detail.Timeline[0].Type != "user" {
		t.Errorf("expected first event type 'user', got %q", detail.Timeline[0].Type)
	}
	if detail.Timeline[0].Content != "Fix the login bug please" {
		t.Errorf("expected first event content, got %q", detail.Timeline[0].Content)
	}

	// Should have tool_use events for Read and Edit
	toolEvents := 0
	for _, ev := range detail.Timeline {
		if ev.Type == "tool_use" {
			toolEvents++
		}
	}
	if toolEvents < 3 {
		t.Errorf("expected at least 3 tool_use events (Read, Edit, Bash), got %d", toolEvents)
	}

	// File activity should have Read and Edit entries
	if len(detail.FileActivity) < 2 {
		t.Fatalf("expected at least 2 file activity entries, got %d", len(detail.FileActivity))
	}

	// Check file ops
	foundRead := false
	foundEdit := false
	for _, fa := range detail.FileActivity {
		if fa.Operation == "read" && fa.Path == "/work/login.go" {
			foundRead = true
		}
		if fa.Operation == "edit" && fa.Path == "/work/login.go" {
			foundEdit = true
		}
	}
	if !foundRead {
		t.Error("expected file activity entry for Read /work/login.go")
	}
	if !foundEdit {
		t.Error("expected file activity entry for Edit /work/login.go")
	}
}

func TestExtractSessionDetail_Empty(t *testing.T) {
	meta := &model.SessionMeta{UUID: "empty"}
	detail := ExtractSessionDetail(nil, meta)
	if detail == nil {
		t.Fatal("expected non-nil detail even with no messages")
	}
	if len(detail.Timeline) != 0 {
		t.Errorf("expected 0 timeline events, got %d", len(detail.Timeline))
	}
	if len(detail.FileActivity) != 0 {
		t.Errorf("expected 0 file activity entries, got %d", len(detail.FileActivity))
	}
}

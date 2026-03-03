package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestSession(t *testing.T, dir, uuid string) {
	t.Helper()
	content := `{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"` + uuid + `","slug":"test-slug","gitBranch":"main","isSidechain":false,"message":{"role":"user","content":"Hello"}}
{"type":"assistant","uuid":"a1","timestamp":"2026-03-03T10:01:00.000Z","sessionId":"` + uuid + `","slug":"test-slug","gitBranch":"main","isSidechain":false,"message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	path := filepath.Join(dir, uuid+".jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestScanner_DiscoverAndParse(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "-Users-r-work-myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeTestSession(t, projectDir, "sess-001")
	writeTestSession(t, projectDir, "sess-002")

	s := New()
	scanner := NewScanner(s, baseDir)

	var msgs []ScanMsg
	msgCh := make(chan ScanMsg, 100)
	done := make(chan struct{})
	go func() {
		for msg := range msgCh {
			msgs = append(msgs, msg)
		}
		close(done)
	}()

	scanner.Run(msgCh)
	close(msgCh)
	<-done

	all := s.AllSessions()
	if len(all) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(all))
	}

	if len(msgs) == 0 {
		t.Error("expected at least one scan message")
	}

	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Type != ScanComplete {
		t.Errorf("expected last message type ScanComplete, got %v", lastMsg.Type)
	}
}

func TestScanner_EmptyProjectsDir(t *testing.T) {
	baseDir := t.TempDir()

	s := New()
	scanner := NewScanner(s, baseDir)

	msgCh := make(chan ScanMsg, 100)
	done := make(chan struct{})
	go func() {
		for range msgCh {
		}
		close(done)
	}()

	scanner.Run(msgCh)
	close(msgCh)
	<-done

	if len(s.AllSessions()) != 0 {
		t.Error("expected 0 sessions for empty projects dir")
	}
}

func TestScanner_NonexistentDir(t *testing.T) {
	s := New()
	scanner := NewScanner(s, "/nonexistent/path")

	msgCh := make(chan ScanMsg, 100)
	done := make(chan struct{})
	var msgs []ScanMsg
	go func() {
		for msg := range msgCh {
			msgs = append(msgs, msg)
		}
		close(done)
	}()

	scanner.Run(msgCh)
	close(msgCh)
	<-done

	found := false
	for _, msg := range msgs {
		if msg.Type == ScanComplete {
			found = true
		}
	}
	if !found {
		t.Error("expected ScanComplete message even for nonexistent dir")
	}
}

func TestScanner_Timeout(t *testing.T) {
	baseDir := t.TempDir()
	s := New()
	scanner := NewScanner(s, baseDir)

	msgCh := make(chan ScanMsg, 100)
	done := make(chan struct{})
	go func() {
		for range msgCh {
		}
		close(done)
	}()

	start := time.Now()
	scanner.Run(msgCh)
	close(msgCh)
	<-done

	if time.Since(start) > 5*time.Second {
		t.Error("scanner took too long for empty dir")
	}
}

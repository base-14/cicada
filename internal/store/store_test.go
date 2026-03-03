package store

import (
	"testing"
	"time"

	"github.com/r/cicada/internal/model"
)

func newTestMeta(uuid, slug, project string, start time.Time, tokensIn int64) *model.SessionMeta {
	return &model.SessionMeta{
		UUID:         uuid,
		Slug:         slug,
		ProjectPath:  project,
		StartTime:    start,
		EndTime:      start.Add(10 * time.Minute),
		Duration:     10 * time.Minute,
		TokensIn:     tokensIn,
		TokensOut:    tokensIn / 2,
		Models:       map[string]int{"claude-opus-4-6": 1},
		ToolUsage:    map[string]int{"Read": 2, "Edit": 1},
		FileOps:      map[string]int{"read": 2, "edit": 1},
		SkillsUsed:   map[string]int{},
		CommandsUsed: map[string]int{},
		MessageCount: 5,
	}
}

func TestStore_AddAndGet(t *testing.T) {
	s := New()
	now := time.Now()
	meta := newTestMeta("uuid-1", "cool-slug", "/work/project", now, 1000)

	s.Add(meta)

	got := s.Get("uuid-1")
	if got == nil {
		t.Fatal("expected to find session")
	}
	if got.Slug != "cool-slug" {
		t.Errorf("expected slug 'cool-slug', got %q", got.Slug)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	s := New()
	if s.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent session")
	}
}

func TestStore_AllSessions(t *testing.T) {
	s := New()
	now := time.Now()
	s.Add(newTestMeta("u1", "s1", "/p1", now, 100))
	s.Add(newTestMeta("u2", "s2", "/p2", now.Add(-time.Hour), 200))

	all := s.AllSessions()
	if len(all) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(all))
	}
}

func TestStore_Projects(t *testing.T) {
	s := New()
	now := time.Now()
	s.Add(newTestMeta("u1", "s1", "/p1", now, 100))
	s.Add(newTestMeta("u2", "s2", "/p1", now, 200))
	s.Add(newTestMeta("u3", "s3", "/p2", now, 300))

	projects := s.Projects()
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestStore_SessionsByProject(t *testing.T) {
	s := New()
	now := time.Now()
	s.Add(newTestMeta("u1", "s1", "/p1", now, 100))
	s.Add(newTestMeta("u2", "s2", "/p1", now, 200))
	s.Add(newTestMeta("u3", "s3", "/p2", now, 300))

	p1Sessions := s.SessionsByProject("/p1")
	if len(p1Sessions) != 2 {
		t.Fatalf("expected 2 sessions for /p1, got %d", len(p1Sessions))
	}
}

func TestStore_Analytics(t *testing.T) {
	s := New()
	now := time.Now()
	s.Add(newTestMeta("u1", "s1", "/p1", now, 1000))
	s.Add(newTestMeta("u2", "s2", "/p2", now, 2000))

	analytics := s.Analytics()
	if analytics.TotalSessions != 2 {
		t.Errorf("expected 2 total sessions, got %d", analytics.TotalSessions)
	}
	if analytics.TotalTokensIn != 3000 {
		t.Errorf("expected 3000 total tokens in, got %d", analytics.TotalTokensIn)
	}
	if analytics.ActiveProjects != 2 {
		t.Errorf("expected 2 active projects, got %d", analytics.ActiveProjects)
	}
	if analytics.ToolsUsed["Read"] != 4 {
		t.Errorf("expected 4 total Read calls, got %d", analytics.ToolsUsed["Read"])
	}
}

func TestStore_ScanProgress(t *testing.T) {
	s := New()
	s.SetScanProgress(50, 100)

	scanned, total := s.ScanProgress()
	if scanned != 50 || total != 100 {
		t.Errorf("expected 50/100, got %d/%d", scanned, total)
	}
}

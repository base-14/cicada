package transfer

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/base-14/cicada/internal/model"
)

func TestExportSession(t *testing.T) {
	// Setup: create a fake claude dir with a JSONL file.
	claudeDir := t.TempDir()
	outputDir := t.TempDir()

	projectPath := "-Users-testuser-myproject"
	uuid := "abcd1234-5678-9012-3456-789012345678"
	slug := "my-test-session"

	// Create the project directory and JSONL file.
	projectDir := filepath.Join(claudeDir, "projects", projectPath)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonlContent := []byte(`{"type":"human","text":"hello"}` + "\n" + `{"type":"assistant","text":"hi"}` + "\n")
	if err := os.WriteFile(filepath.Join(projectDir, uuid+".jsonl"), jsonlContent, 0o644); err != nil {
		t.Fatal(err)
	}

	meta := &model.SessionMeta{
		UUID:        uuid,
		Slug:        slug,
		ProjectPath: projectPath,
	}

	// Export.
	filename, err := ExportSession(claudeDir, meta, outputDir)
	if err != nil {
		t.Fatalf("ExportSession() error: %v", err)
	}

	// Verify filename format: {slug}-{uuid-first-8}.zip
	expectedFilename := slug + "-" + uuid[:8] + ".zip"
	if filename != expectedFilename {
		t.Errorf("filename = %q, want %q", filename, expectedFilename)
	}

	// Open the zip and verify contents.
	zipPath := filepath.Join(outputDir, filename)
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	// Expect exactly 2 files: manifest.json and {uuid}.jsonl
	if len(r.File) != 2 {
		t.Fatalf("zip has %d files, want 2", len(r.File))
	}

	fileNames := make(map[string]bool)
	for _, f := range r.File {
		fileNames[f.Name] = true
	}

	if !fileNames["manifest.json"] {
		t.Error("zip missing manifest.json")
	}
	expectedJSONL := uuid + ".jsonl"
	if !fileNames[expectedJSONL] {
		t.Errorf("zip missing %s", expectedJSONL)
	}

	// Read and verify manifest.
	for _, f := range r.File {
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			var m Manifest
			if err := json.NewDecoder(rc).Decode(&m); err != nil {
				t.Fatal(err)
			}
			rc.Close()

			if m.Version != 1 {
				t.Errorf("manifest version = %d, want 1", m.Version)
			}
			if m.Type != "single" {
				t.Errorf("manifest type = %q, want %q", m.Type, "single")
			}
			if m.SessionUUID != uuid {
				t.Errorf("manifest session_uuid = %q, want %q", m.SessionUUID, uuid)
			}
			if m.ProjectPath != projectPath {
				t.Errorf("manifest project_path = %q, want %q", m.ProjectPath, projectPath)
			}
			if m.Slug != slug {
				t.Errorf("manifest slug = %q, want %q", m.Slug, slug)
			}
			if m.ExportedAt.IsZero() {
				t.Error("manifest exported_at is zero")
			}
		}
	}
}

func TestExportSessionMissingFile(t *testing.T) {
	claudeDir := t.TempDir()
	outputDir := t.TempDir()

	meta := &model.SessionMeta{
		UUID:        "nonexistent-uuid-0000-0000-000000000000",
		Slug:        "missing",
		ProjectPath: "-Users-nobody-noproject",
	}

	_, err := ExportSession(claudeDir, meta, outputDir)
	if err == nil {
		t.Fatal("ExportSession() expected error for missing file, got nil")
	}
}

func TestExportAll(t *testing.T) {
	claudeDir := t.TempDir()
	outputDir := t.TempDir()

	// Create two sessions across two projects.
	sessions := []struct {
		projectPath string
		uuid        string
		slug        string
		content     string
	}{
		{
			projectPath: "-Users-testuser-projectA",
			uuid:        "aaaa1111-2222-3333-4444-555566667777",
			slug:        "session-a",
			content:     `{"type":"human","text":"alpha"}` + "\n",
		},
		{
			projectPath: "-Users-testuser-projectB",
			uuid:        "bbbb1111-2222-3333-4444-555566667777",
			slug:        "session-b",
			content:     `{"type":"human","text":"beta"}` + "\n",
		},
	}

	var metas []*model.SessionMeta
	for _, s := range sessions {
		dir := filepath.Join(claudeDir, "projects", s.projectPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, s.uuid+".jsonl"), []byte(s.content), 0o644); err != nil {
			t.Fatal(err)
		}
		metas = append(metas, &model.SessionMeta{
			UUID:        s.uuid,
			Slug:        s.slug,
			ProjectPath: s.projectPath,
		})
	}

	filename, err := ExportAll(claudeDir, metas, outputDir)
	if err != nil {
		t.Fatalf("ExportAll() error: %v", err)
	}

	// Verify filename format: cicada-export-YYYY-MM-DD.zip
	if !strings.HasPrefix(filename, "cicada-export-") || !strings.HasSuffix(filename, ".zip") {
		t.Errorf("unexpected filename format: %q", filename)
	}

	// Open zip and verify contents.
	zipPath := filepath.Join(outputDir, filename)
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	// Expect: manifest.json + 2 jsonl files = 3 files.
	if len(r.File) != 3 {
		t.Fatalf("zip has %d files, want 3", len(r.File))
	}

	fileNames := make(map[string]bool)
	for _, f := range r.File {
		fileNames[f.Name] = true
	}

	if !fileNames["manifest.json"] {
		t.Error("zip missing manifest.json")
	}
	for _, s := range sessions {
		expected := s.projectPath + "/" + s.uuid + ".jsonl"
		if !fileNames[expected] {
			t.Errorf("zip missing %s", expected)
		}
	}

	// Read and verify manifest.
	for _, f := range r.File {
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			var m Manifest
			if err := json.NewDecoder(rc).Decode(&m); err != nil {
				t.Fatal(err)
			}
			rc.Close()

			if m.Version != 1 {
				t.Errorf("manifest version = %d, want 1", m.Version)
			}
			if m.Type != "bulk" {
				t.Errorf("manifest type = %q, want %q", m.Type, "bulk")
			}
			if len(m.Sessions) != 2 {
				t.Fatalf("manifest sessions count = %d, want 2", len(m.Sessions))
			}
			if m.ExportedAt.IsZero() {
				t.Error("manifest exported_at is zero")
			}

			// Verify each session entry.
			for i, s := range sessions {
				entry := m.Sessions[i]
				if entry.SessionUUID != s.uuid {
					t.Errorf("session[%d] uuid = %q, want %q", i, entry.SessionUUID, s.uuid)
				}
				if entry.ProjectPath != s.projectPath {
					t.Errorf("session[%d] project_path = %q, want %q", i, entry.ProjectPath, s.projectPath)
				}
				if entry.Slug != s.slug {
					t.Errorf("session[%d] slug = %q, want %q", i, entry.Slug, s.slug)
				}
			}
		}
	}
}

func TestExportAllSkipsUnreadable(t *testing.T) {
	claudeDir := t.TempDir()
	outputDir := t.TempDir()

	// One valid session and one with missing file.
	goodUUID := "good1111-2222-3333-4444-555566667777"
	goodProject := "-Users-testuser-good"
	dir := filepath.Join(claudeDir, "projects", goodProject)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, goodUUID+".jsonl"), []byte(`{"x":1}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	metas := []*model.SessionMeta{
		{UUID: goodUUID, Slug: "good", ProjectPath: goodProject},
		{UUID: "bad-uuid", Slug: "bad", ProjectPath: "-Users-testuser-missing"},
	}

	filename, err := ExportAll(claudeDir, metas, outputDir)
	if err != nil {
		t.Fatalf("ExportAll() should succeed with partial data, got error: %v", err)
	}

	r, err := zip.OpenReader(filepath.Join(outputDir, filename))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// Should have manifest.json + 1 good jsonl = 2 files.
	if len(r.File) != 2 {
		t.Errorf("zip has %d files, want 2", len(r.File))
	}
}

func TestExportAllAllUnreadable(t *testing.T) {
	claudeDir := t.TempDir()
	outputDir := t.TempDir()

	metas := []*model.SessionMeta{
		{UUID: "bad-1", Slug: "bad1", ProjectPath: "-missing1"},
		{UUID: "bad-2", Slug: "bad2", ProjectPath: "-missing2"},
	}

	_, err := ExportAll(claudeDir, metas, outputDir)
	if err == nil {
		t.Fatal("ExportAll() expected error when all sessions unreadable, got nil")
	}
}

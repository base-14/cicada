package transfer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/base-14/cicada/internal/model"
)

func TestReadBundleSingle(t *testing.T) {
	// Setup: create a fake claude dir with a JSONL file, export it, then read the bundle.
	claudeDir := t.TempDir()
	outputDir := t.TempDir()

	projectPath := "-Users-testuser-myproject"
	uuid := "abcd1234-5678-9012-3456-789012345678"
	slug := "test-session"
	jsonlContent := []byte(`{"type":"human","text":"hello"}` + "\n" + `{"type":"assistant","text":"hi"}` + "\n")

	// Create JSONL file for export.
	projectDir := filepath.Join(claudeDir, "projects", projectPath)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, uuid+".jsonl"), jsonlContent, 0o644); err != nil {
		t.Fatal(err)
	}

	meta := &model.SessionMeta{
		UUID:        uuid,
		Slug:        slug,
		ProjectPath: projectPath,
	}

	filename, err := ExportSession(claudeDir, meta, outputDir)
	if err != nil {
		t.Fatalf("ExportSession() error: %v", err)
	}

	// ReadBundle.
	zipPath := filepath.Join(outputDir, filename)
	manifest, files, err := ReadBundle(zipPath)
	if err != nil {
		t.Fatalf("ReadBundle() error: %v", err)
	}

	// Verify manifest.
	if manifest.Version != 1 {
		t.Errorf("manifest version = %d, want 1", manifest.Version)
	}
	if manifest.Type != "single" {
		t.Errorf("manifest type = %q, want %q", manifest.Type, "single")
	}
	if manifest.SessionUUID != uuid {
		t.Errorf("manifest session_uuid = %q, want %q", manifest.SessionUUID, uuid)
	}
	if manifest.ProjectPath != projectPath {
		t.Errorf("manifest project_path = %q, want %q", manifest.ProjectPath, projectPath)
	}

	// Verify files.
	expectedKey := uuid + ".jsonl"
	data, ok := files[expectedKey]
	if !ok {
		t.Fatalf("files missing key %q, got keys: %v", expectedKey, mapKeys(files))
	}
	if !bytes.Equal(data, jsonlContent) {
		t.Errorf("file content mismatch:\n  got:  %q\n  want: %q", data, jsonlContent)
	}
}

func TestReadBundleBulk(t *testing.T) {
	claudeDir := t.TempDir()
	outputDir := t.TempDir()

	sessions := []struct {
		projectPath string
		uuid        string
		slug        string
		content     []byte
	}{
		{"-Users-testuser-projA", "aaaa1111-2222-3333-4444-555566667777", "sess-a", []byte(`{"a":1}` + "\n")},
		{"-Users-testuser-projB", "bbbb1111-2222-3333-4444-555566667777", "sess-b", []byte(`{"b":2}` + "\n")},
	}

	var metas []*model.SessionMeta
	for _, s := range sessions {
		dir := filepath.Join(claudeDir, "projects", s.projectPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, s.uuid+".jsonl"), s.content, 0o644); err != nil {
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

	manifest, files, err := ReadBundle(filepath.Join(outputDir, filename))
	if err != nil {
		t.Fatalf("ReadBundle() error: %v", err)
	}

	if manifest.Type != "bulk" {
		t.Errorf("manifest type = %q, want %q", manifest.Type, "bulk")
	}
	if len(files) != 2 {
		t.Errorf("file count = %d, want 2", len(files))
	}
}

func TestReadBundleInvalidPath(t *testing.T) {
	_, _, err := ReadBundle("/nonexistent/path/to/bundle.zip")
	if err == nil {
		t.Fatal("ReadBundle() expected error for nonexistent file, got nil")
	}
}

func TestPlaceSession(t *testing.T) {
	claudeDir := t.TempDir()
	projectPath := "-Users-testuser-myproject"
	uuid := "abcd1234-5678-9012-3456-789012345678"
	content := []byte(`{"type":"human","text":"hello"}` + "\n")

	// Pre-create the project directory.
	dir := filepath.Join(claudeDir, "projects", projectPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := PlaceSession(claudeDir, projectPath, uuid, content); err != nil {
		t.Fatalf("PlaceSession() error: %v", err)
	}

	// Verify file exists with correct content.
	placed := filepath.Join(dir, uuid+".jsonl")
	got, err := os.ReadFile(placed)
	if err != nil {
		t.Fatalf("reading placed file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("placed content mismatch:\n  got:  %q\n  want: %q", got, content)
	}
}

func TestPlaceSessionCreatesDir(t *testing.T) {
	claudeDir := t.TempDir()
	projectPath := "-Users-newuser-newproject"
	uuid := "dddd1111-2222-3333-4444-555566667777"
	content := []byte(`{"new":true}` + "\n")

	// Do NOT create the directory beforehand.
	if err := PlaceSession(claudeDir, projectPath, uuid, content); err != nil {
		t.Fatalf("PlaceSession() error: %v", err)
	}

	// Verify both dir and file exist.
	placed := filepath.Join(claudeDir, "projects", projectPath, uuid+".jsonl")
	got, err := os.ReadFile(placed)
	if err != nil {
		t.Fatalf("reading placed file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("placed content mismatch:\n  got:  %q\n  want: %q", got, content)
	}
}

func TestRoundTripSingleExportImport(t *testing.T) {
	// Setup: create a multi-line JSONL session.
	claudeDir := t.TempDir()
	outputDir := t.TempDir()
	importClaudeDir := t.TempDir()

	projectPath := "-Users-testuser-roundtrip"
	uuid := "rtrt1234-5678-9012-3456-789012345678"
	slug := "round-trip"
	originalContent := []byte(
		`{"type":"human","text":"line one"}` + "\n" +
			`{"type":"assistant","text":"line two"}` + "\n" +
			`{"type":"human","text":"line three"}` + "\n",
	)

	projectDir := filepath.Join(claudeDir, "projects", projectPath)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, uuid+".jsonl"), originalContent, 0o644); err != nil {
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

	// Read bundle.
	manifest, files, err := ReadBundle(filepath.Join(outputDir, filename))
	if err != nil {
		t.Fatalf("ReadBundle() error: %v", err)
	}

	// Place session into a different claude dir with a different project path.
	importProjectPath := "-Users-otheruser-imported"
	dataKey := manifest.SessionUUID + ".jsonl"
	data, ok := files[dataKey]
	if !ok {
		t.Fatalf("files missing key %q", dataKey)
	}

	if err := PlaceSession(importClaudeDir, importProjectPath, manifest.SessionUUID, data); err != nil {
		t.Fatalf("PlaceSession() error: %v", err)
	}

	// Verify byte-for-byte match with original.
	importedPath := filepath.Join(importClaudeDir, "projects", importProjectPath, uuid+".jsonl")
	imported, err := os.ReadFile(importedPath)
	if err != nil {
		t.Fatalf("reading imported file: %v", err)
	}
	if !bytes.Equal(imported, originalContent) {
		t.Errorf("round-trip content mismatch:\n  got:  %q\n  want: %q", imported, originalContent)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"tilde prefix", "~/foo/bar", filepath.Join(home, "foo/bar")},
		{"no tilde", "/absolute/path", "/absolute/path"},
		{"tilde only", "~", home},
		{"relative", "relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.in)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// mapKeys returns the keys of a map for diagnostic output.
func mapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

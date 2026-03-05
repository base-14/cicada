# Session Export/Import — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add export (`e`/`E`) and import (`i`) keybindings to the sessions tab for sharing and migrating Claude Code sessions as zip bundles.

**Architecture:** New `internal/transfer/` package handles zip creation/reading with manifest + raw JSONL files. TUI gets new components (text input, project picker) for the import flow. Export runs as a `tea.Cmd` returning a result message. Import is multi-step: path input → read zip → project picker (single) or direct placement (bulk).

**Tech Stack:** Go stdlib `archive/zip`, `encoding/json`, `os`. Bubbletea for TUI components.

**Design doc:** `docs/plans/2026-03-05-session-export-import-design.md`

---

### Task 1: Manifest types and serialization

**Files:**
- Create: `internal/transfer/manifest.go`
- Create: `internal/transfer/manifest_test.go`

**Step 1: Write the failing test**

```go
// internal/transfer/manifest_test.go
package transfer

import (
	"encoding/json"
	"testing"
	"time"
)

func TestManifestSingleMarshal(t *testing.T) {
	m := Manifest{
		Version:     1,
		Type:        "single",
		ExportedAt:  time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC),
		ProjectPath: "-Users-r-work-myproject",
		SessionUUID: "abc-123",
		Slug:        "happy-cat",
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Manifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
	if got.Type != "single" {
		t.Errorf("type = %q, want %q", got.Type, "single")
	}
	if got.SessionUUID != "abc-123" {
		t.Errorf("session_uuid = %q, want %q", got.SessionUUID, "abc-123")
	}
}

func TestManifestBulkMarshal(t *testing.T) {
	m := Manifest{
		Version:    1,
		Type:       "bulk",
		ExportedAt: time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC),
		Sessions: []BulkSessionEntry{
			{ProjectPath: "-Users-r-projA", SessionUUID: "uuid1", Slug: "cat"},
			{ProjectPath: "-Users-r-projB", SessionUUID: "uuid2", Slug: "dog"},
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Manifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != "bulk" {
		t.Errorf("type = %q, want %q", got.Type, "bulk")
	}
	if len(got.Sessions) != 2 {
		t.Fatalf("sessions len = %d, want 2", len(got.Sessions))
	}
	if got.Sessions[0].Slug != "cat" {
		t.Errorf("sessions[0].slug = %q, want %q", got.Sessions[0].Slug, "cat")
	}
}

func TestManifestValidate(t *testing.T) {
	tests := []struct {
		name    string
		m       Manifest
		wantErr bool
	}{
		{
			name:    "valid single",
			m:       Manifest{Version: 1, Type: "single", SessionUUID: "abc", ProjectPath: "-p"},
			wantErr: false,
		},
		{
			name:    "valid bulk",
			m:       Manifest{Version: 1, Type: "bulk", Sessions: []BulkSessionEntry{{SessionUUID: "x", ProjectPath: "-p"}}},
			wantErr: false,
		},
		{
			name:    "bad version",
			m:       Manifest{Version: 0, Type: "single"},
			wantErr: true,
		},
		{
			name:    "bad type",
			m:       Manifest{Version: 1, Type: "unknown"},
			wantErr: true,
		},
		{
			name:    "single missing uuid",
			m:       Manifest{Version: 1, Type: "single", ProjectPath: "-p"},
			wantErr: true,
		},
		{
			name:    "bulk no sessions",
			m:       Manifest{Version: 1, Type: "bulk"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `Manifest` type not defined

**Step 3: Write minimal implementation**

```go
// internal/transfer/manifest.go
package transfer

import (
	"fmt"
	"time"
)

// Manifest describes the contents of an export zip bundle.
type Manifest struct {
	Version     int                `json:"version"`
	Type        string             `json:"type"`
	ExportedAt  time.Time          `json:"exported_at"`
	ProjectPath string             `json:"project_path,omitempty"`
	SessionUUID string             `json:"session_uuid,omitempty"`
	Slug        string             `json:"slug,omitempty"`
	Sessions    []BulkSessionEntry `json:"sessions,omitempty"`
}

// BulkSessionEntry is one session in a bulk export manifest.
type BulkSessionEntry struct {
	ProjectPath string `json:"project_path"`
	SessionUUID string `json:"session_uuid"`
	Slug        string `json:"slug,omitempty"`
}

// Validate checks that the manifest has all required fields.
func (m *Manifest) Validate() error {
	if m.Version != 1 {
		return fmt.Errorf("unsupported manifest version: %d", m.Version)
	}
	switch m.Type {
	case "single":
		if m.SessionUUID == "" {
			return fmt.Errorf("single export manifest missing session_uuid")
		}
		if m.ProjectPath == "" {
			return fmt.Errorf("single export manifest missing project_path")
		}
	case "bulk":
		if len(m.Sessions) == 0 {
			return fmt.Errorf("bulk export manifest has no sessions")
		}
	default:
		return fmt.Errorf("unknown manifest type: %q", m.Type)
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```
git add internal/transfer/manifest.go internal/transfer/manifest_test.go
git commit -m "feat(transfer): add manifest types and validation"
```

---

### Task 2: Single session export

**Files:**
- Create: `internal/transfer/export.go`
- Create: `internal/transfer/export_test.go`

**Step 1: Write the failing test**

```go
// internal/transfer/export_test.go
package transfer

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/base-14/cicada/internal/model"
)

func TestExportSession(t *testing.T) {
	// Set up a fake claude dir with a session file
	claudeDir := t.TempDir()
	projectDir := filepath.Join(claudeDir, "projects", "-Users-r-work-proj")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	jsonlContent := []byte(`{"type":"user","message":{"role":"user","content":"hello"}}` + "\n")
	if err := os.WriteFile(filepath.Join(projectDir, "abc12345-uuid.jsonl"), jsonlContent, 0644); err != nil {
		t.Fatal(err)
	}

	meta := &model.SessionMeta{
		UUID:        "abc12345-uuid",
		Slug:        "happy-cat",
		ProjectPath: "-Users-r-work-proj",
	}

	outputDir := t.TempDir()
	filename, err := ExportSession(claudeDir, meta, outputDir)
	if err != nil {
		t.Fatalf("ExportSession: %v", err)
	}

	// Check filename format
	if filename != "happy-cat-abc12345.zip" {
		t.Errorf("filename = %q, want %q", filename, "happy-cat-abc12345.zip")
	}

	// Open and verify zip contents
	zipPath := filepath.Join(outputDir, filename)
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()

	// Should contain manifest.json and the JSONL file
	names := make(map[string]bool)
	for _, f := range r.File {
		names[f.Name] = true
	}
	if !names["manifest.json"] {
		t.Error("zip missing manifest.json")
	}
	if !names["abc12345-uuid.jsonl"] {
		t.Error("zip missing abc12345-uuid.jsonl")
	}

	// Verify manifest content
	for _, f := range r.File {
		if f.Name == "manifest.json" {
			rc, _ := f.Open()
			var m Manifest
			json.NewDecoder(rc).Decode(&m)
			rc.Close()
			if m.Type != "single" {
				t.Errorf("manifest type = %q, want %q", m.Type, "single")
			}
			if m.SessionUUID != "abc12345-uuid" {
				t.Errorf("manifest uuid = %q, want %q", m.SessionUUID, "abc12345-uuid")
			}
		}
	}
}

func TestExportSessionMissingFile(t *testing.T) {
	claudeDir := t.TempDir()
	meta := &model.SessionMeta{
		UUID:        "nonexistent",
		Slug:        "ghost",
		ProjectPath: "-Users-r-nope",
	}
	_, err := ExportSession(claudeDir, meta, t.TempDir())
	if err == nil {
		t.Error("expected error for missing JSONL file")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `ExportSession` not defined

**Step 3: Write minimal implementation**

```go
// internal/transfer/export.go
package transfer

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/base-14/cicada/internal/model"
)

// ExportSession exports a single session as a zip bundle.
// Returns the output filename (not full path).
func ExportSession(claudeDir string, meta *model.SessionMeta, outputDir string) (string, error) {
	jsonlPath := filepath.Join(claudeDir, "projects", meta.ProjectPath, meta.UUID+".jsonl")
	jsonlData, err := os.ReadFile(jsonlPath)
	if err != nil {
		return "", fmt.Errorf("read session file: %w", err)
	}

	manifest := Manifest{
		Version:     1,
		Type:        "single",
		ExportedAt:  time.Now().UTC(),
		ProjectPath: meta.ProjectPath,
		SessionUUID: meta.UUID,
		Slug:        meta.Slug,
	}

	uuidPrefix := meta.UUID
	if len(uuidPrefix) > 8 {
		uuidPrefix = uuidPrefix[:8]
	}
	slug := meta.Slug
	if slug == "" {
		slug = uuidPrefix
	}
	filename := fmt.Sprintf("%s-%s.zip", slug, uuidPrefix)

	zipPath := filepath.Join(outputDir, filename)
	if err := writeZip(zipPath, manifest, map[string][]byte{
		meta.UUID + ".jsonl": jsonlData,
	}); err != nil {
		return "", err
	}

	return filename, nil
}

// writeZip creates a zip file with a manifest and the given files.
func writeZip(zipPath string, manifest Manifest, files map[string][]byte) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// Write manifest
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	mw, err := w.Create("manifest.json")
	if err != nil {
		return fmt.Errorf("create manifest entry: %w", err)
	}
	if _, err := mw.Write(manifestData); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Write session files
	for name, data := range files {
		fw, err := w.Create(name)
		if err != nil {
			return fmt.Errorf("create entry %s: %w", name, err)
		}
		if _, err := fw.Write(data); err != nil {
			return fmt.Errorf("write entry %s: %w", name, err)
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```
git add internal/transfer/export.go internal/transfer/export_test.go
git commit -m "feat(transfer): add single session export"
```

---

### Task 3: Bulk export

**Files:**
- Modify: `internal/transfer/export.go`
- Modify: `internal/transfer/export_test.go`

**Step 1: Write the failing test**

Add to `export_test.go`:

```go
func TestExportAll(t *testing.T) {
	claudeDir := t.TempDir()

	// Create two projects with sessions
	for _, p := range []struct{ project, uuid string }{
		{"-Users-r-projA", "uuid1"},
		{"-Users-r-projA", "uuid2"},
		{"-Users-r-projB", "uuid3"},
	} {
		dir := filepath.Join(claudeDir, "projects", p.project)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, p.uuid+".jsonl"), []byte(`{"type":"user"}`+"\n"), 0644)
	}

	metas := []*model.SessionMeta{
		{UUID: "uuid1", Slug: "cat", ProjectPath: "-Users-r-projA"},
		{UUID: "uuid2", Slug: "dog", ProjectPath: "-Users-r-projA"},
		{UUID: "uuid3", Slug: "fox", ProjectPath: "-Users-r-projB"},
	}

	outputDir := t.TempDir()
	filename, err := ExportAll(claudeDir, metas, outputDir)
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}

	// Filename should start with "cicada-export-"
	if len(filename) < 14 || filename[:14] != "cicada-export-" {
		t.Errorf("filename = %q, want prefix %q", filename, "cicada-export-")
	}

	// Verify zip contents
	r, err := zip.OpenReader(filepath.Join(outputDir, filename))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()

	names := make(map[string]bool)
	for _, f := range r.File {
		names[f.Name] = true
	}

	if !names["manifest.json"] {
		t.Error("zip missing manifest.json")
	}
	if !names["-Users-r-projA/uuid1.jsonl"] {
		t.Error("zip missing -Users-r-projA/uuid1.jsonl")
	}
	if !names["-Users-r-projB/uuid3.jsonl"] {
		t.Error("zip missing -Users-r-projB/uuid3.jsonl")
	}

	// Verify manifest
	for _, f := range r.File {
		if f.Name == "manifest.json" {
			rc, _ := f.Open()
			var m Manifest
			json.NewDecoder(rc).Decode(&m)
			rc.Close()
			if m.Type != "bulk" {
				t.Errorf("manifest type = %q, want %q", m.Type, "bulk")
			}
			if len(m.Sessions) != 3 {
				t.Errorf("manifest sessions = %d, want 3", len(m.Sessions))
			}
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `ExportAll` not defined

**Step 3: Write minimal implementation**

Add to `export.go`:

```go
// ExportAll exports multiple sessions as a bulk zip bundle.
// Returns the output filename (not full path).
func ExportAll(claudeDir string, metas []*model.SessionMeta, outputDir string) (string, error) {
	files := make(map[string][]byte)
	var entries []BulkSessionEntry

	for _, meta := range metas {
		jsonlPath := filepath.Join(claudeDir, "projects", meta.ProjectPath, meta.UUID+".jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			continue // skip unreadable sessions
		}
		zipEntryPath := meta.ProjectPath + "/" + meta.UUID + ".jsonl"
		files[zipEntryPath] = data
		entries = append(entries, BulkSessionEntry{
			ProjectPath: meta.ProjectPath,
			SessionUUID: meta.UUID,
			Slug:        meta.Slug,
		})
	}

	if len(entries) == 0 {
		return "", fmt.Errorf("no sessions could be read")
	}

	manifest := Manifest{
		Version:    1,
		Type:       "bulk",
		ExportedAt: time.Now().UTC(),
		Sessions:   entries,
	}

	filename := fmt.Sprintf("cicada-export-%s.zip", time.Now().Format("2006-01-02"))
	zipPath := filepath.Join(outputDir, filename)
	if err := writeZip(zipPath, manifest, files); err != nil {
		return "", err
	}

	return filename, nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```
git add internal/transfer/export.go internal/transfer/export_test.go
git commit -m "feat(transfer): add bulk export"
```

---

### Task 4: Import — read bundle and place session

**Files:**
- Create: `internal/transfer/import.go`
- Create: `internal/transfer/import_test.go`

**Step 1: Write the failing test**

```go
// internal/transfer/import_test.go
package transfer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/base-14/cicada/internal/model"
)

func TestReadBundleSingle(t *testing.T) {
	// Create a zip via ExportSession, then read it back
	claudeDir := t.TempDir()
	projDir := filepath.Join(claudeDir, "projects", "-proj")
	os.MkdirAll(projDir, 0755)
	content := []byte(`{"type":"user","message":{"role":"user","content":"hi"}}` + "\n")
	os.WriteFile(filepath.Join(projDir, "test-uuid.jsonl"), content, 0644)

	meta := &model.SessionMeta{UUID: "test-uuid", Slug: "test", ProjectPath: "-proj"}
	outputDir := t.TempDir()
	filename, err := ExportSession(claudeDir, meta, outputDir)
	if err != nil {
		t.Fatal(err)
	}

	manifest, files, err := ReadBundle(filepath.Join(outputDir, filename))
	if err != nil {
		t.Fatalf("ReadBundle: %v", err)
	}

	if manifest.Type != "single" {
		t.Errorf("type = %q, want %q", manifest.Type, "single")
	}
	if manifest.SessionUUID != "test-uuid" {
		t.Errorf("uuid = %q, want %q", manifest.SessionUUID, "test-uuid")
	}
	if len(files) != 1 {
		t.Fatalf("files count = %d, want 1", len(files))
	}
	if _, ok := files["test-uuid.jsonl"]; !ok {
		t.Error("missing test-uuid.jsonl in files map")
	}
}

func TestReadBundleBulk(t *testing.T) {
	claudeDir := t.TempDir()
	for _, s := range []struct{ proj, uuid string }{
		{"-pA", "u1"}, {"-pB", "u2"},
	} {
		dir := filepath.Join(claudeDir, "projects", s.proj)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, s.uuid+".jsonl"), []byte(`{"type":"user"}`+"\n"), 0644)
	}

	metas := []*model.SessionMeta{
		{UUID: "u1", Slug: "a", ProjectPath: "-pA"},
		{UUID: "u2", Slug: "b", ProjectPath: "-pB"},
	}

	outputDir := t.TempDir()
	filename, _ := ExportAll(claudeDir, metas, outputDir)

	manifest, files, err := ReadBundle(filepath.Join(outputDir, filename))
	if err != nil {
		t.Fatalf("ReadBundle: %v", err)
	}

	if manifest.Type != "bulk" {
		t.Errorf("type = %q, want %q", manifest.Type, "bulk")
	}
	if len(files) != 2 {
		t.Errorf("files count = %d, want 2", len(files))
	}
}

func TestReadBundleInvalidPath(t *testing.T) {
	_, _, err := ReadBundle("/nonexistent/path.zip")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestPlaceSession(t *testing.T) {
	claudeDir := t.TempDir()
	os.MkdirAll(filepath.Join(claudeDir, "projects"), 0755)

	data := []byte(`{"type":"user","message":{"role":"user","content":"hi"}}` + "\n")
	err := PlaceSession(claudeDir, "-new-project", "new-uuid", data)
	if err != nil {
		t.Fatalf("PlaceSession: %v", err)
	}

	// Verify file exists
	placed := filepath.Join(claudeDir, "projects", "-new-project", "new-uuid.jsonl")
	got, err := os.ReadFile(placed)
	if err != nil {
		t.Fatalf("read placed file: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("placed content mismatch")
	}
}

func TestPlaceSessionCreatesDir(t *testing.T) {
	claudeDir := t.TempDir()
	// Don't create projects dir — PlaceSession should create it
	data := []byte(`{"type":"user"}` + "\n")
	err := PlaceSession(claudeDir, "-brand-new", "uuid1", data)
	if err != nil {
		t.Fatalf("PlaceSession: %v", err)
	}

	placed := filepath.Join(claudeDir, "projects", "-brand-new", "uuid1.jsonl")
	if _, err := os.Stat(placed); os.IsNotExist(err) {
		t.Error("placed file does not exist")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `ReadBundle`, `PlaceSession` not defined

**Step 3: Write minimal implementation**

```go
// internal/transfer/import.go
package transfer

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// ReadBundle reads a zip export bundle and returns the manifest and file contents.
// The files map is keyed by the path inside the zip (e.g., "uuid.jsonl" or "project/uuid.jsonl").
func ReadBundle(zipPath string) (*Manifest, map[string][]byte, error) {
	zipPath = ExpandPath(zipPath)

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	var manifest *Manifest
	files := make(map[string][]byte)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("open entry %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("read entry %s: %w", f.Name, err)
		}

		if f.Name == "manifest.json" {
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, nil, fmt.Errorf("parse manifest: %w", err)
			}
			if err := m.Validate(); err != nil {
				return nil, nil, fmt.Errorf("invalid manifest: %w", err)
			}
			manifest = &m
		} else {
			files[f.Name] = data
		}
	}

	if manifest == nil {
		return nil, nil, fmt.Errorf("zip missing manifest.json")
	}

	return manifest, files, nil
}

// PlaceSession writes a JSONL file to the correct location under claudeDir.
func PlaceSession(claudeDir, projectPath, uuid string, data []byte) error {
	targetDir := filepath.Join(claudeDir, "projects", projectPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	targetPath := filepath.Join(targetDir, uuid+".jsonl")
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```
git add internal/transfer/import.go internal/transfer/import_test.go
git commit -m "feat(transfer): add import bundle reading and session placement"
```

---

### Task 5: Round-trip integration test

**Files:**
- Modify: `internal/transfer/import_test.go`

**Step 1: Write the failing test**

Add to `import_test.go`:

```go
func TestRoundTripSingleExportImport(t *testing.T) {
	// Set up source
	srcDir := t.TempDir()
	projDir := filepath.Join(srcDir, "projects", "-src-proj")
	os.MkdirAll(projDir, 0755)
	original := []byte(`{"type":"user","message":{"role":"user","content":"hello world"}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}` + "\n")
	os.WriteFile(filepath.Join(projDir, "round-trip-uuid.jsonl"), original, 0644)

	// Export
	meta := &model.SessionMeta{UUID: "round-trip-uuid", Slug: "roundtrip", ProjectPath: "-src-proj"}
	exportDir := t.TempDir()
	filename, err := ExportSession(srcDir, meta, exportDir)
	if err != nil {
		t.Fatal(err)
	}

	// Import to a different project path
	dstDir := t.TempDir()
	os.MkdirAll(filepath.Join(dstDir, "projects"), 0755)
	manifest, files, err := ReadBundle(filepath.Join(exportDir, filename))
	if err != nil {
		t.Fatal(err)
	}

	// Place under a different project
	for name, data := range files {
		uuid := strings.TrimSuffix(name, ".jsonl")
		err := PlaceSession(dstDir, "-dst-proj", uuid, data)
		if err != nil {
			t.Fatalf("PlaceSession: %v", err)
		}
	}

	// Verify
	if manifest.SessionUUID != "round-trip-uuid" {
		t.Errorf("uuid = %q, want %q", manifest.SessionUUID, "round-trip-uuid")
	}

	placed, err := os.ReadFile(filepath.Join(dstDir, "projects", "-dst-proj", "round-trip-uuid.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if string(placed) != string(original) {
		t.Error("placed file content differs from original")
	}
}
```

**Step 2: Run test to verify it passes** (should pass since all pieces are implemented)

Run: `make test`
Expected: PASS

**Step 3: Commit**

```
git add internal/transfer/import_test.go
git commit -m "test(transfer): add round-trip integration test"
```

---

### Task 6: Add VisibleSessions method to SessionsView

**Files:**
- Modify: `internal/tui/views/sessions.go` (after line 92)
- Modify or create: `internal/tui/views/sessions_test.go`

**Step 1: Write the failing test**

```go
// In sessions_test.go, add:
func TestVisibleSessions(t *testing.T) {
	s := store.New()
	s.Add(&model.SessionMeta{UUID: "a", Slug: "alpha", ProjectPath: "-p", StartTime: time.Now()})
	s.Add(&model.SessionMeta{UUID: "b", Slug: "beta", ProjectPath: "-p", StartTime: time.Now().Add(time.Hour)})

	v := NewSessionsView(s)

	rows := v.VisibleSessions()
	if len(rows) != 2 {
		t.Errorf("VisibleSessions() = %d rows, want 2", len(rows))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `VisibleSessions` not defined

**Step 3: Write minimal implementation**

Add to `sessions.go` after `SelectedSession()` (after line 92):

```go
// VisibleSessions returns all currently visible (filtered) sessions.
func (v *SessionsView) VisibleSessions() []*model.SessionMeta {
	v.refreshRows()
	return v.rows
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```
git add internal/tui/views/sessions.go internal/tui/views/sessions_test.go
git commit -m "feat(tui): add VisibleSessions method to SessionsView"
```

---

### Task 7: TextInput component

**Files:**
- Create: `internal/tui/components/textinput.go`
- Create: `internal/tui/components/textinput_test.go`

**Step 1: Write the failing test**

```go
// internal/tui/components/textinput_test.go
package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTextInputTyping(t *testing.T) {
	ti := NewTextInput("Path: ")
	ti.Active = true

	ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	if ti.Value != "hi" {
		t.Errorf("Value = %q, want %q", ti.Value, "hi")
	}
}

func TestTextInputBackspace(t *testing.T) {
	ti := NewTextInput("Path: ")
	ti.Active = true
	ti.Value = "abc"

	ti.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if ti.Value != "ab" {
		t.Errorf("Value = %q after backspace, want %q", ti.Value, "ab")
	}
}

func TestTextInputEscDeactivates(t *testing.T) {
	ti := NewTextInput("Path: ")
	ti.Active = true
	ti.Value = "something"

	ti.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if ti.Active {
		t.Error("Active should be false after Esc")
	}
	if ti.Value != "" {
		t.Errorf("Value should be cleared after Esc, got %q", ti.Value)
	}
}

func TestTextInputEnterSignalsComplete(t *testing.T) {
	ti := NewTextInput("Path: ")
	ti.Active = true
	ti.Value = "/path/to/file.zip"

	handled := ti.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("Enter should be handled")
	}
	// Value preserved for caller to read, but Active set to false
	if ti.Active {
		t.Error("Active should be false after Enter")
	}
	if ti.Value != "/path/to/file.zip" {
		t.Errorf("Value should be preserved after Enter, got %q", ti.Value)
	}
}

func TestTextInputView(t *testing.T) {
	ti := NewTextInput("Path: ")
	ti.Active = true
	ti.Value = "test"

	view := ti.View()
	if view != "Path: test" {
		t.Errorf("View = %q, want %q", view, "Path: test")
	}
}

func TestTextInputInactiveIgnoresKeys(t *testing.T) {
	ti := NewTextInput("Path: ")
	// Not active
	handled := ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("inactive input should not handle keys")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `TextInput` not defined

**Step 3: Write minimal implementation**

```go
// internal/tui/components/textinput.go
package components

import tea "github.com/charmbracelet/bubbletea"

// TextInput is a simple one-line text input.
type TextInput struct {
	Active bool
	Value  string
	Prompt string
}

// NewTextInput creates a new inactive TextInput with the given prompt.
func NewTextInput(prompt string) *TextInput {
	return &TextInput{Prompt: prompt}
}

// Update handles key events. Returns true if the key was consumed.
func (ti *TextInput) Update(msg tea.KeyMsg) bool {
	if !ti.Active {
		return false
	}

	switch msg.Type {
	case tea.KeyEsc:
		ti.Active = false
		ti.Value = ""
		return true
	case tea.KeyEnter:
		ti.Active = false
		return true
	case tea.KeyBackspace:
		if len(ti.Value) > 0 {
			ti.Value = ti.Value[:len(ti.Value)-1]
		}
		return true
	case tea.KeyRunes:
		ti.Value += string(msg.Runes)
		return true
	}

	return true
}

// View renders the text input.
func (ti *TextInput) View() string {
	if !ti.Active {
		return ""
	}
	return ti.Prompt + ti.Value
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```
git add internal/tui/components/textinput.go internal/tui/components/textinput_test.go
git commit -m "feat(tui): add TextInput component"
```

---

### Task 8: ProjectPicker component

**Files:**
- Create: `internal/tui/components/projectpicker.go`
- Create: `internal/tui/components/projectpicker_test.go`

**Step 1: Write the failing test**

```go
// internal/tui/components/projectpicker_test.go
package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestProjectPickerNavigation(t *testing.T) {
	pp := NewProjectPicker([]string{"-projA", "-projB", "-projC"}, "-projA")
	pp.Active = true

	if pp.Selected != 0 {
		t.Errorf("initial Selected = %d, want 0", pp.Selected)
	}

	pp.Update(tea.KeyMsg{Type: tea.KeyDown})
	if pp.Selected != 1 {
		t.Errorf("Selected after down = %d, want 1", pp.Selected)
	}

	pp.Update(tea.KeyMsg{Type: tea.KeyUp})
	if pp.Selected != 0 {
		t.Errorf("Selected after up = %d, want 0", pp.Selected)
	}

	// Can't go above 0
	pp.Update(tea.KeyMsg{Type: tea.KeyUp})
	if pp.Selected != 0 {
		t.Errorf("Selected should stay at 0, got %d", pp.Selected)
	}
}

func TestProjectPickerSelectsProject(t *testing.T) {
	pp := NewProjectPicker([]string{"-projA", "-projB"}, "-projA")
	pp.Active = true
	pp.Selected = 1

	got := pp.SelectedProject()
	if got != "-projB" {
		t.Errorf("SelectedProject = %q, want %q", got, "-projB")
	}
}

func TestProjectPickerCustomInput(t *testing.T) {
	pp := NewProjectPicker([]string{"-projA"}, "")
	pp.Active = true

	// Navigate to custom input (last entry)
	pp.Selected = len(pp.Projects) // custom entry is after all projects
	pp.EnteringCustom = true
	pp.CustomInput = "-my-custom-proj"

	got := pp.SelectedProject()
	if got != "-my-custom-proj" {
		t.Errorf("SelectedProject = %q, want %q", got, "-my-custom-proj")
	}
}

func TestProjectPickerTabTogglesCustom(t *testing.T) {
	pp := NewProjectPicker([]string{"-projA"}, "")
	pp.Active = true

	pp.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !pp.EnteringCustom {
		t.Error("Tab should toggle EnteringCustom to true")
	}

	pp.Update(tea.KeyMsg{Type: tea.KeyTab})
	if pp.EnteringCustom {
		t.Error("Tab should toggle EnteringCustom back to false")
	}
}

func TestProjectPickerCustomTyping(t *testing.T) {
	pp := NewProjectPicker([]string{}, "")
	pp.Active = true
	pp.EnteringCustom = true

	pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	if pp.CustomInput != "ab" {
		t.Errorf("CustomInput = %q, want %q", pp.CustomInput, "ab")
	}
}

func TestProjectPickerEscCancels(t *testing.T) {
	pp := NewProjectPicker([]string{"-projA"}, "")
	pp.Active = true

	pp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if pp.Active {
		t.Error("Esc should deactivate picker")
	}
}

func TestProjectPickerViewHighlightsOriginal(t *testing.T) {
	pp := NewProjectPicker([]string{"-projA", "-projB"}, "-projA")
	pp.Active = true

	view := pp.View(60)
	if !strings.Contains(view, "-projA") {
		t.Error("view should contain -projA")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `ProjectPicker` not defined

**Step 3: Write minimal implementation**

```go
// internal/tui/components/projectpicker.go
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ProjectPicker shows a list of existing projects for import target selection.
type ProjectPicker struct {
	Active         bool
	Projects       []string
	Selected       int
	CustomInput    string
	EnteringCustom bool
	OriginalPath   string
}

// NewProjectPicker creates a new project picker.
func NewProjectPicker(projects []string, originalPath string) *ProjectPicker {
	return &ProjectPicker{
		Projects:     projects,
		OriginalPath: originalPath,
	}
}

// Update handles key events. Returns true if the key was consumed.
func (pp *ProjectPicker) Update(msg tea.KeyMsg) bool {
	if !pp.Active {
		return false
	}

	if pp.EnteringCustom {
		switch msg.Type {
		case tea.KeyEsc:
			pp.Active = false
			return true
		case tea.KeyTab:
			pp.EnteringCustom = false
			return true
		case tea.KeyEnter:
			pp.Active = false
			return true
		case tea.KeyBackspace:
			if len(pp.CustomInput) > 0 {
				pp.CustomInput = pp.CustomInput[:len(pp.CustomInput)-1]
			}
			return true
		case tea.KeyRunes:
			pp.CustomInput += string(msg.Runes)
			return true
		}
		return true
	}

	switch msg.Type {
	case tea.KeyEsc:
		pp.Active = false
		return true
	case tea.KeyTab:
		pp.EnteringCustom = true
		return true
	case tea.KeyUp:
		if pp.Selected > 0 {
			pp.Selected--
		}
		return true
	case tea.KeyDown:
		maxIdx := len(pp.Projects) - 1
		if maxIdx < 0 {
			maxIdx = 0
		}
		if pp.Selected < maxIdx {
			pp.Selected++
		}
		return true
	case tea.KeyEnter:
		pp.Active = false
		return true
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			maxIdx := len(pp.Projects) - 1
			if pp.Selected < maxIdx {
				pp.Selected++
			}
			return true
		case "k":
			if pp.Selected > 0 {
				pp.Selected--
			}
			return true
		}
	}

	return true
}

// SelectedProject returns the currently selected project path.
func (pp *ProjectPicker) SelectedProject() string {
	if pp.EnteringCustom {
		return pp.CustomInput
	}
	if len(pp.Projects) == 0 {
		return pp.CustomInput
	}
	if pp.Selected >= len(pp.Projects) {
		return pp.CustomInput
	}
	return pp.Projects[pp.Selected]
}

// View renders the project picker.
func (pp *ProjectPicker) View(width int) string {
	if !pp.Active {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n  Select target project:\n\n")

	for i, proj := range pp.Projects {
		prefix := "  "
		if i == pp.Selected && !pp.EnteringCustom {
			prefix = "> "
		}
		suffix := ""
		if proj == pp.OriginalPath {
			suffix = " (original)"
		}
		b.WriteString(fmt.Sprintf("  %s%s%s\n", prefix, proj, suffix))
	}

	b.WriteString("\n")
	customPrefix := "  "
	if pp.EnteringCustom {
		customPrefix = "> "
	}
	b.WriteString(fmt.Sprintf("  %s[Custom] %s\n", customPrefix, pp.CustomInput))
	b.WriteString("\n  Tab: toggle custom  Enter: confirm  Esc: cancel\n")

	return b.String()
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```
git add internal/tui/components/projectpicker.go internal/tui/components/projectpicker_test.go
git commit -m "feat(tui): add ProjectPicker component"
```

---

### Task 9: Wire export keybindings into App

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Write the failing test**

Add to existing `internal/tui/app_test.go` (or create if needed):

```go
func TestExportKeyBinding(t *testing.T) {
	s := store.New()
	s.Add(&model.SessionMeta{UUID: "test-uuid", Slug: "test", ProjectPath: "-proj"})

	// Create fake JSONL file
	projDir := filepath.Join(t.TempDir(), "-proj")
	os.MkdirAll(projDir, 0755)
	os.WriteFile(filepath.Join(projDir, "test-uuid.jsonl"), []byte(`{"type":"user"}`+"\n"), 0644)

	app := NewApp(s, t.TempDir())
	app.activeTab = 2 // Sessions tab
	app.width = 120
	app.height = 40

	// Press 'e' — should return a command (not nil)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	_, cmd := app.Update(msg)
	if cmd == nil {
		t.Error("pressing 'e' on sessions tab should return a command")
	}
}
```

Note: This test may need adjustment based on how `NewApp` works with the projectsDir. The key thing is verifying the keybinding triggers the export command.

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — no handler for 'e' key

**Step 3: Write minimal implementation**

Modify `internal/tui/app.go`:

1. Add imports:
```go
import (
	// ... existing imports ...
	"os"
	"github.com/base-14/cicada/internal/transfer"
	"github.com/base-14/cicada/internal/tui/components"
)
```

2. Add new message types after `clearNotificationMsg` (after line 35):
```go
// ExportResultMsg is sent after an export attempt.
type ExportResultMsg struct {
	Err      error
	Filename string
	Count    int
}

// ImportReadyMsg is sent after reading a zip bundle.
type ImportReadyMsg struct {
	Err      error
	Manifest *transfer.Manifest
	Files    map[string][]byte
}

// ImportResultMsg is sent after placing imported sessions.
type ImportResultMsg struct {
	Err   error
	Count int
}
```

3. Add fields to App struct (after line 61):
```go
	importInput       *components.TextInput
	projectPicker     *components.ProjectPicker
	importManifest    *transfer.Manifest
	importFiles       map[string][]byte
```

4. Add keybinding cases in the `tea.KeyRunes` switch inside `Update()`, after the "y" case (around line 239):
```go
case "e":
	if a.activeTab == 2 && !a.sessionsView.FilterActive() {
		session := a.sessionsView.SelectedSession()
		if session != nil {
			return a, a.exportSessionCmd(session)
		}
	}
	return a, nil
case "E":
	if a.activeTab == 2 && !a.sessionsView.FilterActive() {
		sessions := a.sessionsView.VisibleSessions()
		if len(sessions) > 0 {
			return a, a.exportAllCmd(sessions)
		}
	}
	return a, nil
```

5. Add ExportResultMsg handler after `CopyResultMsg` handler (after line 287):
```go
case ExportResultMsg:
	if msg.Err != nil {
		a.notification = "Export failed: " + msg.Err.Error()
	} else if msg.Count > 0 {
		a.notification = fmt.Sprintf("Exported %d sessions to %s", msg.Count, msg.Filename)
	} else {
		a.notification = "Exported to " + msg.Filename
	}
	a.notificationTime = time.Now()
	return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
```

6. Add helper methods:
```go
func (a App) exportSessionCmd(meta *model.SessionMeta) tea.Cmd {
	projectsDir := a.projectsDir
	return func() tea.Msg {
		claudeDir := filepath.Dir(projectsDir)
		wd, err := os.Getwd()
		if err != nil {
			return ExportResultMsg{Err: err}
		}
		filename, err := transfer.ExportSession(claudeDir, meta, wd)
		return ExportResultMsg{Err: err, Filename: filename}
	}
}

func (a App) exportAllCmd(metas []*model.SessionMeta) tea.Cmd {
	projectsDir := a.projectsDir
	return func() tea.Msg {
		claudeDir := filepath.Dir(projectsDir)
		wd, err := os.Getwd()
		if err != nil {
			return ExportResultMsg{Err: err}
		}
		filename, err := transfer.ExportAll(claudeDir, metas, wd)
		return ExportResultMsg{Err: err, Filename: filename, Count: len(metas)}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```
git add internal/tui/app.go
git commit -m "feat(tui): wire export keybindings e/E on sessions tab"
```

---

### Task 10: Wire import keybinding and input flow

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Add import input handling**

In `Update()`, add handling for import-related UI states. Before the existing `tea.KeyMsg` switch (after the project detail block around line 132), add:

```go
// Import flow: text input active
if a.importInput != nil && a.importInput.Active {
	switch msg.Type {
	case tea.KeyEnter:
		a.importInput.Active = false
		zipPath := a.importInput.Value
		a.importInput = nil
		return a, a.readImportBundleCmd(zipPath)
	case tea.KeyEsc:
		a.importInput = nil
		return a, nil
	default:
		a.importInput.Update(msg)
		return a, nil
	}
}

// Import flow: project picker active
if a.projectPicker != nil && a.projectPicker.Active {
	switch msg.Type {
	case tea.KeyEnter:
		selectedProject := a.projectPicker.SelectedProject()
		a.projectPicker = nil
		return a, a.placeImportCmd(selectedProject)
	case tea.KeyEsc:
		a.projectPicker = nil
		a.importManifest = nil
		a.importFiles = nil
		return a, nil
	default:
		a.projectPicker.Update(msg)
		return a, nil
	}
}
```

Add the "i" case in `tea.KeyRunes` (next to "e" and "E"):

```go
case "i":
	if a.activeTab == 2 && !a.sessionsView.FilterActive() {
		a.importInput = components.NewTextInput("Import zip path: ")
		a.importInput.Active = true
		return a, nil
	}
	return a, nil
```

Add message handlers:

```go
case ImportReadyMsg:
	if msg.Err != nil {
		a.notification = "Import failed: " + msg.Err.Error()
		a.notificationTime = time.Now()
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})
	}
	a.importManifest = msg.Manifest
	a.importFiles = msg.Files
	if msg.Manifest.Type == "bulk" {
		// Bulk: place directly using original paths
		return a, a.placeBulkImportCmd()
	}
	// Single: show project picker
	projects := a.store.Projects()
	a.projectPicker = components.NewProjectPicker(projects, msg.Manifest.ProjectPath)
	a.projectPicker.Active = true
	return a, nil

case ImportResultMsg:
	if msg.Err != nil {
		a.notification = "Import failed: " + msg.Err.Error()
	} else if msg.Count > 1 {
		a.notification = fmt.Sprintf("Imported %d sessions", msg.Count)
	} else {
		a.notification = "Imported session"
	}
	a.importManifest = nil
	a.importFiles = nil
	a.notificationTime = time.Now()
	return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
```

Add helper methods:

```go
func (a App) readImportBundleCmd(zipPath string) tea.Cmd {
	return func() tea.Msg {
		manifest, files, err := transfer.ReadBundle(zipPath)
		return ImportReadyMsg{Err: err, Manifest: manifest, Files: files}
	}
}

func (a App) placeImportCmd(projectPath string) tea.Cmd {
	projectsDir := a.projectsDir
	importManifest := a.importManifest
	importFiles := a.importFiles
	return func() tea.Msg {
		claudeDir := filepath.Dir(projectsDir)
		for name, data := range importFiles {
			uuid := strings.TrimSuffix(filepath.Base(name), ".jsonl")
			if err := transfer.PlaceSession(claudeDir, projectPath, uuid, data); err != nil {
				return ImportResultMsg{Err: err}
			}
		}
		_ = importManifest // used for context, placement done
		return ImportResultMsg{Count: len(importFiles)}
	}
}

func (a App) placeBulkImportCmd() tea.Cmd {
	projectsDir := a.projectsDir
	importManifest := a.importManifest
	importFiles := a.importFiles
	return func() tea.Msg {
		claudeDir := filepath.Dir(projectsDir)
		count := 0
		for _, entry := range importManifest.Sessions {
			zipKey := entry.ProjectPath + "/" + entry.SessionUUID + ".jsonl"
			data, ok := importFiles[zipKey]
			if !ok {
				continue
			}
			if err := transfer.PlaceSession(claudeDir, entry.ProjectPath, entry.SessionUUID, data); err != nil {
				return ImportResultMsg{Err: err}
			}
			count++
		}
		return ImportResultMsg{Count: count}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 3: Commit**

```
git add internal/tui/app.go
git commit -m "feat(tui): wire import keybinding with text input and project picker"
```

---

### Task 11: Render import overlay in View

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Update renderContent to show import UI**

In `renderContent()` (around line 415), add before the existing detail view checks:

```go
// Import input overlay
if a.importInput != nil && a.importInput.Active {
	return "\n  " + a.importInput.View()
}

// Project picker overlay
if a.projectPicker != nil && a.projectPicker.Active {
	return a.projectPicker.View(a.width)
}
```

**Step 2: Run, build, and verify**

Run: `make build && make test`
Expected: PASS, builds cleanly

**Step 3: Commit**

```
git add internal/tui/app.go
git commit -m "feat(tui): render import input and project picker overlays"
```

---

### Task 12: Update help text and status bar

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Update renderHelpOverlay**

In `renderHelpOverlay()` (line 326-358), add after the Clipboard section:

```go
  Export / Import
    e              Export selected session
    E              Export all visible sessions
    i              Import session(s) from zip
```

**Step 2: Update renderStatusBar**

In `renderStatusBar()` (line 458-461), change the sessions tab hint:

```go
if a.activeTab == 2 || (a.showingDetail && a.detailView != nil) {
	help = "e export  i import  y copy  ? help  q quit"
}
```

**Step 3: Run, build, and verify**

Run: `make build && make test && make lint`
Expected: PASS

**Step 4: Commit**

```
git add internal/tui/app.go
git commit -m "feat(tui): update help text and status bar with export/import hints"
```

---

### Task 13: Re-index imported sessions into the store

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Update import result handlers to re-scan imported files**

After `PlaceSession` succeeds, parse the placed JSONL and add to the in-memory store so the session appears immediately without restarting.

Modify `placeImportCmd` and `placeBulkImportCmd` to also parse and add to store. Since store access needs to happen from the main goroutine (or be thread-safe, which it already is via mutex), we can parse in the command and send the metas back.

Add a new message type:

```go
type ImportResultMsg struct {
	Err   error
	Count int
	Metas []*model.SessionMeta
}
```

Update `placeImportCmd`:

```go
func (a App) placeImportCmd(projectPath string) tea.Cmd {
	projectsDir := a.projectsDir
	importFiles := a.importFiles
	return func() tea.Msg {
		claudeDir := filepath.Dir(projectsDir)
		var metas []*model.SessionMeta
		for name, data := range importFiles {
			uuid := strings.TrimSuffix(filepath.Base(name), ".jsonl")
			if err := transfer.PlaceSession(claudeDir, projectPath, uuid, data); err != nil {
				return ImportResultMsg{Err: err}
			}
			// Parse and extract meta for store
			jsonlPath := filepath.Join(claudeDir, "projects", projectPath, uuid+".jsonl")
			messages, err := parser.ReadSessionFile(jsonlPath)
			if err == nil {
				meta := parser.ExtractSessionMeta(messages, projectPath, uuid+".jsonl")
				metas = append(metas, meta)
			}
		}
		return ImportResultMsg{Count: len(importFiles), Metas: metas}
	}
}
```

Update `ImportResultMsg` handler to add metas to store:

```go
case ImportResultMsg:
	if msg.Err != nil {
		a.notification = "Import failed: " + msg.Err.Error()
	} else {
		for _, meta := range msg.Metas {
			a.store.Add(meta)
		}
		if msg.Count > 1 {
			a.notification = fmt.Sprintf("Imported %d sessions", msg.Count)
		} else {
			a.notification = "Imported session"
		}
	}
	a.importManifest = nil
	a.importFiles = nil
	a.notificationTime = time.Now()
	return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
```

Similarly update `placeBulkImportCmd`.

**Step 2: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 3: Commit**

```
git add internal/tui/app.go
git commit -m "feat(tui): re-index imported sessions into store for immediate visibility"
```

---

### Task 14: Final verification

**Step 1: Build and lint**

Run: `make build && make lint`
Expected: Clean build, no lint errors

**Step 2: Run all tests**

Run: `make test`
Expected: All PASS

**Step 3: Manual smoke test**

Run: `make run`

1. Navigate to Sessions tab (press `3`)
2. Press `e` on a session — verify zip appears in CWD
3. Press `E` — verify bulk zip appears
4. Press `i`, type the zip path, Enter — verify import flow works
5. Check `?` help shows export/import keybindings
6. Check status bar shows `e export  i import  y copy  ? help  q quit`

**Step 4: Final commit if any fixes needed**

```
git add -A
git commit -m "fix: address smoke test findings for export/import"
```

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

// ExportSession exports a single session as a zip file containing a manifest
// and the session's JSONL data. It returns the output filename (not the full path).
func ExportSession(claudeDir string, meta *model.SessionMeta, outputDir string) (string, error) {
	jsonlPath := filepath.Join(claudeDir, "projects", meta.ProjectPath, meta.UUID+".jsonl")
	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		return "", fmt.Errorf("reading session JSONL: %w", err)
	}

	manifest := Manifest{
		Version:     1,
		Type:        "single",
		ExportedAt:  time.Now(),
		ProjectPath: meta.ProjectPath,
		SessionUUID: meta.UUID,
		Slug:        meta.Slug,
	}

	files := map[string][]byte{
		meta.UUID + ".jsonl": data,
	}

	filename := meta.Slug + "-" + meta.UUID[:8] + ".zip"
	zipPath := filepath.Join(outputDir, filename)

	if err := writeZip(zipPath, manifest, files); err != nil {
		return "", err
	}

	return filename, nil
}

// ExportAll exports multiple sessions into a single zip file. Sessions that
// cannot be read are skipped. Returns an error if no sessions could be read.
func ExportAll(claudeDir string, metas []*model.SessionMeta, outputDir string) (string, error) {
	files := make(map[string][]byte)
	var entries []BulkSessionEntry

	for _, meta := range metas {
		jsonlPath := filepath.Join(claudeDir, "projects", meta.ProjectPath, meta.UUID+".jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			continue // skip unreadable sessions
		}

		key := meta.ProjectPath + "/" + meta.UUID + ".jsonl"
		files[key] = data
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
		ExportedAt: time.Now(),
		Sessions:   entries,
	}

	filename := "cicada-export-" + time.Now().Format("2006-01-02") + ".zip"
	zipPath := filepath.Join(outputDir, filename)

	if err := writeZip(zipPath, manifest, files); err != nil {
		return "", err
	}

	return filename, nil
}

// writeZip creates a zip file containing the manifest and the provided files.
func writeZip(zipPath string, manifest Manifest, files map[string][]byte) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("creating zip file: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// Write manifest.json.
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	mw, err := w.Create("manifest.json")
	if err != nil {
		return fmt.Errorf("creating manifest entry: %w", err)
	}
	if _, err := mw.Write(manifestData); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	// Write data files.
	for name, data := range files {
		fw, err := w.Create(name)
		if err != nil {
			return fmt.Errorf("creating zip entry %q: %w", name, err)
		}
		if _, err := fw.Write(data); err != nil {
			return fmt.Errorf("writing zip entry %q: %w", name, err)
		}
	}

	return nil
}

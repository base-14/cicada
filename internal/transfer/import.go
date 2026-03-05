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
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// ReadBundle reads a zip export bundle and returns the manifest and file contents.
// The files map is keyed by the path inside the zip (e.g., "uuid.jsonl" or "project/uuid.jsonl").
func ReadBundle(zipPath string) (*Manifest, map[string][]byte, error) {
	zipPath = ExpandPath(zipPath)

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, nil, fmt.Errorf("opening zip bundle: %w", err)
	}
	defer r.Close()

	var manifest *Manifest
	files := make(map[string][]byte)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("opening zip entry %q: %w", f.Name, err)
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("reading zip entry %q: %w", f.Name, err)
		}

		if f.Name == "manifest.json" {
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, nil, fmt.Errorf("parsing manifest: %w", err)
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
		return nil, nil, fmt.Errorf("bundle missing manifest.json")
	}

	return manifest, files, nil
}

// PlaceSession writes a JSONL file to the correct location under claudeDir.
func PlaceSession(claudeDir, projectPath, uuid string, data []byte) error {
	dir := filepath.Join(claudeDir, "projects", projectPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	dest := filepath.Join(dir, uuid+".jsonl")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

package transfer

import (
	"fmt"
	"time"
)

// Manifest describes the contents of an exported session bundle.
type Manifest struct {
	Version     int                `json:"version"`
	Type        string             `json:"type"`
	ExportedAt  time.Time          `json:"exported_at"`
	ProjectPath string             `json:"project_path,omitempty"`
	SessionUUID string             `json:"session_uuid,omitempty"`
	Slug        string             `json:"slug,omitempty"`
	Sessions    []BulkSessionEntry `json:"sessions,omitempty"`
}

// BulkSessionEntry represents a single session within a bulk export.
type BulkSessionEntry struct {
	ProjectPath string `json:"project_path"`
	SessionUUID string `json:"session_uuid"`
	Slug        string `json:"slug,omitempty"`
}

// Validate checks that the manifest is well-formed.
func (m *Manifest) Validate() error {
	if m.Version != 1 {
		return fmt.Errorf("unsupported manifest version: %d", m.Version)
	}

	switch m.Type {
	case "single":
		if m.SessionUUID == "" {
			return fmt.Errorf("single export requires session_uuid")
		}
		if m.ProjectPath == "" {
			return fmt.Errorf("single export requires project_path")
		}
	case "bulk":
		if len(m.Sessions) == 0 {
			return fmt.Errorf("bulk export requires at least one session entry")
		}
	default:
		return fmt.Errorf("unsupported manifest type: %q", m.Type)
	}

	return nil
}

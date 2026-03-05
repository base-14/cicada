package transfer

import (
	"encoding/json"
	"testing"
	"time"
)

func TestManifestRoundTrip_Single(t *testing.T) {
	ts := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	m := Manifest{
		Version:     1,
		Type:        "single",
		ExportedAt:  ts,
		ProjectPath: "/home/user/project",
		SessionUUID: "abc-123",
		Slug:        "my-session",
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Manifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Version != m.Version {
		t.Errorf("Version = %d, want %d", got.Version, m.Version)
	}
	if got.Type != m.Type {
		t.Errorf("Type = %q, want %q", got.Type, m.Type)
	}
	if !got.ExportedAt.Equal(m.ExportedAt) {
		t.Errorf("ExportedAt = %v, want %v", got.ExportedAt, m.ExportedAt)
	}
	if got.ProjectPath != m.ProjectPath {
		t.Errorf("ProjectPath = %q, want %q", got.ProjectPath, m.ProjectPath)
	}
	if got.SessionUUID != m.SessionUUID {
		t.Errorf("SessionUUID = %q, want %q", got.SessionUUID, m.SessionUUID)
	}
	if got.Slug != m.Slug {
		t.Errorf("Slug = %q, want %q", got.Slug, m.Slug)
	}
}

func TestManifestRoundTrip_Bulk(t *testing.T) {
	ts := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	m := Manifest{
		Version:    1,
		Type:       "bulk",
		ExportedAt: ts,
		Sessions: []BulkSessionEntry{
			{ProjectPath: "/home/user/p1", SessionUUID: "uuid-1", Slug: "slug-1"},
			{ProjectPath: "/home/user/p2", SessionUUID: "uuid-2"},
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

	if len(got.Sessions) != 2 {
		t.Fatalf("Sessions len = %d, want 2", len(got.Sessions))
	}
	if got.Sessions[0].SessionUUID != "uuid-1" {
		t.Errorf("Sessions[0].SessionUUID = %q, want %q", got.Sessions[0].SessionUUID, "uuid-1")
	}
	if got.Sessions[1].Slug != "" {
		t.Errorf("Sessions[1].Slug = %q, want empty", got.Sessions[1].Slug)
	}
}

func TestManifestValidate(t *testing.T) {
	tests := []struct {
		name    string
		m       Manifest
		wantErr bool
	}{
		{
			name: "valid single",
			m: Manifest{
				Version:     1,
				Type:        "single",
				ProjectPath: "/project",
				SessionUUID: "uuid-1",
			},
			wantErr: false,
		},
		{
			name: "valid bulk",
			m: Manifest{
				Version: 1,
				Type:    "bulk",
				Sessions: []BulkSessionEntry{
					{ProjectPath: "/p", SessionUUID: "u"},
				},
			},
			wantErr: false,
		},
		{
			name: "bad version",
			m: Manifest{
				Version:     2,
				Type:        "single",
				ProjectPath: "/project",
				SessionUUID: "uuid-1",
			},
			wantErr: true,
		},
		{
			name: "bad type",
			m: Manifest{
				Version:     1,
				Type:        "multi",
				ProjectPath: "/project",
				SessionUUID: "uuid-1",
			},
			wantErr: true,
		},
		{
			name: "single missing uuid",
			m: Manifest{
				Version:     1,
				Type:        "single",
				ProjectPath: "/project",
			},
			wantErr: true,
		},
		{
			name: "single missing project path",
			m: Manifest{
				Version:     1,
				Type:        "single",
				SessionUUID: "uuid-1",
			},
			wantErr: true,
		},
		{
			name: "bulk no sessions",
			m: Manifest{
				Version: 1,
				Type:    "bulk",
			},
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

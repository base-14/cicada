package model

import (
	"testing"
	"time"
)

func TestSessionMeta_Duration(t *testing.T) {
	now := time.Now()
	meta := SessionMeta{
		UUID:      "test-uuid",
		Slug:      "test-slug",
		StartTime: now.Add(-10 * time.Minute),
		EndTime:   now,
	}

	if meta.UUID != "test-uuid" {
		t.Errorf("expected UUID 'test-uuid', got %q", meta.UUID)
	}
	if meta.Slug != "test-slug" {
		t.Errorf("expected Slug 'test-slug', got %q", meta.Slug)
	}

	expectedDuration := 10 * time.Minute
	if meta.EndTime.Sub(meta.StartTime) != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, meta.EndTime.Sub(meta.StartTime))
	}
}

func TestSessionMeta_Defaults(t *testing.T) {
	meta := SessionMeta{}

	if meta.ToolUsage != nil {
		t.Error("expected nil ToolUsage for zero value")
	}
	if meta.MessageCount != 0 {
		t.Errorf("expected 0 MessageCount, got %d", meta.MessageCount)
	}
}

package tui

import (
	"testing"
)

func TestThemeColors(t *testing.T) {
	theme := DefaultTheme()

	if theme.Primary == "" {
		t.Error("expected non-empty Primary color")
	}
	if theme.Secondary == "" {
		t.Error("expected non-empty Secondary color")
	}
	if theme.Muted == "" {
		t.Error("expected non-empty Muted color")
	}
}

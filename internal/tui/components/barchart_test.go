package components

import (
	"strings"
	"testing"
)

func TestBarChart_Render(t *testing.T) {
	items := []BarItem{
		{Label: "Read", Value: 100},
		{Label: "Edit", Value: 50},
		{Label: "Bash", Value: 25},
	}

	result := BarChart(items, 40)
	lines := strings.Split(result, "\n")

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestBarChart_Empty(t *testing.T) {
	result := BarChart(nil, 40)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

package components

import (
	"strings"
	"testing"
)

func TestSparkline_Render(t *testing.T) {
	data := []int{1, 3, 5, 2, 7, 4, 6}
	result := Sparkline(data, 20)

	if result == "" {
		t.Error("expected non-empty sparkline")
	}
	if len([]rune(result)) > 20 {
		t.Errorf("sparkline too long: %d runes (max 20)", len([]rune(result)))
	}
}

func TestSparkline_Empty(t *testing.T) {
	result := Sparkline(nil, 20)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}
}

func TestSparkline_SingleValue(t *testing.T) {
	result := Sparkline([]int{5}, 20)
	if result == "" {
		t.Error("expected non-empty sparkline for single value")
	}
}

func TestBarGraph_Render(t *testing.T) {
	data := []int{1, 3, 0, 5, 2, 7, 4}
	result := BarGraph(data, 30, 6)
	if result == "" {
		t.Error("expected non-empty bar graph")
	}
	lines := strings.Split(result, "\n")
	// 6 data rows + 1 axis + 1 label = 8
	if len(lines) < 8 {
		t.Errorf("expected at least 8 lines, got %d", len(lines))
	}
	// Should contain bar blocks
	if !strings.Contains(result, "█") {
		t.Error("expected bar block characters")
	}
}

func TestBarGraph_Empty(t *testing.T) {
	result := BarGraph(nil, 30, 6)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}
}

func TestBarGraph_AllZero(t *testing.T) {
	data := []int{0, 0, 0, 0, 0}
	result := BarGraph(data, 20, 4)
	if result == "" {
		t.Error("expected non-empty graph even with all zeros")
	}
	// Should not contain bar blocks since all values are 0
	lines := strings.Split(result, "\n")
	for _, line := range lines[:4] { // data rows only
		if strings.Contains(line, "█") {
			t.Error("expected no bars for all-zero data")
		}
	}
}

package components

import "testing"

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

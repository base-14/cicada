package components

import (
	"strings"
	"testing"
)

func TestHeatmap_NonEmpty(t *testing.T) {
	var data [7][24]int
	data[0][9] = 5
	data[3][14] = 10
	data[6][22] = 3

	result := Heatmap(data)
	if result == "" {
		t.Error("expected non-empty heatmap")
	}
}

func TestHeatmap_LineCount(t *testing.T) {
	var data [7][24]int
	data[1][12] = 7

	result := Heatmap(data)
	lines := strings.Split(result, "\n")
	// 1 header line + 7 data rows = 8
	if len(lines) != 8 {
		t.Errorf("expected 8 lines (1 header + 7 days), got %d", len(lines))
	}
}

func TestHeatmap_AllZeros(t *testing.T) {
	var data [7][24]int
	result := Heatmap(data)
	if result == "" {
		t.Error("expected non-empty heatmap even with all zeros")
	}
	// Should have 8 lines
	lines := strings.Split(result, "\n")
	if len(lines) != 8 {
		t.Errorf("expected 8 lines, got %d", len(lines))
	}
}

func TestHeatmap_ContainsDayLabels(t *testing.T) {
	var data [7][24]int
	result := Heatmap(data)
	if !strings.Contains(result, "Mon") {
		t.Error("expected 'Mon' day label")
	}
	if !strings.Contains(result, "Sun") {
		t.Error("expected 'Sun' day label")
	}
}

func TestHeatmap_IntensityBlocks(t *testing.T) {
	var data [7][24]int
	data[0][0] = 1
	data[0][1] = 10
	data[0][2] = 50
	data[0][3] = 100

	result := Heatmap(data)
	// Should contain block characters for non-zero values
	if !strings.Contains(result, "██") {
		t.Error("expected monochrome block characters in output")
	}
	// Zero cells should show dots
	if !strings.Contains(result, "··") {
		t.Error("expected dot markers for zero cells")
	}
}

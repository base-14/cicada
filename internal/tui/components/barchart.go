package components

import (
	"fmt"
	"strings"
)

// BarItem represents a single bar in a bar chart.
type BarItem struct {
	Label string
	Value int
}

// BarChart renders a horizontal bar chart.
func BarChart(items []BarItem, maxWidth int) string {
	if len(items) == 0 {
		return ""
	}

	maxLabel := 0
	maxVal := 0
	for _, item := range items {
		if len(item.Label) > maxLabel {
			maxLabel = len(item.Label)
		}
		if item.Value > maxVal {
			maxVal = item.Value
		}
	}

	barWidth := maxWidth - maxLabel - 10
	if barWidth < 5 {
		barWidth = 5
	}

	var lines []string
	for _, item := range items {
		var bar int
		if maxVal > 0 {
			bar = item.Value * barWidth / maxVal
		}
		if bar < 1 && item.Value > 0 {
			bar = 1
		}
		label := fmt.Sprintf("%-*s", maxLabel, item.Label)
		line := fmt.Sprintf("%s %s %d", label, strings.Repeat("\u2588", bar), item.Value)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

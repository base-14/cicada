package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bar colors — gradient from cyan to purple.
var barColors = []string{"#06B6D4", "#3B82F6", "#6366F1", "#7C3AED", "#9333EA"}

// BarItem represents a single bar in a bar chart.
type BarItem struct {
	Label string
	Value int
}

// BarChart renders a horizontal bar chart with colored bars.
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

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Faint(true)

	var lines []string
	for i, item := range items {
		var bar int
		if maxVal > 0 {
			bar = item.Value * barWidth / maxVal
		}
		if bar < 1 && item.Value > 0 {
			bar = 1
		}
		color := barColors[i%len(barColors)]
		barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

		label := labelStyle.Render(fmt.Sprintf("%-*s", maxLabel, item.Label))
		barStr := barStyle.Render(strings.Repeat("█", bar))
		count := countStyle.Render(fmt.Sprintf("%d", item.Value))
		line := fmt.Sprintf("%s %s %s", label, barStr, count)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

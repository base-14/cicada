package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bar colors — cyan-focused gradient.
var barColors = []string{"#00FFFF", "#06B6D4", "#22D3EE", "#67E8F9", "#A5F3FC"}

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
	if barWidth > 40 {
		barWidth = 40
	}
	if barWidth < 10 {
		barWidth = 10
	}

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Faint(true)

	var lines []string
	for i, item := range items {
		// Use half-block resolution: each character position = 2 units
		var units int
		if maxVal > 0 {
			units = item.Value * barWidth * 2 / maxVal
		}
		if units < 1 && item.Value > 0 {
			units = 1
		}
		fullBlocks := units / 2
		halfBlock := units % 2

		color := barColors[i%len(barColors)]
		barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

		barStr := strings.Repeat("█", fullBlocks)
		if halfBlock == 1 {
			barStr += "▌"
		}

		label := labelStyle.Render(fmt.Sprintf("%-*s", maxLabel, item.Label))
		coloredBar := barStyle.Render(barStr)
		count := countStyle.Render(fmt.Sprintf("%d", item.Value))
		line := fmt.Sprintf("%s %s %s", label, coloredBar, count)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var sparkBlocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Sparkline renders a single-row sparkline from data within maxWidth characters.
func Sparkline(data []int, maxWidth int) string {
	if len(data) == 0 {
		return ""
	}

	if len(data) > maxWidth {
		sampled := make([]int, maxWidth)
		ratio := float64(len(data)) / float64(maxWidth)
		for i := range sampled {
			idx := int(float64(i) * ratio)
			if idx >= len(data) {
				idx = len(data) - 1
			}
			sampled[i] = data[idx]
		}
		data = sampled
	}

	minVal, maxVal := data[0], data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	result := make([]rune, len(data))
	span := maxVal - minVal
	for i, v := range data {
		if span == 0 {
			result[i] = sparkBlocks[len(sparkBlocks)/2]
		} else {
			idx := (v - minVal) * (len(sparkBlocks) - 1) / span
			result[i] = sparkBlocks[idx]
		}
	}

	return string(result)
}

// BarGraph renders a multi-row vertical bar graph with height rows.
// Each column is one data point. Labels show day numbers below.
func BarGraph(data []int, width, height int) string {
	if len(data) == 0 {
		return ""
	}

	// Sample data to fit width
	if len(data) > width {
		sampled := make([]int, width)
		ratio := float64(len(data)) / float64(width)
		for i := range sampled {
			idx := int(float64(i) * ratio)
			if idx >= len(data) {
				idx = len(data) - 1
			}
			sampled[i] = data[idx]
		}
		data = sampled
	}

	maxVal := 0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	var b strings.Builder

	// Render rows top to bottom
	for row := height; row >= 1; row-- {
		threshold := maxVal * row / height
		for _, v := range data {
			if v >= threshold && v > 0 {
				b.WriteString(barStyle.Render("█"))
			} else {
				b.WriteString(" ")
			}
		}
		// Y-axis label on right for top and middle
		if row == height {
			b.WriteString(dimStyle.Render(fmt.Sprintf(" %d", maxVal)))
		}
		b.WriteString("\n")
	}

	// X-axis: day markers
	for i := range data {
		dayNum := len(data) - i
		if dayNum%7 == 0 {
			b.WriteString(dimStyle.Render("┼"))
		} else {
			b.WriteString(dimStyle.Render("─"))
		}
	}
	b.WriteString("\n")

	// X-axis labels
	labelLine := strings.Repeat(" ", len(data))
	labelRunes := []rune(labelLine)
	for i := range data {
		dayNum := len(data) - i
		if dayNum%7 == 0 {
			label := fmt.Sprintf("%dd", dayNum)
			for j, r := range label {
				pos := i + j
				if pos < len(labelRunes) {
					labelRunes[pos] = r
				}
			}
		}
	}
	b.WriteString(dimStyle.Render(string(labelRunes)))

	return b.String()
}

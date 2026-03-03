package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Monochrome intensity — dark to bright white.
	heatShades = []string{"", "#333333", "#666666", "#999999", "#FFFFFF"}
	dayLabels  = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
)

// Heatmap renders a 7x24 ASCII heatmap (rows = days Mon-Sun, columns = hours 0-23).
// Uses monochrome block characters with intensity for readability on sparse data.
func Heatmap(data [7][24]int) string {
	maxVal := 0
	for day := range 7 {
		for hour := range 24 {
			if data[day][hour] > maxVal {
				maxVal = data[day][hour]
			}
		}
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#1A1A1A"))

	var b strings.Builder

	// Header with hour labels — every 3 hours, each cell is 2 chars wide
	b.WriteString("     ")
	for h := range 24 {
		if h%3 == 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-6d", h)))
		}
	}
	b.WriteString("\n")

	// Data rows — double-width cells
	for day := range 7 {
		b.WriteString(labelStyle.Render(fmt.Sprintf(" %s ", dayLabels[day])))
		for hour := range 24 {
			val := data[day][hour]
			if val == 0 {
				b.WriteString(emptyStyle.Render("··"))
			} else {
				b.WriteString(monoBlock(val, maxVal))
			}
		}
		if day < 6 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// monoBlock returns a double-width block with monochrome intensity.
func monoBlock(val, maxVal int) string {
	if maxVal == 0 {
		return "  "
	}
	idx := val * (len(heatShades) - 1) / maxVal
	if idx >= len(heatShades) {
		idx = len(heatShades) - 1
	}
	if idx < 1 {
		idx = 1
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(heatShades[idx]))
	return style.Render("██")
}

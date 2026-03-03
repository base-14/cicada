package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Heat colors — cool to hot gradient.
	heatColors = []string{"", "#1E3A5F", "#2563EB", "#F59E0B", "#EF4444"}
	heatBlocks = []string{" ", "░", "▒", "▓", "█"}
	dayLabels  = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
)

// Heatmap renders a 7x24 ASCII heatmap (rows = days Mon-Sun, columns = hours 0-23).
// Uses colored block characters based on intensity.
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

	var b strings.Builder

	// Header with hour labels
	b.WriteString("     ")
	for h := range 24 {
		if h%3 == 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-3d", h)))
		} else if (h-1)%3 != 0 && (h-2)%3 != 0 {
			b.WriteString("   ")
		}
	}
	b.WriteString("\n")

	// Data rows
	for day := range 7 {
		b.WriteString(labelStyle.Render(fmt.Sprintf(" %s ", dayLabels[day])))
		for hour := range 24 {
			val := data[day][hour]
			b.WriteString(coloredBlock(val, maxVal))
		}
		if day < 6 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// coloredBlock returns a colored block character based on value relative to max.
func coloredBlock(val, maxVal int) string {
	if val == 0 {
		return heatBlocks[0]
	}
	if maxVal == 0 {
		return heatBlocks[0]
	}
	idx := val * (len(heatBlocks) - 1) / maxVal
	if idx >= len(heatBlocks) {
		idx = len(heatBlocks) - 1
	}
	if idx < 1 {
		idx = 1
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(heatColors[idx]))
	return style.Render(heatBlocks[idx])
}

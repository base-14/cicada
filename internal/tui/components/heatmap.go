package components

import (
	"fmt"
	"strings"
)

var (
	heatBlocks = []string{" ", "░", "▒", "▓", "█"}
	dayLabels  = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
)

// Heatmap renders a 7x24 ASCII heatmap (rows = days Mon-Sun, columns = hours 0-23).
// Uses block characters based on intensity: space for 0, ░ ▒ ▓ █ for increasing values.
func Heatmap(data [7][24]int) string {
	// Find max value for scaling
	maxVal := 0
	for day := range 7 {
		for hour := range 24 {
			if data[day][hour] > maxVal {
				maxVal = data[day][hour]
			}
		}
	}

	var b strings.Builder

	// Header with hour labels
	b.WriteString("     ")
	for h := range 24 {
		if h%3 == 0 {
			b.WriteString(fmt.Sprintf("%-3d", h))
		} else {
			// Skip if previous was a multi-char label
			if (h-1)%3 == 0 || (h-2)%3 == 0 {
				continue
			}
			b.WriteString("   ")
		}
	}
	b.WriteString("\n")

	// Data rows
	for day := range 7 {
		b.WriteString(fmt.Sprintf(" %s ", dayLabels[day]))
		for hour := range 24 {
			val := data[day][hour]
			block := intensityBlock(val, maxVal)
			b.WriteString(block)
		}
		if day < 6 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// intensityBlock returns a block character based on the value relative to the max.
func intensityBlock(val, maxVal int) string {
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
	return heatBlocks[idx]
}

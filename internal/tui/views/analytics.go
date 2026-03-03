package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/r/cicada/internal/store"
	"github.com/r/cicada/internal/tui/components"
)

// AnalyticsView shows aggregate analytics with charts and heatmaps.
type AnalyticsView struct {
	store *store.Store
}

// NewAnalyticsView creates a new AnalyticsView.
func NewAnalyticsView(s *store.Store) *AnalyticsView {
	return &AnalyticsView{store: s}
}

// buildHeatmap computes a [7][24]int matrix from session start times.
// Rows are days of week (0=Monday .. 6=Sunday), columns are hours (0-23).
func (v *AnalyticsView) buildHeatmap() [7][24]int {
	var heatmap [7][24]int
	sessions := v.store.AllSessions()
	for _, s := range sessions {
		if s.StartTime.IsZero() {
			continue
		}
		weekday := s.StartTime.Weekday() // Sunday=0, Monday=1, ...
		// Convert to Monday=0, ..., Sunday=6
		day := int(weekday) - 1
		if day < 0 {
			day = 6 // Sunday
		}
		hour := s.StartTime.Hour()
		heatmap[day][hour]++
	}
	return heatmap
}

// View renders the analytics view.
func (v *AnalyticsView) View(width, height int) string {
	analytics := v.store.Analytics()

	if analytics.TotalSessions == 0 {
		return "\n  No session data available. Waiting for scan to complete..."
	}

	var b strings.Builder
	b.WriteString("\n")

	// Time period selector (static for now)
	b.WriteString("  Period: All Time\n\n")

	// Sessions per day sparkline (last 30 days)
	if len(analytics.SessionsByDate) > 0 {
		b.WriteString("  Sessions (last 30 days)\n")
		sparkData := buildSparkData(analytics.SessionsByDate, 30)
		sparkWidth := width - 4
		if sparkWidth > 60 {
			sparkWidth = 60
		}
		if sparkWidth < 10 {
			sparkWidth = 10
		}
		b.WriteString("  " + components.Sparkline(sparkData, sparkWidth) + "\n\n")
	}

	// Activity heatmap
	b.WriteString("  Activity Heatmap (day × hour)\n")
	heatmap := v.buildHeatmap()
	heatmapStr := components.Heatmap(heatmap)
	for _, line := range strings.Split(heatmapStr, "\n") {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n")

	// Top 10 tools bar chart
	if len(analytics.ToolsUsed) > 0 {
		b.WriteString("  Top Tools\n")
		topTools := topNToolItems(analytics.ToolsUsed, 10)
		chartWidth := width - 4
		if chartWidth > 60 {
			chartWidth = 60
		}
		b.WriteString("  " + strings.ReplaceAll(components.BarChart(topTools, chartWidth), "\n", "\n  ") + "\n\n")
	}

	// Model distribution
	if len(analytics.ModelsUsed) > 0 {
		b.WriteString("  Models\n  ")
		opus, sonnet, haiku, other := categorizeModelCounts(analytics.ModelsUsed)
		total := opus + sonnet + haiku + other
		if total > 0 {
			if opus > 0 {
				fmt.Fprintf(&b, "Opus %d%%  ", opus*100/total)
			}
			if sonnet > 0 {
				fmt.Fprintf(&b, "Sonnet %d%%  ", sonnet*100/total)
			}
			if haiku > 0 {
				fmt.Fprintf(&b, "Haiku %d%%  ", haiku*100/total)
			}
			if other > 0 {
				fmt.Fprintf(&b, "Other %d%%  ", other*100/total)
			}
		}
		b.WriteString("\n\n")
	}

	// Work mode distribution
	totalWork := analytics.WorkModeExplore + analytics.WorkModeBuild + analytics.WorkModeTest
	if totalWork > 0 {
		b.WriteString("  Work Mode\n")
		fmt.Fprintf(&b, "  Exploration %d%%    Building %d%%    Testing %d%%\n",
			analytics.WorkModeExplore*100/totalWork,
			analytics.WorkModeBuild*100/totalWork,
			analytics.WorkModeTest*100/totalWork,
		)
	}

	return b.String()
}

// buildSparkData returns session counts for the last n days, sorted by date.
func buildSparkData(sessionsByDate map[string]int, days int) []int {
	now := time.Now()
	data := make([]int, days)
	for i := range days {
		date := now.AddDate(0, 0, -(days-1-i)).Format("2006-01-02")
		data[i] = sessionsByDate[date]
	}
	return data
}

// topNToolItems returns the top n tools by usage as BarItems.
func topNToolItems(toolsUsed map[string]int, n int) []components.BarItem {
	type kv struct {
		key string
		val int
	}
	var sorted []kv
	for k, v := range toolsUsed {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].val > sorted[j].val
	})
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	items := make([]components.BarItem, len(sorted))
	for i, s := range sorted {
		items[i] = components.BarItem{Label: s.key, Value: s.val}
	}
	return items
}

// categorizeModelCounts buckets model usage into Opus, Sonnet, Haiku, and Other.
func categorizeModelCounts(modelsUsed map[string]int) (opus, sonnet, haiku, other int) {
	for name, count := range modelsUsed {
		lower := strings.ToLower(name)
		switch {
		case strings.Contains(lower, "opus"):
			opus += count
		case strings.Contains(lower, "sonnet"):
			sonnet += count
		case strings.Contains(lower, "haiku"):
			haiku += count
		default:
			other += count
		}
	}
	return
}

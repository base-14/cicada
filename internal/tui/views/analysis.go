package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
	"github.com/r/cicada/internal/tui/components"
)

// AnalysisView shows the merged dashboard + analytics with feel-good stats.
type AnalysisView struct {
	store   *store.Store
	scrollY int
}

// NewAnalysisView creates a new AnalysisView.
func NewAnalysisView(s *store.Store) *AnalysisView {
	return &AnalysisView{store: s}
}

// Update handles key events for scrolling.
func (v *AnalysisView) Update(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyUp:
		if v.scrollY > 0 {
			v.scrollY--
		}
	case tea.KeyDown:
		v.scrollY++
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "k":
			if v.scrollY > 0 {
				v.scrollY--
			}
		case "j":
			v.scrollY++
		}
	}
}

// View renders the analysis view.
func (v *AnalysisView) View(width, height int) string {
	analytics := v.store.Analytics()
	if analytics.TotalSessions == 0 {
		return "\n  No session data available. Waiting for scan to complete..."
	}

	sessions := v.store.AllSessions()
	insights := model.ComputeInsights(sessions)

	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB")).Bold(true)
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)

	var b strings.Builder
	b.WriteString("\n")

	// Summary Stats
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s    %s %s\n",
		labelStyle.Render("Sessions:"), valueStyle.Render(fmt.Sprintf("%d", analytics.TotalSessions)),
		labelStyle.Render("Tokens In:"), valueStyle.Render(formatTokensShort(analytics.TotalTokensIn)),
		labelStyle.Render("Tokens Out:"), valueStyle.Render(formatTokensShort(analytics.TotalTokensOut)),
		labelStyle.Render("Projects:"), valueStyle.Render(fmt.Sprintf("%d", analytics.ActiveProjects)),
	)
	b.WriteString("\n")

	// Streaks
	b.WriteString("  " + subtitleStyle.Render("Streaks") + "\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s\n",
		labelStyle.Render("Current Streak:"), accentStyle.Render(fmt.Sprintf("%d days", insights.CurrentStreak)),
		labelStyle.Render("Longest Streak:"), accentStyle.Render(fmt.Sprintf("%d days", insights.LongestStreak)),
		labelStyle.Render("Active Days:"), valueStyle.Render(fmt.Sprintf("%d", insights.ActiveDays)),
	)
	b.WriteString("\n")

	// Personal Bests
	b.WriteString("  " + subtitleStyle.Render("Personal Bests") + "\n")
	if insights.LongestSession != nil {
		slug := insights.LongestSession.Slug
		if slug == "" {
			slug = insights.LongestSession.UUID[:8]
		}
		project := "/" + strings.ReplaceAll(strings.TrimPrefix(insights.LongestSession.ProjectPath, "-"), "-", "/")
		fmt.Fprintf(&b, "  %s %s (%s · %s)\n",
			labelStyle.Render("Longest Session:"),
			valueStyle.Render(formatDuration(insights.LongestSession.Duration)),
			slug,
			project,
		)
	}
	if insights.MostProductiveDay != "" {
		fmt.Fprintf(&b, "  %s %s (%d sessions)\n",
			labelStyle.Render("Most Productive Day:"),
			valueStyle.Render(insights.MostProductiveDay),
			insights.MostProductiveDayCount,
		)
	}
	fmt.Fprintf(&b, "  %s %s\n",
		labelStyle.Render("Busiest Hour:"),
		valueStyle.Render(fmt.Sprintf("%02d:00", insights.BusiestHour)),
	)
	if insights.FavoriteTool != "" {
		fmt.Fprintf(&b, "  %s %s (%d calls)\n",
			labelStyle.Render("Favorite Tool:"),
			accentStyle.Render(insights.FavoriteTool),
			insights.FavoriteToolCount,
		)
	}
	b.WriteString("\n")

	// Trends
	b.WriteString("  " + subtitleStyle.Render("Trends") + "\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s\n",
		labelStyle.Render("This Week:"), valueStyle.Render(fmt.Sprintf("%d sessions", insights.SessionsThisWeek)),
		labelStyle.Render("Last Week:"), valueStyle.Render(fmt.Sprintf("%d sessions", insights.SessionsLastWeek)),
		labelStyle.Render("Avg Duration:"), valueStyle.Render(formatDuration(insights.AvgDuration)),
	)
	b.WriteString("\n")

	// Sparkline
	if len(analytics.SessionsByDate) > 0 {
		b.WriteString("  " + subtitleStyle.Render("Sessions (last 30 days)") + "\n")
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

	// Heatmap
	b.WriteString("  " + subtitleStyle.Render("Activity Heatmap (day × hour)") + "\n")
	heatmap := buildHeatmapFromSessions(sessions)
	heatmapStr := components.Heatmap(heatmap)
	for _, line := range strings.Split(heatmapStr, "\n") {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n")

	// Top Tools
	if len(analytics.ToolsUsed) > 0 {
		b.WriteString("  " + subtitleStyle.Render("Top Tools") + "\n")
		topTools := topNToolItems(analytics.ToolsUsed, 10)
		chartWidth := width - 4
		if chartWidth > 60 {
			chartWidth = 60
		}
		b.WriteString("  " + strings.ReplaceAll(components.BarChart(topTools, chartWidth), "\n", "\n  ") + "\n\n")
	}

	// Models
	if len(analytics.ModelsUsed) > 0 {
		b.WriteString("  " + subtitleStyle.Render("Models") + "\n  ")
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

	// Work Mode
	totalWork := analytics.WorkModeExplore + analytics.WorkModeBuild + analytics.WorkModeTest
	if totalWork > 0 {
		b.WriteString("  " + subtitleStyle.Render("Work Mode") + "\n")
		fmt.Fprintf(&b, "  Exploration %d%%    Building %d%%    Testing %d%%\n\n",
			analytics.WorkModeExplore*100/totalWork,
			analytics.WorkModeBuild*100/totalWork,
			analytics.WorkModeTest*100/totalWork,
		)
	}

	// Fun Stats
	b.WriteString("  " + subtitleStyle.Render("Fun Stats") + "\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s    %s %s\n",
		labelStyle.Render("Questions Asked:"), valueStyle.Render(fmt.Sprintf("%d", insights.TotalQuestions)),
		labelStyle.Render("Tool Calls:"), valueStyle.Render(fmt.Sprintf("%d", insights.TotalToolCalls)),
		labelStyle.Render("Unique Tools:"), valueStyle.Render(fmt.Sprintf("%d", insights.UniqueTools)),
		labelStyle.Render("Git Branches:"), valueStyle.Render(fmt.Sprintf("%d", insights.UniqueBranches)),
	)

	// Apply scroll
	content := b.String()
	lines := strings.Split(content, "\n")
	if v.scrollY >= len(lines) {
		v.scrollY = max(0, len(lines)-1)
	}
	visibleHeight := height - 2
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	end := v.scrollY + visibleHeight
	if end > len(lines) {
		end = len(lines)
	}

	if v.scrollY < len(lines) {
		return strings.Join(lines[v.scrollY:end], "\n")
	}
	return content
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

// buildHeatmapFromSessions computes a [7][24]int matrix from session start times.
// Rows are days of week (0=Monday .. 6=Sunday), columns are hours (0-23).
func buildHeatmapFromSessions(sessions []*model.SessionMeta) [7][24]int {
	var heatmap [7][24]int
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
		heatmap[day][s.StartTime.Hour()]++
	}
	return heatmap
}

package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/base-14/cicada/internal/model"
	"github.com/base-14/cicada/internal/store"
	"github.com/base-14/cicada/internal/tui/components"
)

// twoColumnMinWidth is the minimum terminal width for two-column layout.
const twoColumnMinWidth = 120

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

// analysisStyles bundles styles used by rendering helpers.
type analysisStyles struct {
	subtitle lipgloss.Style
	label    lipgloss.Style
	value    lipgloss.Style
	accent   lipgloss.Style
}

func newAnalysisStyles() analysisStyles {
	return analysisStyles{
		subtitle: lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")),
		label:    lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")),
		value:    lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB")).Bold(true),
		accent:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true),
	}
}

// View renders the analysis view.
func (v *AnalysisView) View(width, height int) string {
	analytics := v.store.Analytics()
	if analytics.TotalSessions == 0 {
		return "\n  No session data available. Waiting for scan to complete..."
	}

	sessions := v.store.AllSessions()
	history := v.store.HistoryStats()
	insights := model.ComputeInsights(sessions, history)
	styles := newAnalysisStyles()

	// Find earliest session date
	var earliest time.Time
	for _, s := range sessions {
		if !s.StartTime.IsZero() && (earliest.IsZero() || s.StartTime.Before(earliest)) {
			earliest = s.StartTime
		}
	}

	// Build activity data (shared by both layouts)
	activityByDate := make(map[string]int)
	for date, count := range analytics.SessionsByDate {
		activityByDate[date] = count
	}
	if history != nil {
		for date, count := range history.PromptsByDate {
			activityByDate[date] += count
		}
	}

	// Build heatmap data (shared by both layouts)
	heatmap := buildHeatmapFromSessions(sessions)
	if history != nil {
		for d := 0; d < 7; d++ {
			for h := 0; h < 24; h++ {
				heatmap[d][h] += history.Heatmap[d][h]
			}
		}
	}

	var content string
	header := renderSummaryHeader(analytics, earliest, styles)

	if width >= twoColumnMinWidth {
		leftWidth := width/2 - 1
		rightWidth := width - leftWidth - 2

		leftCol := renderLeftColumn(insights, analytics, styles)
		rightCol := renderRightColumn(analytics, activityByDate, heatmap, rightWidth, styles)

		leftStyled := lipgloss.NewStyle().Width(leftWidth).Render(leftCol)
		rightStyled := lipgloss.NewStyle().Width(rightWidth).Render(rightCol)

		columns := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, "  ", rightStyled)
		content = header + "\n" + columns
	} else {
		content = header + "\n" +
			renderLeftColumn(insights, analytics, styles) +
			renderRightColumnContent(analytics, activityByDate, heatmap, width, styles)
	}

	// Apply scroll
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

// renderSummaryHeader renders the full-width summary stats line.
func renderSummaryHeader(analytics *model.Analytics, earliest time.Time, s analysisStyles) string {
	var b strings.Builder
	b.WriteString("\n")

	sinceStr := ""
	if !earliest.IsZero() {
		sinceStr = earliest.Format("Jan 2006")
	}
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s    %s %s",
		s.label.Render("Sessions:"), s.value.Render(fmt.Sprintf("%d", analytics.TotalSessions)),
		s.label.Render("Tokens In:"), s.value.Render(formatTokensShort(analytics.TotalTokensIn)),
		s.label.Render("Tokens Out:"), s.value.Render(formatTokensShort(analytics.TotalTokensOut)),
		s.label.Render("Projects:"), s.value.Render(fmt.Sprintf("%d", analytics.ActiveProjects)),
	)
	if sinceStr != "" {
		fmt.Fprintf(&b, "    %s %s", s.label.Render("Since:"), s.value.Render(sinceStr))
	}
	b.WriteString("\n")

	return b.String()
}

// renderLeftColumn renders text-heavy stats: Streaks, Personal Bests, Trends, Work Mode, Fun Stats.
func renderLeftColumn(insights model.Insights, analytics *model.Analytics, s analysisStyles) string {
	var b strings.Builder

	// Streaks
	b.WriteString("  " + s.subtitle.Render("Streaks") + "\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s\n",
		s.label.Render("Current Streak:"), s.accent.Render(fmt.Sprintf("%d days", insights.CurrentStreak)),
		s.label.Render("Longest Streak:"), s.accent.Render(fmt.Sprintf("%d days", insights.LongestStreak)),
		s.label.Render("Active Days:"), s.value.Render(fmt.Sprintf("%d", insights.ActiveDays)),
	)
	b.WriteString("\n")

	// Personal Bests
	b.WriteString("  " + s.subtitle.Render("Personal Bests") + "\n")
	if insights.LongestSession != nil {
		slug := insights.LongestSession.Slug
		if slug == "" {
			slug = insights.LongestSession.UUID[:8]
		}
		project := "/" + strings.ReplaceAll(strings.TrimPrefix(insights.LongestSession.ProjectPath, "-"), "-", "/")
		fmt.Fprintf(&b, "  %s %s (%s · %s)\n",
			s.label.Render("Longest Session:"),
			s.value.Render(formatDuration(insights.LongestSession.Duration)),
			slug,
			project,
		)
	}
	if insights.MostProductiveDay != "" {
		fmt.Fprintf(&b, "  %s %s (%d sessions)\n",
			s.label.Render("Most Productive Day:"),
			s.value.Render(insights.MostProductiveDay),
			insights.MostProductiveDayCount,
		)
	}
	fmt.Fprintf(&b, "  %s %s\n",
		s.label.Render("Busiest Hour:"),
		s.value.Render(fmt.Sprintf("%02d:00", insights.BusiestHour)),
	)
	if insights.FavoriteTool != "" {
		fmt.Fprintf(&b, "  %s %s (%d calls)\n",
			s.label.Render("Favorite Tool:"),
			s.accent.Render(insights.FavoriteTool),
			insights.FavoriteToolCount,
		)
	}
	b.WriteString("\n")

	// Trends
	b.WriteString("  " + s.subtitle.Render("Trends") + "\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s\n",
		s.label.Render("This Week:"), s.value.Render(fmt.Sprintf("%d sessions", insights.SessionsThisWeek)),
		s.label.Render("Last Week:"), s.value.Render(fmt.Sprintf("%d sessions", insights.SessionsLastWeek)),
		s.label.Render("Avg Duration:"), s.value.Render(formatDuration(insights.AvgDuration)),
	)
	b.WriteString("\n")

	// Work Mode
	totalWork := analytics.WorkModeExplore + analytics.WorkModeBuild + analytics.WorkModeTest
	if totalWork > 0 {
		b.WriteString("  " + s.subtitle.Render("Work Mode") + "\n")
		fmt.Fprintf(&b, "  Exploration %d%%    Building %d%%    Testing %d%%\n\n",
			analytics.WorkModeExplore*100/totalWork,
			analytics.WorkModeBuild*100/totalWork,
			analytics.WorkModeTest*100/totalWork,
		)
	}

	// Fun Stats
	b.WriteString("  " + s.subtitle.Render("Fun Stats") + "\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s    %s %s\n",
		s.label.Render("Questions Asked:"), s.value.Render(fmt.Sprintf("%d", insights.TotalQuestions)),
		s.label.Render("Tool Calls:"), s.value.Render(fmt.Sprintf("%d", insights.TotalToolCalls)),
		s.label.Render("Unique Tools:"), s.value.Render(fmt.Sprintf("%d", insights.UniqueTools)),
		s.label.Render("Git Branches:"), s.value.Render(fmt.Sprintf("%d", insights.UniqueBranches)),
	)

	return b.String()
}

// renderRightColumn renders visual components for two-column layout.
func renderRightColumn(analytics *model.Analytics, activityByDate map[string]int, heatmap [7][24]int, width int, s analysisStyles) string {
	return renderRightColumnContent(analytics, activityByDate, heatmap, width, s)
}

// renderRightColumnContent renders: Activity Bar Graph, Heatmap, Top Tools, Models.
func renderRightColumnContent(analytics *model.Analytics, activityByDate map[string]int, heatmap [7][24]int, width int, s analysisStyles) string {
	var b strings.Builder

	// Activity bar graph
	if len(activityByDate) > 0 {
		b.WriteString("  " + s.subtitle.Render("Activity (last 30 days)") + "\n")
		sparkData := buildSparkData(activityByDate, 30)
		graphWidth := width - 6
		if graphWidth < 30 {
			graphWidth = 30
		}
		graph := components.BarGraph(sparkData, graphWidth, 4)
		for _, line := range strings.Split(graph, "\n") {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
	}

	// Heatmap
	b.WriteString("  " + s.subtitle.Render("Activity Heatmap (day × hour)") + "\n")
	heatmapStr := components.Heatmap(heatmap)
	for _, line := range strings.Split(heatmapStr, "\n") {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n")

	// Top Tools
	if len(analytics.ToolsUsed) > 0 {
		b.WriteString("  " + s.subtitle.Render("Top Tools") + "\n")
		topTools := topNToolItems(analytics.ToolsUsed, 10)
		chartWidth := width - 4
		b.WriteString("  " + strings.ReplaceAll(components.BarChart(topTools, chartWidth), "\n", "\n  ") + "\n\n")
	}

	// Models
	if len(analytics.ModelsUsed) > 0 {
		b.WriteString("  " + s.subtitle.Render("Models") + "\n  ")
		type modelCount struct {
			name  string
			count int
		}
		var models []modelCount
		total := 0
		for name, count := range analytics.ModelsUsed {
			models = append(models, modelCount{name, count})
			total += count
		}
		sort.Slice(models, func(i, j int) bool {
			return models[i].count > models[j].count
		})
		if total > 0 {
			for i, m := range models {
				if i > 0 {
					b.WriteString("  ")
				}
				fmt.Fprintf(&b, "%s %d%%", s.value.Render(m.name), m.count*100/total)
			}
		}
		b.WriteString("\n\n")
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

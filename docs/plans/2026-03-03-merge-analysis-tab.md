# Merge Dashboard + Analytics into Analysis Tab Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Merge the Dashboard and Analytics tabs into a single "Analysis" tab with feel-good stats (streaks, personal bests, trends, fun facts), plus the existing charts and heatmap.

**Architecture:** Remove the separate Dashboard (tab 0) and Analytics (tab 3) tabs. Create a new unified `AnalysisView` in `internal/tui/views/analysis.go` that combines everything. Reduce tab count from 6 to 5: Analysis, Projects, Sessions, Agents, Tools. Add new computed stats (streaks, bests, trends) using a helper struct computed from `store.AllSessions()`. The view is scrollable since it has lots of content.

**Tech Stack:** Go, Bubbletea, Lipgloss, existing components (BarChart, Sparkline, Heatmap)

---

### Task 1: Add Insights model with computed feel-good stats

Create a pure-computation struct that takes a slice of sessions and computes all the feel-good analytics. No TUI code — just data.

**Files:**
- Create: `internal/model/insights.go`
- Create: `internal/model/insights_test.go`

**Step 1: Write the failing tests**

Create `internal/model/insights_test.go`:

```go
package model

import (
	"testing"
	"time"
)

func makeSession(uuid string, start time.Time, duration time.Duration, tools map[string]int, messages int, branches []string) *SessionMeta {
	return &SessionMeta{
		UUID: uuid, Slug: uuid, ProjectPath: "-test",
		StartTime: start, EndTime: start.Add(duration), Duration: duration,
		TokensIn: 1000, TokensOut: 500,
		Models: map[string]int{"claude-opus-4-6": 1},
		ToolUsage: tools, SkillsUsed: map[string]int{},
		CommandsUsed: map[string]int{}, FileOps: map[string]int{},
		GitBranches: branches, MessageCount: messages,
	}
}

func TestComputeInsights_Streaks(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{"Read": 1}, 5, nil),
		makeSession("s2", now.AddDate(0, 0, -1), time.Hour, map[string]int{"Read": 1}, 3, nil),
		makeSession("s3", now.AddDate(0, 0, -2), time.Hour, map[string]int{"Read": 1}, 2, nil),
		// gap on day -3
		makeSession("s4", now.AddDate(0, 0, -4), time.Hour, map[string]int{"Read": 1}, 1, nil),
		makeSession("s5", now.AddDate(0, 0, -5), time.Hour, map[string]int{"Read": 1}, 1, nil),
	}

	insights := ComputeInsights(sessions)

	if insights.CurrentStreak != 3 {
		t.Errorf("expected current streak 3, got %d", insights.CurrentStreak)
	}
	if insights.LongestStreak != 3 {
		t.Errorf("expected longest streak 3, got %d", insights.LongestStreak)
	}
	if insights.ActiveDays != 5 {
		t.Errorf("expected 5 active days, got %d", insights.ActiveDays)
	}
}

func TestComputeInsights_PersonalBests(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	sessions := []*SessionMeta{
		makeSession("s1", now.Add(9*time.Hour), 3*time.Hour, map[string]int{"Read": 5, "Edit": 3}, 10, nil),
		makeSession("s2", now.Add(14*time.Hour), 30*time.Minute, map[string]int{"Bash": 2}, 5, nil),
	}

	insights := ComputeInsights(sessions)

	if insights.LongestSession.UUID != "s1" {
		t.Errorf("expected longest session s1, got %s", insights.LongestSession.UUID)
	}
	if insights.FavoriteTool != "Read" {
		t.Errorf("expected favorite tool 'Read', got %q", insights.FavoriteTool)
	}
	if insights.FavoriteToolCount != 5 {
		t.Errorf("expected favorite tool count 5, got %d", insights.FavoriteToolCount)
	}
	if insights.BusiestHour != 9 {
		t.Errorf("expected busiest hour 9, got %d", insights.BusiestHour)
	}
}

func TestComputeInsights_FunStats(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{"Read": 5, "Edit": 3}, 10, []string{"main", "feat"}),
		makeSession("s2", now.AddDate(0, 0, -1), time.Hour, map[string]int{"Bash": 2, "Read": 1}, 5, []string{"main"}),
	}

	insights := ComputeInsights(sessions)

	if insights.TotalQuestions != 15 {
		t.Errorf("expected 15 total questions, got %d", insights.TotalQuestions)
	}
	if insights.TotalToolCalls != 11 {
		t.Errorf("expected 11 total tool calls, got %d", insights.TotalToolCalls)
	}
	if insights.UniqueTools != 3 {
		t.Errorf("expected 3 unique tools, got %d", insights.UniqueTools)
	}
	if insights.UniqueBranches != 2 {
		t.Errorf("expected 2 unique branches, got %d", insights.UniqueBranches)
	}
}

func TestComputeInsights_Trends(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	// This week: 3 sessions
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{"Read": 5}, 10, nil),
		makeSession("s2", now.AddDate(0, 0, -1), time.Hour, map[string]int{"Read": 3}, 8, nil),
		makeSession("s3", now.AddDate(0, 0, -2), time.Hour, map[string]int{"Read": 2}, 6, nil),
		// Last week: 1 session
		makeSession("s4", now.AddDate(0, 0, -8), time.Hour, map[string]int{"Read": 1}, 4, nil),
	}

	insights := ComputeInsights(sessions)

	if insights.SessionsThisWeek != 3 {
		t.Errorf("expected 3 sessions this week, got %d", insights.SessionsThisWeek)
	}
	if insights.SessionsLastWeek != 1 {
		t.Errorf("expected 1 session last week, got %d", insights.SessionsLastWeek)
	}
}

func TestComputeInsights_MostProductiveDay(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{}, 5, nil),
		makeSession("s2", now.Add(2*time.Hour), time.Hour, map[string]int{}, 3, nil),
		makeSession("s3", now.AddDate(0, 0, -1), time.Hour, map[string]int{}, 2, nil),
	}

	insights := ComputeInsights(sessions)

	expected := now.Format("2006-01-02")
	if insights.MostProductiveDay != expected {
		t.Errorf("expected most productive day %s, got %s", expected, insights.MostProductiveDay)
	}
	if insights.MostProductiveDayCount != 2 {
		t.Errorf("expected most productive day count 2, got %d", insights.MostProductiveDayCount)
	}
}

func TestComputeInsights_Empty(t *testing.T) {
	insights := ComputeInsights(nil)
	if insights.CurrentStreak != 0 {
		t.Errorf("expected 0 streak for nil sessions, got %d", insights.CurrentStreak)
	}
	if insights.ActiveDays != 0 {
		t.Errorf("expected 0 active days, got %d", insights.ActiveDays)
	}
}

func TestComputeInsights_AverageDuration(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	sessions := []*SessionMeta{
		makeSession("s1", now, 2*time.Hour, map[string]int{}, 5, nil),
		makeSession("s2", now.AddDate(0, 0, -1), 4*time.Hour, map[string]int{}, 3, nil),
	}

	insights := ComputeInsights(sessions)

	if insights.AvgDuration != 3*time.Hour {
		t.Errorf("expected avg duration 3h, got %s", insights.AvgDuration)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `ComputeInsights` undefined

**Step 3: Write the implementation**

Create `internal/model/insights.go`:

```go
package model

import (
	"sort"
	"time"
)

// Insights holds computed feel-good analytics from session data.
type Insights struct {
	// Streaks
	CurrentStreak int
	LongestStreak int
	ActiveDays    int

	// Personal bests
	LongestSession       *SessionMeta
	MostProductiveDay      string // "2006-01-02"
	MostProductiveDayCount int    // sessions on that day
	BusiestHour            int    // 0-23
	FavoriteTool           string
	FavoriteToolCount      int

	// Trends
	SessionsThisWeek int
	SessionsLastWeek int
	AvgDuration      time.Duration

	// Fun stats
	TotalQuestions int // sum of MessageCount
	TotalToolCalls int
	UniqueTools    int
	UniqueBranches int
}

// ComputeInsights calculates feel-good analytics from a slice of sessions.
func ComputeInsights(sessions []*SessionMeta) Insights {
	var ins Insights
	if len(sessions) == 0 {
		return ins
	}

	// Collect dates, tools, branches
	dateSet := make(map[string]int)    // date -> session count
	toolsAgg := make(map[string]int)   // tool -> total count
	branchSet := make(map[string]bool)
	hourCounts := make(map[int]int)    // hour -> session count

	now := time.Now().Truncate(24 * time.Hour)
	weekStart := now.AddDate(0, 0, -int(now.Weekday())) // Sunday
	lastWeekStart := weekStart.AddDate(0, 0, -7)

	var totalDuration time.Duration

	for _, s := range sessions {
		// Messages (questions)
		ins.TotalQuestions += s.MessageCount

		// Duration
		totalDuration += s.Duration

		// Longest session
		if ins.LongestSession == nil || s.Duration > ins.LongestSession.Duration {
			ins.LongestSession = s
		}

		// Date tracking
		if !s.StartTime.IsZero() {
			date := s.StartTime.Format("2006-01-02")
			dateSet[date]++

			hour := s.StartTime.Hour()
			hourCounts[hour]++

			// Weekly trends
			if !s.StartTime.Before(weekStart) {
				ins.SessionsThisWeek++
			} else if !s.StartTime.Before(lastWeekStart) {
				ins.SessionsLastWeek++
			}
		}

		// Tools
		for tool, count := range s.ToolUsage {
			toolsAgg[tool] += count
			ins.TotalToolCalls += count
		}

		// Branches
		for _, br := range s.GitBranches {
			branchSet[br] = true
		}
	}

	// Active days
	ins.ActiveDays = len(dateSet)

	// Unique tools
	ins.UniqueTools = len(toolsAgg)

	// Unique branches
	ins.UniqueBranches = len(branchSet)

	// Average duration
	ins.AvgDuration = totalDuration / time.Duration(len(sessions))

	// Favorite tool
	for tool, count := range toolsAgg {
		if count > ins.FavoriteToolCount {
			ins.FavoriteTool = tool
			ins.FavoriteToolCount = count
		}
	}

	// Busiest hour
	maxHourCount := 0
	for hour, count := range hourCounts {
		if count > maxHourCount {
			maxHourCount = count
			ins.BusiestHour = hour
		}
	}

	// Most productive day
	for date, count := range dateSet {
		if count > ins.MostProductiveDayCount {
			ins.MostProductiveDay = date
			ins.MostProductiveDayCount = count
		}
	}

	// Streaks — sort dates descending, walk backwards from today
	dates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	// Current streak: count consecutive days from today backwards
	today := now.Format("2006-01-02")
	if _, ok := dateSet[today]; ok {
		ins.CurrentStreak = 1
		for i := 1; ; i++ {
			prev := now.AddDate(0, 0, -i).Format("2006-01-02")
			if _, ok := dateSet[prev]; ok {
				ins.CurrentStreak++
			} else {
				break
			}
		}
	}

	// Longest streak: find longest consecutive run in all dates
	if len(dates) > 0 {
		allDates := make(map[string]bool, len(dateSet))
		for d := range dateSet {
			allDates[d] = true
		}
		// Find earliest date
		earliest := dates[len(dates)-1]
		t, _ := time.Parse("2006-01-02", earliest)
		latest := dates[0]
		tEnd, _ := time.Parse("2006-01-02", latest)

		streak := 0
		for d := t; !d.After(tEnd); d = d.AddDate(0, 0, 1) {
			if allDates[d.Format("2006-01-02")] {
				streak++
				if streak > ins.LongestStreak {
					ins.LongestStreak = streak
				}
			} else {
				streak = 0
			}
		}
	}

	return ins
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/model/insights.go internal/model/insights_test.go
git commit -m "feat: add Insights model with streaks, bests, trends, fun stats"
```

---

### Task 2: Create unified AnalysisView

Replace both Dashboard (in app.go) and AnalyticsView with a single scrollable AnalysisView.

**Files:**
- Create: `internal/tui/views/analysis.go`
- Create: `internal/tui/views/analysis_test.go`

**Step 1: Write the failing tests**

Create `internal/tui/views/analysis_test.go`:

```go
package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

func TestNewAnalysisView(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)
	if view == nil {
		t.Fatal("expected non-nil analysis view")
	}
}

func TestAnalysisView_Empty(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)
	content := view.View(80, 24)
	if content == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(content, "No session data") {
		t.Error("expected empty state message")
	}
}

func TestAnalysisView_WithSessions(t *testing.T) {
	s := store.New()
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "session-1", ProjectPath: "-test",
		StartTime: now, EndTime: now.Add(time.Hour), Duration: time.Hour,
		TokensIn: 10000, TokensOut: 5000,
		Models: map[string]int{"claude-opus-4-6": 5},
		ToolUsage: map[string]int{"Read": 10, "Edit": 5, "Bash": 3},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, GitBranches: []string{"main"},
		SubagentCount: 1, MessageCount: 20,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "session-2", ProjectPath: "-test",
		StartTime: yesterday, EndTime: yesterday.Add(30 * time.Minute), Duration: 30 * time.Minute,
		TokensIn: 5000, TokensOut: 2000,
		Models: map[string]int{"claude-sonnet-4-6": 3},
		ToolUsage: map[string]int{"Read": 5, "Write": 2},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, GitBranches: []string{"main", "feat"},
		SubagentCount: 0, MessageCount: 10,
	})

	view := NewAnalysisView(s)
	content := view.View(100, 60)

	// Stats section
	if !strings.Contains(content, "Sessions") {
		t.Error("expected sessions count")
	}

	// Streaks
	if !strings.Contains(content, "Streak") {
		t.Error("expected streak section")
	}

	// Personal bests
	if !strings.Contains(content, "Longest Session") {
		t.Error("expected longest session")
	}

	// Favorite tool
	if !strings.Contains(content, "Read") {
		t.Error("expected favorite tool 'Read'")
	}

	// Heatmap
	if !strings.Contains(content, "Heatmap") {
		t.Error("expected heatmap")
	}

	// Tools chart
	if !strings.Contains(content, "Tools") {
		t.Error("expected tools section")
	}

	// Models
	if !strings.Contains(content, "Models") {
		t.Error("expected models section")
	}
}

func TestAnalysisView_Scrolling(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)

	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	if view.scrollY != 1 {
		t.Errorf("expected scrollY=1, got %d", view.scrollY)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.scrollY != 0 {
		t.Errorf("expected scrollY=0, got %d", view.scrollY)
	}
}

func TestAnalysisView_VimScrolling(t *testing.T) {
	s := store.New()
	view := NewAnalysisView(s)

	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if view.scrollY != 1 {
		t.Errorf("expected scrollY=1 after 'j', got %d", view.scrollY)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if view.scrollY != 0 {
		t.Errorf("expected scrollY=0 after 'k', got %d", view.scrollY)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `NewAnalysisView` undefined

**Step 3: Write the implementation**

Create `internal/tui/views/analysis.go`:

```go
package views

import (
	"fmt"
	"strings"

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

	// ── Summary Stats ──
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s    %s %s\n",
		labelStyle.Render("Sessions:"), valueStyle.Render(fmt.Sprintf("%d", analytics.TotalSessions)),
		labelStyle.Render("Tokens In:"), valueStyle.Render(formatTokensShort(analytics.TotalTokensIn)),
		labelStyle.Render("Tokens Out:"), valueStyle.Render(formatTokensShort(analytics.TotalTokensOut)),
		labelStyle.Render("Projects:"), valueStyle.Render(fmt.Sprintf("%d", analytics.ActiveProjects)),
	)
	b.WriteString("\n")

	// ── Streaks ──
	b.WriteString("  " + subtitleStyle.Render("Streaks") + "\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s\n",
		labelStyle.Render("Current Streak:"), accentStyle.Render(fmt.Sprintf("%d days", insights.CurrentStreak)),
		labelStyle.Render("Longest Streak:"), accentStyle.Render(fmt.Sprintf("%d days", insights.LongestStreak)),
		labelStyle.Render("Active Days:"), valueStyle.Render(fmt.Sprintf("%d", insights.ActiveDays)),
	)
	b.WriteString("\n")

	// ── Personal Bests ──
	b.WriteString("  " + subtitleStyle.Render("Personal Bests") + "\n")
	if insights.LongestSession != nil {
		slug := insights.LongestSession.Slug
		if slug == "" {
			slug = insights.LongestSession.UUID[:8]
		}
		fmt.Fprintf(&b, "  %s %s (%s)\n",
			labelStyle.Render("Longest Session:"),
			valueStyle.Render(formatDuration(insights.LongestSession.Duration)),
			slug,
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

	// ── Trends ──
	b.WriteString("  " + subtitleStyle.Render("Trends") + "\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s\n",
		labelStyle.Render("This Week:"), valueStyle.Render(fmt.Sprintf("%d sessions", insights.SessionsThisWeek)),
		labelStyle.Render("Last Week:"), valueStyle.Render(fmt.Sprintf("%d sessions", insights.SessionsLastWeek)),
		labelStyle.Render("Avg Duration:"), valueStyle.Render(formatDuration(insights.AvgDuration)),
	)
	b.WriteString("\n")

	// ── Sessions sparkline (last 30 days) ──
	if len(analytics.SessionsByDate) > 0 {
		b.WriteString("  " + subtitleStyle.Render("Sessions (last 30 days)") + "\n")
		sparkData := buildSparkData(analytics.SessionsByDate, 30)
		sparkWidth := width - 4
		if sparkWidth > 60 { sparkWidth = 60 }
		if sparkWidth < 10 { sparkWidth = 10 }
		b.WriteString("  " + components.Sparkline(sparkData, sparkWidth) + "\n\n")
	}

	// ── Activity Heatmap ──
	b.WriteString("  " + subtitleStyle.Render("Activity Heatmap (day × hour)") + "\n")
	heatmap := buildHeatmapFromSessions(sessions)
	heatmapStr := components.Heatmap(heatmap)
	for _, line := range strings.Split(heatmapStr, "\n") {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n")

	// ── Top Tools bar chart ──
	if len(analytics.ToolsUsed) > 0 {
		b.WriteString("  " + subtitleStyle.Render("Top Tools") + "\n")
		topTools := topNToolItems(analytics.ToolsUsed, 10)
		chartWidth := width - 4
		if chartWidth > 60 { chartWidth = 60 }
		b.WriteString("  " + strings.ReplaceAll(components.BarChart(topTools, chartWidth), "\n", "\n  ") + "\n\n")
	}

	// ── Models ──
	if len(analytics.ModelsUsed) > 0 {
		b.WriteString("  " + subtitleStyle.Render("Models") + "\n  ")
		opus, sonnet, haiku, other := categorizeModelCounts(analytics.ModelsUsed)
		total := opus + sonnet + haiku + other
		if total > 0 {
			if opus > 0 { fmt.Fprintf(&b, "Opus %d%%  ", opus*100/total) }
			if sonnet > 0 { fmt.Fprintf(&b, "Sonnet %d%%  ", sonnet*100/total) }
			if haiku > 0 { fmt.Fprintf(&b, "Haiku %d%%  ", haiku*100/total) }
			if other > 0 { fmt.Fprintf(&b, "Other %d%%  ", other*100/total) }
		}
		b.WriteString("\n\n")
	}

	// ── Work Mode ──
	totalWork := analytics.WorkModeExplore + analytics.WorkModeBuild + analytics.WorkModeTest
	if totalWork > 0 {
		b.WriteString("  " + subtitleStyle.Render("Work Mode") + "\n")
		fmt.Fprintf(&b, "  Exploration %d%%    Building %d%%    Testing %d%%\n\n",
			analytics.WorkModeExplore*100/totalWork,
			analytics.WorkModeBuild*100/totalWork,
			analytics.WorkModeTest*100/totalWork,
		)
	}

	// ── Fun Stats ──
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
	if visibleHeight < 1 { visibleHeight = 1 }
	end := v.scrollY + visibleHeight
	if end > len(lines) { end = len(lines) }

	if v.scrollY < len(lines) {
		return strings.Join(lines[v.scrollY:end], "\n")
	}
	return content
}

// buildHeatmapFromSessions computes a [7][24]int matrix from session start times.
func buildHeatmapFromSessions(sessions []*model.SessionMeta) [7][24]int {
	var heatmap [7][24]int
	for _, s := range sessions {
		if s.StartTime.IsZero() {
			continue
		}
		weekday := s.StartTime.Weekday()
		day := int(weekday) - 1
		if day < 0 { day = 6 }
		heatmap[day][s.StartTime.Hour()]++
	}
	return heatmap
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/views/analysis.go internal/tui/views/analysis_test.go
git commit -m "feat: add unified AnalysisView with feel-good stats"
```

---

### Task 3: Rewire app.go — merge tabs, remove old code

Replace the 6-tab layout with 5 tabs, remove Dashboard render code, remove old AnalyticsView, wire up new AnalysisView with scrolling support.

**Files:**
- Modify: `internal/tui/app.go` (major changes)
- Modify: `internal/tui/app_test.go` (fix tab indices)
- Delete: `internal/tui/views/analytics.go` (replaced by analysis.go)
- Delete: `internal/tui/views/analytics_test.go` (replaced by analysis_test.go)

**Step 1: Update app.go**

Changes:
1. `tabNames` becomes `[]string{"Analysis", "Projects", "Sessions", "Agents", "Tools"}` (5 tabs)
2. Replace `analyticsView *views.AnalyticsView` with `analysisView *views.AnalysisView` in App struct
3. Update `NewApp` to create `NewAnalysisView`
4. Remove `renderDashboard()` method entirely
5. Remove `buildSparklineData`, `topNTools`, `categorizeModels` helper functions (they live in views package now)
6. Update `renderContent()` switch — tab 0 now calls `a.analysisView.View(...)`, tabs 1-4 map to Projects/Sessions/Agents/Tools
7. Update key forwarding: up/down/j/k for tab 4 (Tools, was 5), Enter for tab 1 (Projects) and 2 (Sessions)
8. Update number key handling: `"1", "2", "3", "4", "5"` instead of 1-6
9. Forward up/down/j/k to analysisView on tab 0

**Step 2: Update app_test.go**

- Tab indices shift: Projects=1, Sessions=2, Agents=3, Tools=4
- `TestApp_NumberKeyNavigation`: pressing '3' goes to Sessions (still index 2, OK)
- `TestApp_SlashActivatesFilterOnSessions`: activeTab=2 (still correct)
- `TestApp_SlashActivatesFilterOnProjects`: activeTab=1 (still correct)
- `TestApp_ProjectDetailDrillIn`: activeTab=1 (still correct)
- Fix `TestApp_TabWrapAround`: last tab is now 4 (was 5)
- Number keys: change `"1", "2", "3", "4", "5", "6"` to `"1", "2", "3", "4", "5"`
- `TestApp_FilterActiveBlocksQuit`: activeTab=2 (still correct)

**Step 3: Delete old analytics files**

```bash
rm internal/tui/views/analytics.go internal/tui/views/analytics_test.go
```

**Step 4: Run tests, build, lint**

Run: `make test`
Expected: PASS (all tests)

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: merge Dashboard and Analytics into unified Analysis tab"
```

---

### Task 4: Update help overlay for new tab layout

The help overlay says "1-6 Switch view" — update to "1-5".

**Files:**
- Modify: `internal/tui/app.go` (help overlay text)

**Step 1: Update help text**

Change `1-6            Switch view` to `1-5            Switch view`.

**Step 2: Run tests, commit**

Run: `make test`

```bash
git add internal/tui/app.go
git commit -m "fix: update help overlay for 5-tab layout"
```

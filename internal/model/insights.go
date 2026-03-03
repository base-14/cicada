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
	LongestSession         *SessionMeta
	MostProductiveDay      string // "2006-01-02"
	MostProductiveDayCount int    // sessions on that day
	BusiestHour            int    // 0-23
	FavoriteTool           string
	FavoriteToolCount      int

	// Trends
	SessionsThisWeek int
	SessionsLastWeek int
	AvgDuration      time.Duration

	// Totals
	TotalQuestions int // sum of MessageCount
	TotalToolCalls int
	UniqueTools    int
	UniqueBranches int
	TopWords       []WordCount
	AvgPromptWords float64
	P95PromptWords int
}

// ComputeInsights calculates feel-good analytics from a slice of sessions.
// If history is non-nil, it enriches active days, streaks, hour counts, and total questions.
func ComputeInsights(sessions []*SessionMeta, history *HistoryStats) Insights {
	var ins Insights
	if len(sessions) == 0 {
		return ins
	}

	// Collect dates, tools, branches
	dateSet := make(map[string]int)  // date -> session count
	toolsAgg := make(map[string]int) // tool -> total count
	branchSet := make(map[string]bool)
	hourCounts := make(map[int]int) // hour -> session count

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

	// Merge history data if available
	if history != nil {
		for date := range history.ActiveDays {
			if _, exists := dateSet[date]; !exists {
				dateSet[date]++
			}
		}
		for hour, count := range history.HourCounts {
			hourCounts[hour] += count
		}
		ins.TotalQuestions = history.TotalPrompts
		ins.TopWords = history.TopWords
		ins.AvgPromptWords = history.AvgPromptWords
		ins.P95PromptWords = history.P95PromptWords
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

	// Busiest hour (ties broken by earlier hour)
	maxHourCount := 0
	for hour, count := range hourCounts {
		if count > maxHourCount || (count == maxHourCount && hour < ins.BusiestHour) {
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

	// Streaks -- sort dates descending, walk backwards from today
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
		// Find earliest and latest dates
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

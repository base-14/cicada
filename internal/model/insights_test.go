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

	insights := ComputeInsights(sessions, nil)

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
	// Use local midnight so that .Hour() returns expected values
	y, m, d := time.Now().Date()
	localMidnight := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	sessions := []*SessionMeta{
		makeSession("s1", localMidnight.Add(9*time.Hour), 3*time.Hour, map[string]int{"Read": 5, "Edit": 3}, 10, nil),
		makeSession("s2", localMidnight.Add(14*time.Hour), 30*time.Minute, map[string]int{"Bash": 2}, 5, nil),
	}

	insights := ComputeInsights(sessions, nil)

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

func TestComputeInsights_Totals(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{"Read": 5, "Edit": 3}, 10, []string{"main", "feat"}),
		makeSession("s2", now.AddDate(0, 0, -1), time.Hour, map[string]int{"Bash": 2, "Read": 1}, 5, []string{"main"}),
	}

	insights := ComputeInsights(sessions, nil)

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

	insights := ComputeInsights(sessions, nil)

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

	insights := ComputeInsights(sessions, nil)

	expected := now.Format("2006-01-02")
	if insights.MostProductiveDay != expected {
		t.Errorf("expected most productive day %s, got %s", expected, insights.MostProductiveDay)
	}
	if insights.MostProductiveDayCount != 2 {
		t.Errorf("expected most productive day count 2, got %d", insights.MostProductiveDayCount)
	}
}

func TestComputeInsights_Empty(t *testing.T) {
	insights := ComputeInsights(nil, nil)
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

	insights := ComputeInsights(sessions, nil)

	if insights.AvgDuration != 3*time.Hour {
		t.Errorf("expected avg duration 3h, got %s", insights.AvgDuration)
	}
}

func TestComputeInsights_WithHistory_EnrichesActiveDays(t *testing.T) {
	now := time.Date(2026, 3, 3, 12, 0, 0, 0, time.Local)
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{"Read": 1}, 5, nil),
	}
	history := &HistoryStats{
		TotalPrompts: 100,
		ActiveDays: map[string]bool{
			now.Format("2006-01-02"):                    true,
			now.AddDate(0, 0, -1).Format("2006-01-02"):  true,
			now.AddDate(0, 0, -5).Format("2006-01-02"):  true,
		},
		PromptsByDate: map[string]int{},
		HourCounts:    map[int]int{},
	}

	insights := ComputeInsights(sessions, history)

	if insights.ActiveDays != 3 {
		t.Errorf("expected 3 active days, got %d", insights.ActiveDays)
	}
	if insights.TotalQuestions != 100 {
		t.Errorf("expected 100 total questions from history, got %d", insights.TotalQuestions)
	}
}

func TestComputeInsights_WithHistory_EnrichesStreaks(t *testing.T) {
	// Use time.Now() so "today" in the test matches "today" in ComputeInsights.
	now := time.Now().Truncate(24 * time.Hour).Add(12 * time.Hour)
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{}, 1, nil),
	}
	history := &HistoryStats{
		TotalPrompts: 10,
		ActiveDays: map[string]bool{
			now.AddDate(0, 0, -1).Format("2006-01-02"): true,
			now.AddDate(0, 0, -2).Format("2006-01-02"): true,
		},
		PromptsByDate: map[string]int{},
		HourCounts:    map[int]int{},
	}

	insights := ComputeInsights(sessions, history)

	if insights.CurrentStreak != 3 {
		t.Errorf("expected current streak 3 (today + 2 from history), got %d", insights.CurrentStreak)
	}
}

func TestComputeInsights_TopWords(t *testing.T) {
	now := time.Date(2026, 3, 3, 12, 0, 0, 0, time.Local)
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{"Read": 1}, 5, nil),
	}
	history := &HistoryStats{
		TotalPrompts:  5,
		ActiveDays:    map[string]bool{now.Format("2006-01-02"): true},
		PromptsByDate: map[string]int{},
		HourCounts:    map[int]int{},
		TopWords: []WordCount{
			{Word: "authentication", Count: 10},
			{Word: "login", Count: 7},
			{Word: "fix", Count: 5},
		},
	}

	insights := ComputeInsights(sessions, history)

	if len(insights.TopWords) != 3 {
		t.Fatalf("expected 3 top words, got %d", len(insights.TopWords))
	}
	if insights.TopWords[0].Word != "authentication" {
		t.Errorf("expected first top word 'authentication', got %q", insights.TopWords[0].Word)
	}
}

func TestComputeInsights_NilHistory(t *testing.T) {
	now := time.Date(2026, 3, 3, 12, 0, 0, 0, time.Local)
	sessions := []*SessionMeta{
		makeSession("s1", now, time.Hour, map[string]int{"Read": 1}, 5, nil),
	}
	insights := ComputeInsights(sessions, nil)
	if insights.TotalQuestions != 5 {
		t.Errorf("expected 5 questions from session MessageCount, got %d", insights.TotalQuestions)
	}
}

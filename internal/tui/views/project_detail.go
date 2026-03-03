package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/tui/components"
)

var projectDetailTabNames = []string{"Overview", "Sessions", "Tools", "Activity", "Skills"}

// ProjectDetailView shows detailed information about a single project.
type ProjectDetailView struct {
	project   string
	sessions  []*model.SessionMeta
	activeTab int
	scrollY   int
}

// NewProjectDetailView creates a new ProjectDetailView.
func NewProjectDetailView(project string, sessions []*model.SessionMeta) *ProjectDetailView {
	sorted := make([]*model.SessionMeta, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.After(sorted[j].StartTime)
	})
	return &ProjectDetailView{
		project:  project,
		sessions: sorted,
	}
}

// Update handles key events for sub-tab navigation and scrolling.
func (v *ProjectDetailView) Update(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyLeft:
		v.activeTab = (v.activeTab - 1 + len(projectDetailTabNames)) % len(projectDetailTabNames)
		v.scrollY = 0
	case tea.KeyRight:
		v.activeTab = (v.activeTab + 1) % len(projectDetailTabNames)
		v.scrollY = 0
	case tea.KeyUp:
		if v.scrollY > 0 {
			v.scrollY--
		}
	case tea.KeyDown:
		v.scrollY++
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "h":
			v.activeTab = (v.activeTab - 1 + len(projectDetailTabNames)) % len(projectDetailTabNames)
			v.scrollY = 0
		case "l":
			v.activeTab = (v.activeTab + 1) % len(projectDetailTabNames)
			v.scrollY = 0
		case "k":
			if v.scrollY > 0 {
				v.scrollY--
			}
		case "j":
			v.scrollY++
		}
	}
}

// View renders the project detail with sub-tabs and scrollable content.
func (v *ProjectDetailView) View(width, height int) string {
	var b strings.Builder
	b.WriteString("\n")

	// Sub-tab bar
	b.WriteString("  ")
	for i, name := range projectDetailTabNames {
		if i == v.activeTab {
			b.WriteString("[" + name + "]")
		} else {
			b.WriteString(" " + name + " ")
		}
		if i < len(projectDetailTabNames)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")
	b.WriteString("  " + strings.Repeat("─", width-4) + "\n")

	var content string
	switch v.activeTab {
	case 0:
		content = v.renderOverview(width)
	case 1:
		content = v.renderSessions(width)
	case 2:
		content = v.renderTools(width)
	case 3:
		content = v.renderActivity(width)
	case 4:
		content = v.renderSkills(width)
	}

	lines := strings.Split(content, "\n")
	if v.scrollY >= len(lines) {
		v.scrollY = max(0, len(lines)-1)
	}
	visibleHeight := height - 6
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	end := v.scrollY + visibleHeight
	if end > len(lines) {
		end = len(lines)
	}
	if v.scrollY < len(lines) {
		b.WriteString(strings.Join(lines[v.scrollY:end], "\n"))
	}

	return b.String()
}

func (v *ProjectDetailView) decodedName() string {
	return "/" + strings.ReplaceAll(strings.TrimPrefix(v.project, "-"), "-", "/")
}

func (v *ProjectDetailView) renderOverview(width int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Project: %s\n\n", v.decodedName()))

	var totalTokensIn, totalTokensOut int64
	var totalDuration time.Duration
	var totalMessages, totalSubagents, totalTools int
	models := make(map[string]int)
	branches := make(map[string]bool)
	var earliest, latest time.Time

	for _, s := range v.sessions {
		totalTokensIn += s.TokensIn
		totalTokensOut += s.TokensOut
		totalDuration += s.Duration
		totalMessages += s.MessageCount
		totalSubagents += s.SubagentCount
		for _, count := range s.ToolUsage {
			totalTools += count
		}
		for m, count := range s.Models {
			models[m] += count
		}
		for _, br := range s.GitBranches {
			branches[br] = true
		}
		if !s.StartTime.IsZero() && (earliest.IsZero() || s.StartTime.Before(earliest)) {
			earliest = s.StartTime
		}
		if !s.EndTime.IsZero() && s.EndTime.After(latest) {
			latest = s.EndTime
		}
	}

	b.WriteString(fmt.Sprintf("  %-15s %d\n", "Sessions:", len(v.sessions)))
	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Total Time:", formatDuration(totalDuration)))
	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Tokens In:", formatTokensShort(totalTokensIn)))
	b.WriteString(fmt.Sprintf("  %-15s %s\n", "Tokens Out:", formatTokensShort(totalTokensOut)))
	b.WriteString(fmt.Sprintf("  %-15s %d\n", "Messages:", totalMessages))
	b.WriteString(fmt.Sprintf("  %-15s %d\n", "Tool Calls:", totalTools))
	b.WriteString(fmt.Sprintf("  %-15s %d\n", "Subagents:", totalSubagents))
	b.WriteString("\n")

	if !earliest.IsZero() {
		b.WriteString(fmt.Sprintf("  %-15s %s → %s\n\n", "Active:",
			earliest.Format("2006-01-02"), latest.Format("2006-01-02")))
	}

	if len(models) > 0 {
		b.WriteString("  Models:\n")
		for name, count := range models {
			b.WriteString(fmt.Sprintf("    %-40s %d messages\n", name, count))
		}
		b.WriteString("\n")
	}

	// Work mode breakdown
	var explore, build, test int
	for _, s := range v.sessions {
		for tool, count := range s.ToolUsage {
			switch tool {
			case "Read", "Grep", "Glob", "WebFetch", "WebSearch", "LS", "SemanticSearch":
				explore += count
			case "Write", "Edit", "StrReplace":
				build += count
			case "Bash", "Agent", "TaskCreate", "TaskUpdate":
				test += count
			}
		}
	}
	totalWork := explore + build + test
	if totalWork > 0 {
		b.WriteString("  Work Mode:\n")
		fmt.Fprintf(&b, "    Exploration %d%%    Building %d%%    Testing %d%%\n\n",
			explore*100/totalWork, build*100/totalWork, test*100/totalWork)
	}

	if len(branches) > 0 {
		brList := make([]string, 0, len(branches))
		for br := range branches {
			brList = append(brList, br)
		}
		sort.Strings(brList)
		b.WriteString("  Git Branches: " + strings.Join(brList, ", ") + "\n")
	}

	return b.String()
}

func (v *ProjectDetailView) renderSessions(width int) string {
	if len(v.sessions) == 0 {
		return "  No sessions in this project."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %-30s %-20s %-12s %10s\n", "Session", "Date", "Duration", "Tokens"))
	b.WriteString("  " + strings.Repeat("─", width-4) + "\n")

	for _, s := range v.sessions {
		slug := s.Slug
		if slug == "" {
			slug = s.UUID[:8]
		}
		if len(slug) > 30 {
			slug = slug[:27] + "..."
		}
		date := ""
		if !s.StartTime.IsZero() {
			date = s.StartTime.Format("2006-01-02 15:04")
		}
		b.WriteString(fmt.Sprintf("  %-30s %-20s %-12s %10s\n",
			slug, date, formatDuration(s.Duration), formatTokensShort(s.TokensIn+s.TokensOut)))
	}
	return b.String()
}

func (v *ProjectDetailView) renderTools(width int) string {
	toolsAgg := make(map[string]int)
	for _, s := range v.sessions {
		for tool, count := range s.ToolUsage {
			toolsAgg[tool] += count
		}
	}
	if len(toolsAgg) == 0 {
		return "  No tool usage in this project."
	}
	var b strings.Builder
	b.WriteString("  Top Tools\n")
	topTools := topNToolItems(toolsAgg, 10)
	chartWidth := width - 4
	b.WriteString("  " + strings.ReplaceAll(components.BarChart(topTools, chartWidth), "\n", "\n  ") + "\n")
	return b.String()
}

func (v *ProjectDetailView) renderActivity(width int) string {
	var b strings.Builder

	sessionsByDate := make(map[string]int)
	for _, s := range v.sessions {
		if !s.StartTime.IsZero() {
			sessionsByDate[s.StartTime.Format("2006-01-02")]++
		}
	}
	if len(sessionsByDate) > 0 {
		b.WriteString("  Sessions (last 30 days)\n")
		sparkData := buildSparkData(sessionsByDate, 30)
		sparkWidth := width - 4
		if sparkWidth < 10 {
			sparkWidth = 10
		}
		b.WriteString("  " + components.Sparkline(sparkData, sparkWidth) + "\n\n")
	}

	var heatmap [7][24]int
	for _, s := range v.sessions {
		if s.StartTime.IsZero() {
			continue
		}
		weekday := s.StartTime.Weekday()
		day := int(weekday) - 1
		if day < 0 {
			day = 6
		}
		heatmap[day][s.StartTime.Hour()]++
	}
	b.WriteString("  Activity Heatmap (day × hour)\n")
	heatmapStr := components.Heatmap(heatmap)
	for _, line := range strings.Split(heatmapStr, "\n") {
		b.WriteString("  " + line + "\n")
	}
	return b.String()
}

func (v *ProjectDetailView) renderSkills(width int) string {
	skillsAgg := make(map[string]int)
	for _, s := range v.sessions {
		for skill, count := range s.SkillsUsed {
			skillsAgg[skill] += count
		}
	}
	if len(skillsAgg) == 0 {
		return "  No skills used in this project."
	}
	type skillEntry struct {
		name  string
		count int
	}
	var entries []skillEntry
	for name, count := range skillsAgg {
		entries = append(entries, skillEntry{name, count})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].count > entries[j].count })

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %-40s %10s\n", "Skill", "Uses"))
	b.WriteString("  " + strings.Repeat("─", 52) + "\n")
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("  %-40s %10d\n", e.name, e.count))
	}
	return b.String()
}

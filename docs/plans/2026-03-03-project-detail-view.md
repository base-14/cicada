# Project Detail View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a project detail drill-in view with 5 tabs (Overview, Sessions, Tools, Activity, Skills) accessible by pressing Enter on a selected project.

**Architecture:** Mirrors the existing `SessionDetailView` pattern — a new `ProjectDetailView` struct with sub-tab navigation, scrolling, and Esc to return. Data is aggregated from all `SessionMeta` objects for the selected project via `store.SessionsByProject()`. The `ProjectsView` exposes a `SelectedProject()` method, and `app.go` wires up Enter on the projects tab to open the detail view.

**Tech Stack:** Go, Bubbletea, Lipgloss, existing components (BarChart, Sparkline, Heatmap)

---

### Task 1: Add SelectedProject() to ProjectsView

Expose the currently selected project path so `app.go` can use it to open the detail view.

**Files:**
- Modify: `internal/tui/views/projects.go`
- Test: `internal/tui/views/projects_test.go`

**Step 1: Write the failing test**

Add to `internal/tui/views/projects_test.go`:

```go
func TestProjectsView_SelectedProject(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-proj1",
		StartTime: now.Add(-time.Hour), EndTime: now,
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "s2", ProjectPath: "-Users-r-work-proj2",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 1,
	})

	view := NewProjectsView(s)
	// Trigger a View() call first so rows get computed
	view.View(80, 24)

	path := view.SelectedProject()
	if path == "" {
		t.Error("expected non-empty selected project path")
	}

	// Navigate down and check selection changes
	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	view.View(80, 24)
	path2 := view.SelectedProject()
	if path2 == "" {
		t.Error("expected non-empty selected project after navigation")
	}
	if path == path2 {
		t.Error("expected different project after navigating down")
	}
}

func TestProjectsView_SelectedProject_Empty(t *testing.T) {
	s := store.New()
	view := NewProjectsView(s)
	view.View(80, 24)
	path := view.SelectedProject()
	if path != "" {
		t.Errorf("expected empty path for empty store, got %q", path)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `SelectedProject` undefined

**Step 3: Write minimal implementation**

In `internal/tui/views/projects.go`:

1. Add a field `lastRows []ProjectRow` to `ProjectsView` to cache the computed rows.
2. In `View()`, after building `rows`, store them: `v.lastRows = rows`.
3. Add the method:

```go
// SelectedProject returns the project path of the currently selected project, or empty string.
func (v *ProjectsView) SelectedProject() string {
	if len(v.lastRows) == 0 {
		return ""
	}
	idx := v.selected
	if idx >= len(v.lastRows) {
		idx = len(v.lastRows) - 1
	}
	return v.lastRows[idx].Path
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/views/projects.go internal/tui/views/projects_test.go
git commit -m "feat: add SelectedProject() to ProjectsView"
```

---

### Task 2: Create ProjectDetailView with Overview tab

Create the project detail view struct with sub-tab navigation, scrolling, and the Overview tab that shows aggregated project stats.

**Files:**
- Create: `internal/tui/views/project_detail.go`
- Create: `internal/tui/views/project_detail_test.go`

**Step 1: Write the failing tests**

Create `internal/tui/views/project_detail_test.go`:

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

func newTestProjectSessions() (*store.Store, string) {
	s := store.New()
	now := time.Now()
	project := "-Users-r-work-myproject"

	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "fix-login", ProjectPath: project,
		StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour),
		Duration:      time.Hour,
		InitialPrompt: "Fix the login bug",
		TokensIn: 10000, TokensOut: 5000,
		CacheRead: 1000, CacheWrite: 500,
		Models:       map[string]int{"claude-opus-4-6": 5},
		ToolUsage:    map[string]int{"Read": 10, "Edit": 5, "Bash": 3},
		SkillsUsed:   map[string]int{"tdd": 2},
		CommandsUsed: map[string]int{},
		FileOps:      map[string]int{"read": 10, "edit": 5},
		GitBranches:  []string{"main", "fix-login"},
		SubagentCount: 1,
		MessageCount:  20,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "add-tests", ProjectPath: project,
		StartTime: now.Add(-time.Hour), EndTime: now,
		Duration:      time.Hour,
		InitialPrompt: "Add unit tests",
		TokensIn: 8000, TokensOut: 4000,
		CacheRead: 800, CacheWrite: 400,
		Models:       map[string]int{"claude-sonnet-4-6": 3},
		ToolUsage:    map[string]int{"Read": 5, "Write": 8, "Bash": 6},
		SkillsUsed:   map[string]int{"tdd": 1, "debugging": 3},
		CommandsUsed: map[string]int{},
		FileOps:      map[string]int{"read": 5, "write": 8},
		GitBranches:  []string{"main"},
		SubagentCount: 0,
		MessageCount:  15,
	})

	return s, project
}

func TestNewProjectDetailView(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)
	if view == nil {
		t.Fatal("expected non-nil project detail view")
	}
	if view.activeTab != 0 {
		t.Errorf("expected activeTab=0, got %d", view.activeTab)
	}
}

func TestProjectDetailView_OverviewTab(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)
	content := view.View(100, 30)

	if content == "" {
		t.Error("expected non-empty overview")
	}
	// Should show project name
	if !strings.Contains(content, "myproject") {
		t.Error("expected project name in overview")
	}
	// Should show aggregated session count
	if !strings.Contains(content, "2") {
		t.Error("expected session count in overview")
	}
	// Should show Overview tab label
	if !strings.Contains(content, "Overview") {
		t.Error("expected 'Overview' tab label")
	}
}

func TestProjectDetailView_TabNavigation(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	// Right arrow to Sessions tab
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	if view.activeTab != 1 {
		t.Errorf("expected activeTab=1, got %d", view.activeTab)
	}

	// Left wraps from 0 to last
	view.activeTab = 0
	view.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if view.activeTab != 4 {
		t.Errorf("expected activeTab=4, got %d", view.activeTab)
	}
}

func TestProjectDetailView_Scrolling(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	if view.scrollY != 1 {
		t.Errorf("expected scrollY=1, got %d", view.scrollY)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.scrollY != 0 {
		t.Errorf("expected scrollY=0, got %d", view.scrollY)
	}
}

func TestProjectDetailView_EmptySessions(t *testing.T) {
	view := NewProjectDetailView("-empty-project", nil)
	content := view.View(100, 30)
	if content == "" {
		t.Error("expected non-empty view for empty project")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `NewProjectDetailView` undefined

**Step 3: Write minimal implementation**

Create `internal/tui/views/project_detail.go`:

```go
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

// ProjectDetailView shows detailed information about a project.
type ProjectDetailView struct {
	project   string
	sessions  []*model.SessionMeta
	activeTab int
	scrollY   int
}

// NewProjectDetailView creates a new ProjectDetailView.
func NewProjectDetailView(project string, sessions []*model.SessionMeta) *ProjectDetailView {
	// Sort sessions by start time descending (newest first)
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

// View renders the project detail view.
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

	// Content
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

	// Apply scroll
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

	name := v.decodedName()
	b.WriteString(fmt.Sprintf("  Project: %s\n\n", name))

	// Aggregate stats
	var totalTokensIn, totalTokensOut, totalCacheRead, totalCacheWrite int64
	var totalDuration time.Duration
	var totalMessages, totalSubagents, totalTools int
	models := make(map[string]int)
	branches := make(map[string]bool)
	var earliest, latest time.Time

	for _, s := range v.sessions {
		totalTokensIn += s.TokensIn
		totalTokensOut += s.TokensOut
		totalCacheRead += s.CacheRead
		totalCacheWrite += s.CacheWrite
		totalDuration += s.Duration
		totalMessages += s.MessageCount
		totalSubagents += s.SubagentCount

		for tool, count := range s.ToolUsage {
			totalTools += count
			_ = tool
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

	// Date range
	if !earliest.IsZero() {
		b.WriteString(fmt.Sprintf("  %-15s %s → %s\n", "Active:",
			earliest.Format("2006-01-02"),
			latest.Format("2006-01-02")))
		b.WriteString("\n")
	}

	// Models
	if len(models) > 0 {
		b.WriteString("  Models:\n")
		for name, count := range models {
			b.WriteString(fmt.Sprintf("    %-40s %d messages\n", name, count))
		}
		b.WriteString("\n")
	}

	// Work mode
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
		fmt.Fprintf(&b, "    Exploration %d%%    Building %d%%    Testing %d%%\n",
			explore*100/totalWork, build*100/totalWork, test*100/totalWork)
		b.WriteString("\n")
	}

	// Git branches
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
		dur := formatDuration(s.Duration)
		tokens := formatTokensShort(s.TokensIn + s.TokensOut)

		b.WriteString(fmt.Sprintf("  %-30s %-20s %-12s %10s\n", slug, date, dur, tokens))
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
	if chartWidth > 60 {
		chartWidth = 60
	}
	b.WriteString("  " + strings.ReplaceAll(components.BarChart(topTools, chartWidth), "\n", "\n  ") + "\n")

	return b.String()
}

func (v *ProjectDetailView) renderActivity(width int) string {
	var b strings.Builder

	// Sessions by date sparkline
	sessionsByDate := make(map[string]int)
	for _, s := range v.sessions {
		if !s.StartTime.IsZero() {
			date := s.StartTime.Format("2006-01-02")
			sessionsByDate[date]++
		}
	}

	if len(sessionsByDate) > 0 {
		b.WriteString("  Sessions (last 30 days)\n")
		sparkData := buildSparkData(sessionsByDate, 30)
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
		hour := s.StartTime.Hour()
		heatmap[day][hour]++
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
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %-40s %10s\n", "Skill", "Uses"))
	b.WriteString("  " + strings.Repeat("─", 52) + "\n")

	for _, e := range entries {
		b.WriteString(fmt.Sprintf("  %-40s %10d\n", e.name, e.count))
	}

	return b.String()
}
```

Note: This reuses `formatDuration`, `formatTokensShort`, `topNToolItems`, `buildSparkData` helpers that already exist in the `views` package (from `session_detail.go` and `analytics.go`).

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/views/project_detail.go internal/tui/views/project_detail_test.go
git commit -m "feat: add ProjectDetailView with all 5 tabs"
```

---

### Task 3: Wire up project detail in app.go

Connect the project detail view to the app: Enter on projects tab opens detail, Esc returns to list.

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/app_test.go`

**Step 1: Write the failing test**

Add to `internal/tui/app_test.go`:

```go
func TestApp_ProjectDetailDrillIn(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-proj1",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})

	app := NewApp(s, "/tmp/test")
	// Set window size and switch to projects tab
	app, _ = app.Update(tea.WindowSizeMsg{Width: 100, Height: 30}).(App), nil
	app.activeTab = 1
	// Trigger View to populate lastRows
	app.View()
	// Press Enter to open project detail
	result, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = result.(App)

	if !app.showingProjectDetail {
		t.Error("expected showingProjectDetail to be true after Enter on projects tab")
	}
	if app.projectDetailView == nil {
		t.Error("expected projectDetailView to be non-nil")
	}
}

func TestApp_ProjectDetailEscReturns(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-proj1",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})

	app := NewApp(s, "/tmp/test")
	app, _ = app.Update(tea.WindowSizeMsg{Width: 100, Height: 30}).(App), nil
	app.activeTab = 1
	app.View()
	result, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = result.(App)

	// Press Esc to close
	result, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = result.(App)

	if app.showingProjectDetail {
		t.Error("expected showingProjectDetail=false after Esc")
	}
	if app.projectDetailView != nil {
		t.Error("expected projectDetailView=nil after Esc")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `showingProjectDetail` and `projectDetailView` undefined

**Step 3: Write minimal implementation**

In `internal/tui/app.go`:

1. Add fields to `App` struct:

```go
projectDetailView    *views.ProjectDetailView
showingProjectDetail bool
```

2. In the `Update` method's `tea.KeyMsg` handler, add project detail handling right after the session detail block (before the general key switch). When `showingProjectDetail`, forward keys to it except Esc and Ctrl+C:

```go
// When showing project detail view
if a.showingProjectDetail && a.projectDetailView != nil {
    switch msg.Type {
    case tea.KeyEsc:
        a.showingProjectDetail = false
        a.projectDetailView = nil
        return a, nil
    case tea.KeyCtrlC:
        return a, tea.Quit
    default:
        a.projectDetailView.Update(msg)
        return a, nil
    }
}
```

3. In the Enter key handler, add a case for `activeTab == 1`:

```go
case tea.KeyEnter:
    if a.activeTab == 1 {
        a.openProjectDetail()
    }
    if a.activeTab == 2 {
        a.openSessionDetail()
    }
    return a, nil
```

4. Add the `openProjectDetail` method:

```go
func (a *App) openProjectDetail() {
    project := a.projectsView.SelectedProject()
    if project == "" {
        return
    }
    sessions := a.store.SessionsByProject(project)
    a.projectDetailView = views.NewProjectDetailView(project, sessions)
    a.showingProjectDetail = true
}
```

5. In `renderContent()`, add project detail rendering before the switch, after the session detail check:

```go
if a.showingProjectDetail && a.projectDetailView != nil && a.activeTab == 1 {
    return a.projectDetailView.View(a.width, a.height-4)
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat: wire up project detail drill-in on Enter"
```

---

### Task 4: Add tab-specific tests for all 5 project detail tabs

Add tests for Sessions, Tools, Activity, and Skills tabs to ensure all render correctly.

**Files:**
- Modify: `internal/tui/views/project_detail_test.go`

**Step 1: Write the tests**

Add to `internal/tui/views/project_detail_test.go`:

```go
func TestProjectDetailView_SessionsTab(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	// Switch to Sessions tab
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	content := view.View(100, 30)

	if !strings.Contains(content, "Sessions") {
		t.Error("expected 'Sessions' tab label")
	}
	if !strings.Contains(content, "fix-login") {
		t.Error("expected session slug 'fix-login' in sessions list")
	}
	if !strings.Contains(content, "add-tests") {
		t.Error("expected session slug 'add-tests' in sessions list")
	}
}

func TestProjectDetailView_ToolsTab(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	// Switch to Tools tab (index 2)
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	view.Update(tea.KeyMsg{Type: tea.KeyRight})
	content := view.View(100, 30)

	if !strings.Contains(content, "Tools") {
		t.Error("expected 'Tools' tab label")
	}
	if !strings.Contains(content, "Read") {
		t.Error("expected 'Read' tool in tools tab")
	}
}

func TestProjectDetailView_ActivityTab(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	// Switch to Activity tab (index 3)
	for i := 0; i < 3; i++ {
		view.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
	content := view.View(100, 30)

	if !strings.Contains(content, "Activity") {
		t.Error("expected 'Activity' tab label")
	}
	if !strings.Contains(content, "Heatmap") {
		t.Error("expected heatmap in activity tab")
	}
}

func TestProjectDetailView_SkillsTab(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	// Switch to Skills tab (index 4)
	for i := 0; i < 4; i++ {
		view.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
	content := view.View(100, 30)

	if !strings.Contains(content, "Skills") {
		t.Error("expected 'Skills' tab label")
	}
	if !strings.Contains(content, "tdd") {
		t.Error("expected 'tdd' skill in skills tab")
	}
	if !strings.Contains(content, "debugging") {
		t.Error("expected 'debugging' skill in skills tab")
	}
}

func TestProjectDetailView_VimKeys(t *testing.T) {
	s, project := newTestProjectSessions()
	sessions := s.SessionsByProject(project)
	view := NewProjectDetailView(project, sessions)

	// h/l for tab navigation
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if view.activeTab != 1 {
		t.Errorf("expected tab 1 after 'l', got %d", view.activeTab)
	}
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if view.activeTab != 0 {
		t.Errorf("expected tab 0 after 'h', got %d", view.activeTab)
	}

	// j/k for scrolling
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

**Step 2: Run tests**

Run: `make test`
Expected: PASS (all tests should pass since implementation is already complete)

**Step 3: Commit**

```bash
git add internal/tui/views/project_detail_test.go
git commit -m "test: add comprehensive tests for all project detail tabs"
```

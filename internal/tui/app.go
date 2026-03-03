package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/r/cicada/internal/store"
	"github.com/r/cicada/internal/tui/components"
	"github.com/r/cicada/internal/tui/views"
)

var tabNames = []string{"Dashboard", "Projects", "Sessions", "Analytics", "Agents", "Tools"}

// Messages from the scanner goroutine
type ScanBatchMsg struct {
	Scanned int
	Total   int
}

// ScanCompleteMsg is sent when the background scan finishes.
type ScanCompleteMsg struct{}

// App is the root Bubbletea model.
type App struct {
	store        *store.Store
	styles       Styles
	activeTab    int
	width        int
	height       int
	scanScanned  int
	scanTotal    int
	scanDone     bool
	projectsView *views.ProjectsView
	sessionsView *views.SessionsView
}

// NewApp creates a new App model.
func NewApp(s *store.Store) App {
	theme := DefaultTheme()
	return App{
		store:        s,
		styles:       NewStyles(theme),
		projectsView: views.NewProjectsView(s),
		sessionsView: views.NewSessionsView(s),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			a.activeTab = (a.activeTab + 1) % len(tabNames)
			return a, nil
		case tea.KeyShiftTab:
			a.activeTab = (a.activeTab - 1 + len(tabNames)) % len(tabNames)
			return a, nil
		case tea.KeyEsc:
			return a, nil
		case tea.KeyCtrlC:
			return a, tea.Quit
		case tea.KeyUp, tea.KeyDown:
			// Forward navigation keys to the active view
			if a.activeTab == 2 {
				a.sessionsView.Update(msg)
			}
			return a, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				return a, tea.Quit
			case "1", "2", "3", "4", "5", "6":
				idx := int(msg.Runes[0]-'0') - 1
				if idx < len(tabNames) {
					a.activeTab = idx
				}
				return a, nil
			case "j", "k":
				if a.activeTab == 2 {
					a.sessionsView.Update(msg)
				}
				return a, nil
			}
		}

	case ScanBatchMsg:
		a.scanScanned = msg.Scanned
		a.scanTotal = msg.Total
		return a, nil

	case ScanCompleteMsg:
		a.scanDone = true
		return a, nil
	}

	return a, nil
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Tab bar
	b.WriteString(a.renderTabBar())
	b.WriteString("\n")

	// Content area
	contentHeight := a.height - 4
	content := a.renderContent()
	contentStyle := lipgloss.NewStyle().Height(contentHeight).Width(a.width)
	b.WriteString(contentStyle.Render(content))
	b.WriteString("\n")

	// Status bar
	b.WriteString(a.renderStatusBar())

	return b.String()
}

func (a App) renderTabBar() string {
	var tabs []string
	for i, name := range tabNames {
		if i == a.activeTab {
			tabs = append(tabs, a.styles.TabActive.Render(name))
		} else {
			tabs = append(tabs, a.styles.TabInactive.Render(name))
		}
	}
	title := a.styles.Title.Render("cicada")
	return title + " " + strings.Join(tabs, "")
}

func (a App) renderContent() string {
	switch a.activeTab {
	case 0:
		return a.renderDashboard()
	case 1:
		return a.projectsView.View(a.width, a.height-4)
	case 2:
		return a.sessionsView.View(a.width, a.height-4)
	default:
		return fmt.Sprintf("  %s view — coming soon", tabNames[a.activeTab])
	}
}

func (a App) renderDashboard() string {
	analytics := a.store.Analytics()

	var b strings.Builder
	b.WriteString("\n")

	// Stats row
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s    %s %s\n",
		a.styles.StatLabel.Render("Sessions:"),
		a.styles.StatValue.Render(fmt.Sprintf("%d", analytics.TotalSessions)),
		a.styles.StatLabel.Render("Tokens In:"),
		a.styles.StatValue.Render(formatTokens(analytics.TotalTokensIn)),
		a.styles.StatLabel.Render("Tokens Out:"),
		a.styles.StatValue.Render(formatTokens(analytics.TotalTokensOut)),
		a.styles.StatLabel.Render("Projects:"),
		a.styles.StatValue.Render(fmt.Sprintf("%d", analytics.ActiveProjects)),
	)
	b.WriteString("\n")

	// Sessions by date sparkline (last 30 days)
	if len(analytics.SessionsByDate) > 0 {
		b.WriteString("  " + a.styles.Subtitle.Render("Sessions (last 30 days)") + "\n")
		sparkData := buildSparklineData(analytics.SessionsByDate, 30)
		sparkWidth := a.width - 4
		if sparkWidth > 60 {
			sparkWidth = 60
		}
		if sparkWidth < 10 {
			sparkWidth = 10
		}
		b.WriteString("  " + components.Sparkline(sparkData, sparkWidth) + "\n\n")
	}

	// Top 5 tools bar chart
	if len(analytics.ToolsUsed) > 0 {
		b.WriteString("  " + a.styles.Subtitle.Render("Top Tools") + "\n")
		topTools := topNTools(analytics.ToolsUsed, 5)
		chartWidth := a.width - 4
		if chartWidth > 60 {
			chartWidth = 60
		}
		b.WriteString("  " + strings.ReplaceAll(components.BarChart(topTools, chartWidth), "\n", "\n  ") + "\n\n")
	}

	// Model distribution
	if len(analytics.ModelsUsed) > 0 {
		b.WriteString("  " + a.styles.Subtitle.Render("Models") + "\n")
		opus, sonnet, haiku, other := categorizeModels(analytics.ModelsUsed)
		total := opus + sonnet + haiku + other
		if total > 0 {
			if opus > 0 {
				fmt.Fprintf(&b, "  Opus   %d%%  ", opus*100/total)
			}
			if sonnet > 0 {
				fmt.Fprintf(&b, "  Sonnet %d%%  ", sonnet*100/total)
			}
			if haiku > 0 {
				fmt.Fprintf(&b, "  Haiku  %d%%  ", haiku*100/total)
			}
			if other > 0 {
				fmt.Fprintf(&b, "  Other  %d%%  ", other*100/total)
			}
			b.WriteString("\n\n")
		}
	}

	// Work mode split
	totalWork := analytics.WorkModeExplore + analytics.WorkModeBuild + analytics.WorkModeTest
	if totalWork > 0 {
		b.WriteString("  " + a.styles.Subtitle.Render("Work Mode") + "\n")
		fmt.Fprintf(&b, "  Exploration %d%%    Building %d%%    Testing %d%%\n",
			analytics.WorkModeExplore*100/totalWork,
			analytics.WorkModeBuild*100/totalWork,
			analytics.WorkModeTest*100/totalWork,
		)
	}

	return b.String()
}

// buildSparklineData returns session counts for the last n days, sorted by date.
func buildSparklineData(sessionsByDate map[string]int, days int) []int {
	now := time.Now()
	data := make([]int, days)
	for i := range days {
		date := now.AddDate(0, 0, -(days-1-i)).Format("2006-01-02")
		data[i] = sessionsByDate[date]
	}
	return data
}

// topNTools returns the top n tools by usage as BarItems.
func topNTools(toolsUsed map[string]int, n int) []components.BarItem {
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

// categorizeModels buckets model usage into Opus, Sonnet, Haiku, and Other.
func categorizeModels(modelsUsed map[string]int) (opus, sonnet, haiku, other int) {
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

func (a App) renderStatusBar() string {
	var status string
	if a.scanDone {
		status = fmt.Sprintf("Ready — %d sessions indexed", a.scanScanned)
	} else if a.scanTotal > 0 {
		status = fmt.Sprintf("Scanning... %d/%d sessions", a.scanScanned, a.scanTotal)
	} else {
		status = "Discovering projects..."
	}

	help := "? help  q quit"
	gap := a.width - len(status) - len(help) - 4
	if gap < 0 {
		gap = 1
	}
	return a.styles.StatusBar.Width(a.width).Render(
		"  " + status + strings.Repeat(" ", gap) + help,
	)
}

func formatTokens(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

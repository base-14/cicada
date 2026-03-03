package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/r/cicada/internal/store"
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
	store       *store.Store
	styles      Styles
	activeTab   int
	width       int
	height      int
	scanScanned int
	scanTotal   int
	scanDone    bool
}

// NewApp creates a new App model.
func NewApp(s *store.Store) App {
	theme := DefaultTheme()
	return App{
		store:  s,
		styles: NewStyles(theme),
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
	default:
		return fmt.Sprintf("  %s view — coming soon", tabNames[a.activeTab])
	}
}

func (a App) renderDashboard() string {
	analytics := a.store.Analytics()

	var b strings.Builder
	b.WriteString("\n")
	fmt.Fprintf(&b, "  %s %s    %s %s    %s %s\n",
		a.styles.StatLabel.Render("Sessions:"),
		a.styles.StatValue.Render(fmt.Sprintf("%d", analytics.TotalSessions)),
		a.styles.StatLabel.Render("Tokens In:"),
		a.styles.StatValue.Render(formatTokens(analytics.TotalTokensIn)),
		a.styles.StatLabel.Render("Projects:"),
		a.styles.StatValue.Render(fmt.Sprintf("%d", analytics.ActiveProjects)),
	)
	return b.String()
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

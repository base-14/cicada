package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/base-14/cicada/internal/parser"
	"github.com/base-14/cicada/internal/store"
	"github.com/base-14/cicada/internal/tui/views"
)

var tabNames = []string{"Analysis", "Projects", "Sessions", "Agents", "Tools"}

// Messages from the scanner goroutine
type ScanBatchMsg struct {
	Scanned int
	Total   int
}

// ScanCompleteMsg is sent when the background scan finishes.
type ScanCompleteMsg struct{}

// HistoryScanCompleteMsg is sent when history.jsonl scanning finishes.
type HistoryScanCompleteMsg struct{ Count int }

// App is the root Bubbletea model.
type App struct {
	store          *store.Store
	styles         Styles
	activeTab      int
	width          int
	height         int
	scanScanned    int
	scanTotal      int
	scanDone       bool
	historyCount   int
	projectsView   *views.ProjectsView
	sessionsView   *views.SessionsView
	analysisView   *views.AnalysisView
	agentsView     *views.AgentsView
	toolsView      *views.ToolsView
	detailView           *views.SessionDetailView
	showingDetail        bool
	projectDetailView    *views.ProjectDetailView
	showingProjectDetail bool
	showingHelp          bool
	projectsDir          string // path to ~/.claude/projects
}

// NewApp creates a new App model. projectsDir is the path to ~/.claude/projects.
func NewApp(s *store.Store, projectsDir string) App {
	theme := DefaultTheme()
	return App{
		store:         s,
		styles:        NewStyles(theme),
		projectsView:  views.NewProjectsView(s),
		sessionsView:  views.NewSessionsView(s),
		analysisView:  views.NewAnalysisView(s),
		agentsView:    views.NewAgentsView(s),
		toolsView:     views.NewToolsView(s),
		projectsDir:   projectsDir,
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
		// When showing help overlay, any key dismisses it
		if a.showingHelp {
			a.showingHelp = false
			return a, nil
		}

		// When showing detail view, forward all keys to it except Esc and ctrl+c
		if a.showingDetail && a.detailView != nil {
			switch msg.Type {
			case tea.KeyEsc:
				a.showingDetail = false
				a.detailView = nil
				return a, nil
			case tea.KeyCtrlC:
				return a, tea.Quit
			default:
				a.detailView.Update(msg)
				return a, nil
			}
		}

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

		switch msg.Type {
		case tea.KeyTab:
			a.activeTab = (a.activeTab + 1) % len(tabNames)
			return a, nil
		case tea.KeyShiftTab:
			a.activeTab = (a.activeTab - 1 + len(tabNames)) % len(tabNames)
			return a, nil
		case tea.KeyCtrlC:
			return a, tea.Quit
		case tea.KeyUp, tea.KeyDown:
			// Forward navigation keys to the active view
			switch a.activeTab {
			case 0:
				a.analysisView.Update(msg)
			case 1:
				a.projectsView.Update(msg)
			case 2:
				a.sessionsView.Update(msg)
			case 4:
				a.toolsView.Update(msg)
			}
			return a, nil
		case tea.KeyEsc:
			// Forward Esc to views with active filters
			switch a.activeTab {
			case 1:
				a.projectsView.Update(msg)
			case 2:
				a.sessionsView.Update(msg)
			}
			return a, nil
		case tea.KeyBackspace:
			// Forward backspace to views with active filters
			switch a.activeTab {
			case 1:
				a.projectsView.Update(msg)
			case 2:
				a.sessionsView.Update(msg)
			}
			return a, nil
		case tea.KeyEnter:
			if a.activeTab == 1 {
				a.openProjectDetail()
			}
			if a.activeTab == 2 {
				a.openSessionDetail()
			}
			return a, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "?":
				a.showingHelp = true
				return a, nil
			case "/":
				// Forward '/' to activate filter on views that support it
				switch a.activeTab {
				case 1:
					a.projectsView.Update(msg)
				case 2:
					a.sessionsView.Update(msg)
				}
				return a, nil
			case "q":
				// Don't quit if filter is active
				if a.activeTab == 1 && a.projectsView.FilterActive() {
					a.projectsView.Update(msg)
					return a, nil
				}
				if a.activeTab == 2 && a.sessionsView.FilterActive() {
					a.sessionsView.Update(msg)
					return a, nil
				}
				return a, tea.Quit
			case "1", "2", "3", "4", "5":
				// Don't switch tabs if filter is active
				if a.activeTab == 1 && a.projectsView.FilterActive() {
					a.projectsView.Update(msg)
					return a, nil
				}
				if a.activeTab == 2 && a.sessionsView.FilterActive() {
					a.sessionsView.Update(msg)
					return a, nil
				}
				idx := int(msg.Runes[0]-'0') - 1
				if idx < len(tabNames) {
					a.activeTab = idx
				}
				return a, nil
			case "j", "k":
				switch a.activeTab {
				case 0:
					a.analysisView.Update(msg)
				case 1:
					a.projectsView.Update(msg)
				case 2:
					a.sessionsView.Update(msg)
				case 4:
					a.toolsView.Update(msg)
				}
				return a, nil
			default:
				// Forward any other runes to views with active filters
				if a.activeTab == 1 && a.projectsView.FilterActive() {
					a.projectsView.Update(msg)
					return a, nil
				}
				if a.activeTab == 2 && a.sessionsView.FilterActive() {
					a.sessionsView.Update(msg)
					return a, nil
				}
			}
		}

	case ScanBatchMsg:
		a.scanScanned = msg.Scanned
		a.scanTotal = msg.Total
		return a, nil

	case ScanCompleteMsg:
		a.scanDone = true
		return a, nil

	case HistoryScanCompleteMsg:
		a.historyCount = msg.Count
		return a, nil
	}

	return a, nil
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Help overlay takes over the entire view
	if a.showingHelp {
		return a.renderHelpOverlay()
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

func (a App) renderHelpOverlay() string {
	help := `  🦗 cicada — Claude Code Session Analyzer

  Navigation
    1-5            Switch view
    Tab/Shift+Tab  Next/prev view
    Enter          Open selected item
    Esc            Go back

  Lists
    ↑/↓ j/k       Navigate rows
    /              Search/filter

  Session Detail
    ←/→ h/l       Switch sub-tab
    ↑/↓ j/k       Scroll content

  General
    ?              Toggle this help
    q              Quit

  Press any key to dismiss`

	style := lipgloss.NewStyle().
		Width(a.width).
		Height(a.height).
		Padding(2, 4)

	return style.Render(help)
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
	title := a.styles.Title.Render("🦗")
	return title + " " + strings.Join(tabs, "")
}

func (a *App) openSessionDetail() {
	session := a.sessionsView.SelectedSession()
	if session == nil {
		return
	}

	// Check cache first
	detail := a.store.GetDetail(session.UUID)
	if detail == nil {
		// Lazy load: parse JSONL file
		jsonlPath := filepath.Join(a.projectsDir, session.ProjectPath, session.UUID+".jsonl")
		messages, err := parser.ReadSessionFile(jsonlPath)
		if err != nil {
			// Can't load detail, silently fail
			return
		}
		detail = parser.ExtractSessionDetail(messages, session)
		a.store.SetDetail(session.UUID, detail)
	}

	a.detailView = views.NewSessionDetailView(a.store, session, detail)
	a.showingDetail = true
}

func (a *App) openProjectDetail() {
	project := a.projectsView.SelectedProject()
	if project == "" {
		return
	}
	sessions := a.store.SessionsByProject(project)
	a.projectDetailView = views.NewProjectDetailView(project, sessions)
	a.showingProjectDetail = true
}

func (a App) renderContent() string {
	// Show detail view when drilling in from sessions
	if a.showingDetail && a.detailView != nil && a.activeTab == 2 {
		return a.detailView.View(a.width, a.height-4)
	}

	// Show project detail view when drilling in from projects
	if a.showingProjectDetail && a.projectDetailView != nil && a.activeTab == 1 {
		return a.projectDetailView.View(a.width, a.height-4)
	}

	switch a.activeTab {
	case 0:
		return a.analysisView.View(a.width, a.height-4)
	case 1:
		return a.projectsView.View(a.width, a.height-4)
	case 2:
		return a.sessionsView.View(a.width, a.height-4)
	case 3:
		return a.agentsView.View(a.width, a.height-4)
	case 4:
		return a.toolsView.View(a.width, a.height-4)
	default:
		return fmt.Sprintf("  %s view — coming soon", tabNames[a.activeTab])
	}
}

func (a App) renderStatusBar() string {
	var status string
	if a.scanDone {
		if a.historyCount > 0 {
			status = fmt.Sprintf("Ready — %d sessions, %d prompts indexed", a.scanScanned, a.historyCount)
		} else {
			status = fmt.Sprintf("Ready — %d sessions indexed", a.scanScanned)
		}
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


package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds the color palette for the TUI.
type Theme struct {
	Primary   string
	Secondary string
	Accent    string
	Muted     string
	Error     string
	Success   string
	Warning   string
	BgDark    string
	BgLight   string
	Fg        string
	FgDim     string
}

// DefaultTheme returns the default color theme.
func DefaultTheme() Theme {
	return Theme{
		Primary:   "#00FFFF",
		Secondary: "#06B6D4",
		Accent:    "#F59E0B",
		Muted:     "#6B7280",
		Error:     "#EF4444",
		Success:   "#10B981",
		Warning:   "#F59E0B",
		BgDark:    "#1F2937",
		BgLight:   "#374151",
		Fg:        "#F9FAFB",
		FgDim:     "#9CA3AF",
	}
}

// Styles holds pre-built lipgloss styles.
type Styles struct {
	TabBar      lipgloss.Style
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	StatusBar   lipgloss.Style
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	StatLabel   lipgloss.Style
	StatValue   lipgloss.Style
	Selected    lipgloss.Style
	ViewPort    lipgloss.Style
}

// NewStyles creates a Styles from a Theme.
func NewStyles(t Theme) Styles {
	return Styles{
		TabBar:      lipgloss.NewStyle().Background(lipgloss.Color(t.BgDark)).Padding(0, 1),
		TabActive:   lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true).Padding(0, 2),
		TabInactive: lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgDim)).Padding(0, 2),
		StatusBar:   lipgloss.NewStyle().Background(lipgloss.Color(t.BgDark)).Foreground(lipgloss.Color(t.FgDim)).Padding(0, 1),
		Title:       lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true),
		Subtitle:    lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary)),
		StatLabel:   lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgDim)),
		StatValue:   lipgloss.NewStyle().Foreground(lipgloss.Color(t.Fg)).Bold(true),
		Selected:    lipgloss.NewStyle().Background(lipgloss.Color(t.BgLight)).Foreground(lipgloss.Color(t.Fg)),
		ViewPort:    lipgloss.NewStyle().Padding(1, 2),
	}
}

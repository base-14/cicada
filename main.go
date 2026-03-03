package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/store"
	"github.com/r/cicada/internal/tui"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	projectsDir := filepath.Join(homeDir, ".claude", "projects")

	st := store.New()
	app := tui.NewApp(st)

	p := tea.NewProgram(app, tea.WithAltScreen())

	// Start background scanner
	go func() {
		msgCh := make(chan store.ScanMsg, 100)
		scanner := store.NewScanner(st, projectsDir)

		go scanner.Run(msgCh)

		for msg := range msgCh {
			switch msg.Type {
			case store.SessionsBatch:
				p.Send(tui.ScanBatchMsg{Scanned: msg.Scanned, Total: msg.Total})
			case store.ScanComplete:
				p.Send(tui.ScanCompleteMsg{})
				return
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

package store

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/r/cicada/internal/parser"
)

// ScanMsgType identifies the type of scan progress message.
type ScanMsgType int

const (
	// ProjectsDiscovered is sent after discovering project directories.
	ProjectsDiscovered ScanMsgType = iota
	// SessionsBatch is sent after parsing a batch of session files.
	SessionsBatch
	// ScanComplete is sent when the scan finishes.
	ScanComplete
)

// ScanMsg is sent from the scanner to report progress.
type ScanMsg struct {
	Type     ScanMsgType
	Projects []string
	Count    int
	Scanned  int
	Total    int
}

// Scanner discovers and parses JSONL session files.
type Scanner struct {
	store   *Store
	baseDir string
}

// NewScanner creates a new scanner.
func NewScanner(store *Store, baseDir string) *Scanner {
	return &Scanner{
		store:   store,
		baseDir: baseDir,
	}
}

// Run performs the scan synchronously, sending progress messages to msgCh.
// Call this in a goroutine for background scanning.
func (sc *Scanner) Run(msgCh chan<- ScanMsg) {
	entries, err := os.ReadDir(sc.baseDir)
	if err != nil {
		msgCh <- ScanMsg{Type: ScanComplete}
		return
	}

	var projectDirs []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			projectDirs = append(projectDirs, entry.Name())
		}
	}

	msgCh <- ScanMsg{Type: ProjectsDiscovered, Projects: projectDirs}

	type sessionFile struct {
		project string
		path    string
	}

	var allFiles []sessionFile
	for _, projName := range projectDirs {
		projPath := filepath.Join(sc.baseDir, projName)
		files, _ := filepath.Glob(filepath.Join(projPath, "*.jsonl"))
		for _, f := range files {
			allFiles = append(allFiles, sessionFile{projName, f})
		}
	}

	total := len(allFiles)
	sc.store.SetScanProgress(0, total)

	batchSize := 50
	scanned := 0

	for _, f := range allFiles {
		messages, err := parser.ReadSessionFile(f.path)
		if err != nil {
			scanned++
			continue
		}

		meta := parser.ExtractSessionMeta(messages, f.project, filepath.Base(f.path))
		sc.store.Add(meta)
		scanned++
		sc.store.SetScanProgress(scanned, total)

		if scanned%batchSize == 0 || scanned == total {
			msgCh <- ScanMsg{
				Type:    SessionsBatch,
				Count:   batchSize,
				Scanned: scanned,
				Total:   total,
			}
		}
	}

	msgCh <- ScanMsg{Type: ScanComplete, Scanned: scanned, Total: total}
}

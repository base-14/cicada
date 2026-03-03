package store

import (
	"bufio"
	"encoding/json"
	"os"
	"time"

	"github.com/r/cicada/internal/model"
)

type historyLine struct {
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
}

// ScanHistory reads and parses a history.jsonl file into HistoryEntry slices.
func ScanHistory(path string) ([]model.HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []model.HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var hl historyLine
		if err := json.Unmarshal(scanner.Bytes(), &hl); err != nil {
			continue // skip malformed lines
		}
		if hl.Timestamp == 0 {
			continue
		}
		entries = append(entries, model.HistoryEntry{
			Timestamp: time.UnixMilli(hl.Timestamp),
			Project:   hl.Project,
			SessionID: hl.SessionID,
		})
	}
	return entries, nil
}

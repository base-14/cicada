package parser

import (
	"bufio"
	"fmt"
	"os"
)

// ReadSessionFile reads a JSONL session file and returns all successfully parsed messages.
// Invalid lines are silently skipped.
func ReadSessionFile(path string) ([]*Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session file %s: %w", path, err)
	}
	defer f.Close()

	var messages []*Message
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		msg, err := ParseMessage(line)
		if err != nil {
			continue // skip bad lines
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		return messages, fmt.Errorf("scan session file %s: %w", path, err)
	}
	return messages, nil
}

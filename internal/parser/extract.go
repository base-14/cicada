package parser

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/r/cicada/internal/model"
)

// fileToolMappings maps file tool names to operation type categories.
var fileToolMappings = map[string]string{
	"Read":           "read",
	"Write":          "write",
	"Edit":           "edit",
	"StrReplace":     "edit",
	"Delete":         "delete",
	"Glob":           "search",
	"LS":             "read",
	"Grep":           "search",
	"SemanticSearch": "search",
}

// ExtractSessionMeta extracts lightweight metadata from parsed messages.
func ExtractSessionMeta(messages []*Message, projectPath string, filename string) *model.SessionMeta {
	meta := &model.SessionMeta{
		ProjectPath:  projectPath,
		Models:       make(map[string]int),
		ToolUsage:    make(map[string]int),
		SkillsUsed:   make(map[string]int),
		CommandsUsed: make(map[string]int),
		FileOps:      make(map[string]int),
	}

	// Extract UUID from filename (strip .jsonl extension)
	meta.UUID = strings.TrimSuffix(filepath.Base(filename), ".jsonl")

	branchSet := make(map[string]bool)
	var firstUserContent string
	var firstTimestamp, lastTimestamp time.Time
	firstTimestampSet := false

	for _, msg := range messages {
		// Track timestamps
		if !msg.Timestamp.IsZero() {
			if !firstTimestampSet {
				firstTimestamp = msg.Timestamp
				firstTimestampSet = true
			}
			lastTimestamp = msg.Timestamp
		}

		// Extract slug from any message
		if meta.Slug == "" && msg.Slug != "" {
			meta.Slug = msg.Slug
		}

		// Collect git branches
		if msg.GitBranch != "" && msg.GitBranch != "HEAD" {
			branchSet[msg.GitBranch] = true
		}

		switch msg.Type {
		case "summary":
			if msg.Summary != "" {
				meta.SessionTitles = append(meta.SessionTitles, msg.Summary)
			}

		case "user":
			if !msg.IsSidechain {
				meta.MessageCount++
				if firstUserContent == "" {
					firstUserContent = msg.UserContent()
				}
			}

		case "assistant":
			if !msg.IsSidechain {
				meta.MessageCount++
				modelName := msg.Model()
				if modelName != "" {
					meta.Models[modelName]++
				}

				usage := msg.Usage()
				if usage != nil {
					meta.TokensIn += int64(usage.InputTokens)
					meta.TokensOut += int64(usage.OutputTokens)
					meta.CacheRead += int64(usage.CacheReadInputTokens)
					meta.CacheWrite += int64(usage.CacheCreationInputTokens)
				}

				for _, tool := range msg.ToolUseBlocks() {
					meta.ToolUsage[tool.Name]++

					// Track skills
					if tool.Name == "Skill" {
						if skill := tool.SkillName(); skill != "" {
							meta.SkillsUsed[skill]++
						}
					}

					// Track file operations
					if opType, ok := fileToolMappings[tool.Name]; ok {
						meta.FileOps[opType]++
					}
				}
			}
		}
	}

	meta.InitialPrompt = firstUserContent
	meta.StartTime = firstTimestamp
	meta.EndTime = lastTimestamp
	if !firstTimestamp.IsZero() && !lastTimestamp.IsZero() {
		meta.Duration = lastTimestamp.Sub(firstTimestamp)
	}

	for branch := range branchSet {
		meta.GitBranches = append(meta.GitBranches, branch)
	}

	return meta
}

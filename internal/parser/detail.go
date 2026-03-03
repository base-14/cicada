package parser

import "github.com/r/cicada/internal/model"

// ExtractSessionDetail extracts full timeline and file activity from parsed messages.
func ExtractSessionDetail(messages []*Message, meta *model.SessionMeta) *model.SessionDetail {
	detail := &model.SessionDetail{Meta: meta}

	for _, msg := range messages {
		if msg.IsSidechain {
			continue
		}

		switch msg.Type {
		case "user":
			detail.Timeline = append(detail.Timeline, model.TimelineEvent{
				Timestamp: msg.Timestamp,
				Type:      "user",
				Content:   truncate(msg.UserContent(), 200),
			})

		case "assistant":
			// Add tool_use events for each tool block
			for _, tool := range msg.ToolUseBlocks() {
				detail.Timeline = append(detail.Timeline, model.TimelineEvent{
					Timestamp: msg.Timestamp,
					Type:      "tool_use",
					ToolName:  tool.Name,
					Content:   truncate(tool.FileToolPath(), 200),
				})

				// Track file operations
				if opType, ok := fileToolMappings[tool.Name]; ok {
					detail.FileActivity = append(detail.FileActivity, model.FileOp{
						Timestamp: msg.Timestamp,
						Path:      tool.FileToolPath(),
						Operation: opType,
						Actor:     meta.UUID,
						ActorType: "session",
						ToolName:  tool.Name,
					})
				}
			}
		}
	}

	return detail
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

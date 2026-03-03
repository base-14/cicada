package parser

import (
	"fmt"

	"github.com/r/cicada/internal/model"
)

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
			detail.ChatMessages = append(detail.ChatMessages, model.ChatMessage{
				Role:      "user",
				Content:   msg.UserContent(),
				Timestamp: msg.Timestamp,
			})

		case "assistant":
			// Iterate all content blocks for chat messages, timeline, and file activity
			for _, block := range msg.assistantContent() {
				switch block.Type {
				case "text":
					if block.Text != "" {
						detail.ChatMessages = append(detail.ChatMessages, model.ChatMessage{
							Role:      "assistant",
							Content:   block.Text,
							Timestamp: msg.Timestamp,
						})
					}
				case "tool_use":
					// Timeline event (existing behavior)
					detail.Timeline = append(detail.Timeline, model.TimelineEvent{
						Timestamp: msg.Timestamp,
						Type:      "tool_use",
						ToolName:  block.Name,
						Content:   truncate(block.FileToolPath(), 200),
					})

					// Track file operations (existing behavior)
					if opType, ok := fileToolMappings[block.Name]; ok {
						detail.FileActivity = append(detail.FileActivity, model.FileOp{
							Timestamp: msg.Timestamp,
							Path:      block.FileToolPath(),
							Operation: opType,
							Actor:     meta.UUID,
							ActorType: "session",
							ToolName:  block.Name,
						})
					}

					// Chat message for tool use
					detail.ChatMessages = append(detail.ChatMessages, model.ChatMessage{
						Role:      "tool",
						Content:   toolSummary(block),
						Timestamp: msg.Timestamp,
						ToolName:  block.Name,
					})
				}
			}
		}
	}

	return detail
}

// toolSummary returns a one-liner description of a tool_use block.
func toolSummary(block ContentBlock) string {
	path := block.FileToolPath()
	if path != "" {
		return fmt.Sprintf("%s → %s", block.Name, path)
	}
	return block.Name
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

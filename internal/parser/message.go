package parser

import (
	"encoding/json"
	"fmt"
	"time"
)

// Message is the top-level parsed representation of a JSONL line.
type Message struct {
	Type      string    `json:"type"`
	UUID      string    `json:"uuid"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"sessionId"`
	Slug      string    `json:"slug"`
	CWD       string    `json:"cwd"`
	GitBranch string    `json:"gitBranch"`

	IsSidechain bool   `json:"isSidechain"`
	IsMeta      bool   `json:"isMeta"`
	AgentID     string `json:"agentId"`

	// type=summary
	Summary  string `json:"summary"`
	LeafUUID string `json:"leafUuid"`

	// type=system
	Subtype string `json:"subtype"`

	// Raw nested message (for user and assistant types)
	RawMessage json.RawMessage `json:"message"`

	// Parsed lazily
	parsedAssistant *assistantInner
	parsedUser      *userInner
}

type assistantInner struct {
	ID         string         `json:"id"`
	Role       string         `json:"role"`
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Content    []ContentBlock `json:"content"`
	Usage      *TokenUsage    `json:"usage"`
}

type userInner struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ParseMessage parses a single JSONL line into a Message.
func ParseMessage(line []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}
	return &msg, nil
}

func (m *Message) ensureAssistant() {
	if m.parsedAssistant != nil || m.Type != "assistant" || len(m.RawMessage) == 0 {
		return
	}
	var inner assistantInner
	if err := json.Unmarshal(m.RawMessage, &inner); err == nil {
		m.parsedAssistant = &inner
	}
}

func (m *Message) ensureUser() {
	if m.parsedUser != nil || m.Type != "user" || len(m.RawMessage) == 0 {
		return
	}
	var inner userInner
	if err := json.Unmarshal(m.RawMessage, &inner); err == nil {
		m.parsedUser = &inner
	}
}

// Model returns the model name from an assistant message. Empty for other types.
func (m *Message) Model() string {
	m.ensureAssistant()
	if m.parsedAssistant != nil {
		return m.parsedAssistant.Model
	}
	return ""
}

// Usage returns token usage from an assistant message. Nil for other types.
func (m *Message) Usage() *TokenUsage {
	m.ensureAssistant()
	if m.parsedAssistant != nil {
		return m.parsedAssistant.Usage
	}
	return nil
}

// assistantContent returns all content blocks from an assistant message.
func (m *Message) assistantContent() []ContentBlock {
	m.ensureAssistant()
	if m.parsedAssistant == nil {
		return nil
	}
	return m.parsedAssistant.Content
}

// ToolUseBlocks returns all tool_use content blocks from an assistant message.
func (m *Message) ToolUseBlocks() []ContentBlock {
	m.ensureAssistant()
	if m.parsedAssistant == nil {
		return nil
	}
	var tools []ContentBlock
	for _, block := range m.parsedAssistant.Content {
		if block.Type == "tool_use" {
			tools = append(tools, block)
		}
	}
	return tools
}

// UserContent returns the text content of a user message.
// Handles both string content and array content (returns empty for arrays).
func (m *Message) UserContent() string {
	m.ensureUser()
	if m.parsedUser == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(m.parsedUser.Content, &s); err == nil {
		return s
	}
	return ""
}

# Chat History Tab Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a "Chat" tab to SessionDetailView that shows the full conversation with user messages, assistant text, and one-liner tool call summaries.

**Architecture:** Add a `ChatMessage` model to hold full message content. Extend `ExtractSessionDetail` to collect chat messages from parsed JSONL (user text, assistant text blocks, tool_use one-liners). Add a "Chat" tab (index 0, shifting existing tabs right) to `SessionDetailView` that renders the conversation with word-wrapping.

**Tech Stack:** Go, Bubbletea, existing parser/model/views

---

### Task 1: Add ChatMessage model

**Files:**
- Create: `internal/model/chat.go`
- Create: `internal/model/chat_test.go`

**Step 1: Write the failing test**

Create `internal/model/chat_test.go`:

```go
package model

import (
	"testing"
	"time"
)

func TestChatMessage_Defaults(t *testing.T) {
	msg := ChatMessage{
		Role:      "user",
		Content:   "Fix the login bug",
		Timestamp: time.Now(),
	}
	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}
	if msg.Content != "Fix the login bug" {
		t.Error("expected content to match")
	}
}

func TestChatMessage_ToolCall(t *testing.T) {
	msg := ChatMessage{
		Role:      "tool",
		Content:   "Read → /work/login.go",
		Timestamp: time.Now(),
		ToolName:  "Read",
	}
	if msg.Role != "tool" {
		t.Errorf("expected role 'tool', got %q", msg.Role)
	}
	if msg.ToolName != "Read" {
		t.Errorf("expected tool name 'Read', got %q", msg.ToolName)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `ChatMessage` undefined

**Step 3: Write minimal implementation**

Create `internal/model/chat.go`:

```go
package model

import "time"

// ChatMessage represents a single message in the session chat history.
type ChatMessage struct {
	Role      string    // "user", "assistant", "tool"
	Content   string    // full text for user/assistant, one-liner summary for tool
	Timestamp time.Time
	ToolName  string    // only set for role="tool"
}
```

**Step 4: Run test to verify it passes**

Run: `make test`
Expected: PASS

**Step 5: Add ChatMessages to SessionDetail**

Modify `internal/model/session.go` — add `ChatMessages` field to `SessionDetail`:

```go
type SessionDetail struct {
	Meta         *SessionMeta
	Timeline     []TimelineEvent
	FileActivity []FileOp
	Subagents    []SubagentMeta
	ChatMessages []ChatMessage
}
```

**Step 6: Run tests, commit**

Run: `make test`

```bash
git add internal/model/chat.go internal/model/chat_test.go internal/model/session.go
git commit -m "feat: add ChatMessage model and ChatMessages to SessionDetail"
```

---

### Task 2: Extract chat messages in parser

Extend `ExtractSessionDetail` to populate `ChatMessages` from parsed JSONL messages.

**Files:**
- Modify: `internal/parser/message.go` (add `AssistantText()` method)
- Modify: `internal/parser/detail.go` (collect chat messages)
- Modify: `internal/parser/extract_test.go` or create `internal/parser/detail_test.go`

**Step 1: Write the failing test**

Add to `internal/parser/detail_test.go` (create if not exists):

```go
package parser

import (
	"testing"
	"time"

	"github.com/r/cicada/internal/model"
)

func TestExtractSessionDetail_ChatMessages(t *testing.T) {
	now := time.Now()
	meta := &model.SessionMeta{UUID: "test-uuid"}

	// Build messages simulating a conversation
	messages := []*Message{
		{
			Type:      "user",
			Timestamp: now,
			RawMessage: mustJSON(`{"role":"user","content":"Fix the login bug"}`),
		},
		{
			Type:      "assistant",
			Timestamp: now.Add(time.Minute),
			RawMessage: mustJSON(`{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"I'll look at the login code."},{"type":"tool_use","name":"Read","id":"t1","input":{"file_path":"/work/login.go"}}]}`),
		},
		{
			Type:      "user",
			Timestamp: now.Add(2 * time.Minute),
			RawMessage: mustJSON(`{"role":"user","content":"Now fix it"}`),
		},
		{
			Type:      "assistant",
			Timestamp: now.Add(3 * time.Minute),
			RawMessage: mustJSON(`{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","name":"Edit","id":"t2","input":{"file_path":"/work/login.go"}},{"type":"text","text":"I've fixed the bug."}]}`),
		},
	}

	detail := ExtractSessionDetail(messages, meta)

	if len(detail.ChatMessages) == 0 {
		t.Fatal("expected chat messages, got none")
	}

	// Expected order: user, assistant text, tool one-liner, user, tool one-liner, assistant text
	// At minimum we expect user messages and assistant text
	var userCount, assistantCount, toolCount int
	for _, msg := range detail.ChatMessages {
		switch msg.Role {
		case "user":
			userCount++
		case "assistant":
			assistantCount++
		case "tool":
			toolCount++
		}
	}

	if userCount != 2 {
		t.Errorf("expected 2 user messages, got %d", userCount)
	}
	if assistantCount < 2 {
		t.Errorf("expected at least 2 assistant messages, got %d", assistantCount)
	}
	if toolCount < 2 {
		t.Errorf("expected at least 2 tool messages, got %d", toolCount)
	}

	// First message should be user
	if detail.ChatMessages[0].Role != "user" {
		t.Errorf("expected first chat message role 'user', got %q", detail.ChatMessages[0].Role)
	}
	if detail.ChatMessages[0].Content != "Fix the login bug" {
		t.Errorf("expected first message content 'Fix the login bug', got %q", detail.ChatMessages[0].Content)
	}
}

func TestExtractSessionDetail_ChatMessages_Empty(t *testing.T) {
	meta := &model.SessionMeta{UUID: "empty"}
	detail := ExtractSessionDetail(nil, meta)
	if len(detail.ChatMessages) != 0 {
		t.Errorf("expected 0 chat messages for nil input, got %d", len(detail.ChatMessages))
	}
}
```

Also add a helper at the bottom of the test file:

```go
func mustJSON(s string) json.RawMessage {
	return json.RawMessage(s)
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — ChatMessages is empty (not populated)

**Step 3: Add AssistantText() to Message**

In `internal/parser/message.go`, add:

```go
// AssistantText returns the concatenated text content blocks from an assistant message.
func (m *Message) AssistantText() string {
	m.ensureAssistant()
	if m.parsedAssistant == nil {
		return ""
	}
	var parts []string
	for _, block := range m.parsedAssistant.Content {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}
```

Add `"strings"` to the import if not already there.

**Step 4: Update ExtractSessionDetail to collect chat messages**

In `internal/parser/detail.go`, modify the function to also build `ChatMessages`. For each message:

- `type=user`: add a ChatMessage with role="user", full content
- `type=assistant`: iterate content blocks in order:
  - `text` blocks: add ChatMessage with role="assistant", full text
  - `tool_use` blocks: add ChatMessage with role="tool", one-liner summary like `"Read → /work/login.go"` or `"Bash → make test"` or just `"Edit → /work/file.go"`

Updated `detail.go`:

```go
package parser

import (
	"fmt"

	"github.com/r/cicada/internal/model"
)

func ExtractSessionDetail(messages []*Message, meta *model.SessionMeta) *model.SessionDetail {
	detail := &model.SessionDetail{Meta: meta}

	for _, msg := range messages {
		if msg.IsSidechain {
			continue
		}

		switch msg.Type {
		case "user":
			content := msg.UserContent()
			detail.Timeline = append(detail.Timeline, model.TimelineEvent{
				Timestamp: msg.Timestamp,
				Type:      "user",
				Content:   truncate(content, 200),
			})
			if content != "" {
				detail.ChatMessages = append(detail.ChatMessages, model.ChatMessage{
					Role:      "user",
					Content:   content,
					Timestamp: msg.Timestamp,
				})
			}

		case "assistant":
			// Process content blocks in order for chat view
			if inner := msg.assistantContent(); inner != nil {
				for _, block := range inner {
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
						// Timeline event
						detail.Timeline = append(detail.Timeline, model.TimelineEvent{
							Timestamp: msg.Timestamp,
							Type:      "tool_use",
							ToolName:  block.Name,
							Content:   truncate(block.FileToolPath(), 200),
						})
						// Chat one-liner
						summary := toolSummary(block)
						detail.ChatMessages = append(detail.ChatMessages, model.ChatMessage{
							Role:      "tool",
							Content:   summary,
							Timestamp: msg.Timestamp,
							ToolName:  block.Name,
						})
						// File activity tracking
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
					}
				}
			}
		}
	}

	return detail
}

// toolSummary returns a one-line summary of a tool_use block.
func toolSummary(block ContentBlock) string {
	path := block.FileToolPath()
	if path != "" {
		return fmt.Sprintf("%s → %s", block.Name, path)
	}
	return block.Name
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
```

**Step 5: Add assistantContent() helper to Message**

In `internal/parser/message.go`, add:

```go
// assistantContent returns the content blocks from an assistant message.
func (m *Message) assistantContent() []ContentBlock {
	m.ensureAssistant()
	if m.parsedAssistant == nil {
		return nil
	}
	return m.parsedAssistant.Content
}
```

**Step 6: Run tests, commit**

Run: `make test`
Expected: PASS

```bash
git add internal/parser/detail.go internal/parser/message.go internal/parser/detail_test.go
git commit -m "feat: extract chat messages from session JSONL"
```

---

### Task 3: Add Chat tab to SessionDetailView

Add a "Chat" tab as the first tab in the session detail view, rendering the conversation.

**Files:**
- Modify: `internal/tui/views/session_detail.go`
- Modify: `internal/tui/views/session_detail_test.go`

**Step 1: Write the failing test**

Add to `internal/tui/views/session_detail_test.go`:

First update `newTestDetail()` to include ChatMessages:

```go
// In the detail construction, add:
detail := &model.SessionDetail{
	Meta: meta,
	Timeline: []model.TimelineEvent{...},  // keep existing
	FileActivity: []model.FileOp{...},      // keep existing
	Subagents: []model.SubagentMeta{...},   // keep existing
	ChatMessages: []model.ChatMessage{
		{Role: "user", Content: "Fix the login bug please", Timestamp: now.Add(-time.Hour)},
		{Role: "assistant", Content: "I'll look at the login code and fix it.", Timestamp: now.Add(-55 * time.Minute)},
		{Role: "tool", Content: "Read → /work/login.go", ToolName: "Read", Timestamp: now.Add(-55 * time.Minute)},
		{Role: "tool", Content: "Edit → /work/login.go", ToolName: "Edit", Timestamp: now.Add(-50 * time.Minute)},
		{Role: "assistant", Content: "I've fixed the login bug.", Timestamp: now.Add(-50 * time.Minute)},
		{Role: "user", Content: "Run the tests", Timestamp: now.Add(-45 * time.Minute)},
		{Role: "tool", Content: "Bash", ToolName: "Bash", Timestamp: now.Add(-40 * time.Minute)},
	},
}
```

Then add:

```go
func TestSessionDetailView_ChatTab(t *testing.T) {
	s := store.New()
	meta, detail := newTestDetail()
	view := NewSessionDetailView(s, meta, detail)
	// Chat is tab 0 (default)
	content := view.View(100, 30)

	if !strings.Contains(content, "Chat") {
		t.Error("expected 'Chat' tab label")
	}
	if !strings.Contains(content, "Fix the login bug") {
		t.Error("expected user message in chat")
	}
	if !strings.Contains(content, "Read →") {
		t.Error("expected tool summary in chat")
	}
}

func TestSessionDetailView_ChatTab_Empty(t *testing.T) {
	s := store.New()
	meta := &model.SessionMeta{
		UUID: "empty", Slug: "empty-session",
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{},
	}
	detail := &model.SessionDetail{Meta: meta}
	view := NewSessionDetailView(s, meta, detail)
	content := view.View(100, 30)

	if !strings.Contains(content, "No chat messages") {
		t.Error("expected empty state message")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — no "Chat" tab, content doesn't match

**Step 3: Implement the Chat tab**

In `internal/tui/views/session_detail.go`:

1. Update tab names:

```go
var detailTabNames = []string{"Chat", "Overview", "Timeline", "Files", "Agents", "Tools"}
```

2. Update the View switch to shift indices:

```go
switch v.activeTab {
case 0:
    content = v.renderChat(width)
case 1:
    content = v.renderOverview(width)
case 2:
    content = v.renderTimeline(width, height-6)
case 3:
    content = v.renderFiles(width, height-6)
case 4:
    content = v.renderAgents(width)
case 5:
    content = v.renderTools(width)
}
```

3. Add `renderChat` method:

```go
func (v *SessionDetailView) renderChat(width int) string {
	if v.detail == nil || len(v.detail.ChatMessages) == 0 {
		return "  No chat messages."
	}

	contentWidth := width - 6
	if contentWidth < 20 {
		contentWidth = 20
	}

	var b strings.Builder
	for _, msg := range v.detail.ChatMessages {
		switch msg.Role {
		case "user":
			b.WriteString("\n  ▶ You:\n")
			for _, line := range wrapText(msg.Content, contentWidth) {
				b.WriteString("    " + line + "\n")
			}
		case "assistant":
			b.WriteString("\n  ◀ Assistant:\n")
			for _, line := range wrapText(msg.Content, contentWidth) {
				b.WriteString("    " + line + "\n")
			}
		case "tool":
			b.WriteString("    ⚙ " + msg.Content + "\n")
		}
	}

	return b.String()
}

// wrapText wraps a string to the given width, breaking on word boundaries.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := words[0]
		for _, word := range words[1:] {
			if len(current)+1+len(word) > width {
				lines = append(lines, current)
				current = word
			} else {
				current += " " + word
			}
		}
		lines = append(lines, current)
	}
	return lines
}
```

**Step 4: Fix existing tests**

The tab index shift means existing tests that navigate to specific tabs need updating. The `TestSessionDetailView_OverviewTab` test expected Overview at tab 0 — it's now at tab 1. But since it tests the default view (tab 0), it will now see the Chat tab instead. Update:

- `TestSessionDetailView_OverviewTab`: The default tab is now Chat. Rename or adjust to test Chat at tab 0, and add a navigation step for Overview.
- `TestSessionDetailView_TimelineTab`: was 1 right press, now needs 2
- `TestSessionDetailView_FilesTab`: was 2 right presses, now needs 3
- `TestSessionDetailView_AgentsTab`: was 3 l presses, now needs 4
- `TestSessionDetailView_ToolsTab`: was 4 right presses, now needs 5
- `TestSessionDetailView_TabWrapping`: last tab was 4, now 5

Update each test's navigation count accordingly.

**Step 5: Run tests, commit**

Run: `make test`
Expected: PASS

```bash
git add internal/tui/views/session_detail.go internal/tui/views/session_detail_test.go
git commit -m "feat: add Chat tab to session detail view"
```

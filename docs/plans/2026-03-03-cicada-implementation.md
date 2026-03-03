# Cicada Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a standalone Go TUI app that reads Claude Code session data from `~/.claude/` and presents analytics, session browsing, and tool usage interactively.

**Architecture:** Background scanner goroutine discovers and parses JSONL session files into an in-memory store (sync.RWMutex-protected). Bubbletea TUI subscribes to batch updates via tea.Msg for progressive rendering. Session detail data loaded lazily on navigation.

**Tech Stack:** Go, Bubbletea, Lipgloss, Bubbles (table, viewport, textinput)

---

### Task 1: Project scaffold — Go module, Makefile, main.go

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `main.go`

**Step 1: Create go.mod**

```bash
cd /Users/r/work/cicada && go mod init github.com/r/cicada
```

**Step 2: Create Makefile**

Create `Makefile`:

```makefile
.PHONY: build test lint fmt run clean

BINARY := cicada

build:
	go build -o $(BINARY) .

test:
	go test ./... -v

lint:
	go vet ./...

fmt:
	gofmt -w .

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
```

**Step 3: Create minimal main.go**

Create `main.go`:

```go
package main

import "fmt"

func main() {
	fmt.Println("cicada")
}
```

**Step 4: Verify build works**

Run: `make build && ./cicada`
Expected: prints "cicada"

**Step 5: Verify test works**

Run: `make test`
Expected: passes (no tests yet, that's ok)

**Step 6: Commit**

```bash
git add go.mod Makefile main.go
git commit -m "scaffold: init Go module with Makefile and main entry point"
```

---

### Task 2: Domain model types

**Files:**
- Create: `internal/model/session.go`
- Create: `internal/model/timeline.go`
- Create: `internal/model/subagent.go`
- Create: `internal/model/analytics.go`
- Test: `internal/model/session_test.go`

**Step 1: Write test for SessionMeta**

Create `internal/model/session_test.go`:

```go
package model

import (
	"testing"
	"time"
)

func TestSessionMeta_Duration(t *testing.T) {
	now := time.Now()
	meta := SessionMeta{
		UUID:      "test-uuid",
		Slug:      "test-slug",
		StartTime: now.Add(-10 * time.Minute),
		EndTime:   now,
	}

	if meta.UUID != "test-uuid" {
		t.Errorf("expected UUID 'test-uuid', got %q", meta.UUID)
	}
	if meta.Slug != "test-slug" {
		t.Errorf("expected Slug 'test-slug', got %q", meta.Slug)
	}

	expectedDuration := 10 * time.Minute
	if meta.EndTime.Sub(meta.StartTime) != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, meta.EndTime.Sub(meta.StartTime))
	}
}

func TestSessionMeta_Defaults(t *testing.T) {
	meta := SessionMeta{}

	if meta.ToolUsage != nil {
		t.Error("expected nil ToolUsage for zero value")
	}
	if meta.MessageCount != 0 {
		t.Errorf("expected 0 MessageCount, got %d", meta.MessageCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `SessionMeta` undefined

**Step 3: Write model types**

Create `internal/model/session.go`:

```go
package model

import "time"

// SessionMeta holds lightweight metadata parsed during background scan.
type SessionMeta struct {
	UUID          string
	Slug          string
	ProjectPath   string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	InitialPrompt string
	SessionTitles []string
	Models        map[string]int // model name → message count
	TokensIn      int64
	TokensOut     int64
	CacheRead     int64
	CacheWrite    int64
	ToolUsage     map[string]int // tool name → count
	SkillsUsed    map[string]int
	CommandsUsed  map[string]int
	GitBranches   []string
	SubagentCount int
	FileOps       map[string]int // operation type → count
	MessageCount  int
}

// SessionDetail holds full session data, loaded lazily on navigation.
type SessionDetail struct {
	Meta         *SessionMeta
	Timeline     []TimelineEvent
	FileActivity []FileOp
	Subagents    []SubagentMeta
}
```

Create `internal/model/timeline.go`:

```go
package model

import "time"

// TimelineEvent represents a single event in a session timeline.
type TimelineEvent struct {
	Timestamp time.Time
	Type      string // "user", "assistant", "tool_use", "tool_result"
	Content   string // truncated for display
	ToolName  string
	ActorID   string // session or subagent agent_id
}

// FileOp represents a file operation performed during a session.
type FileOp struct {
	Timestamp time.Time
	Path      string
	Operation string // "read", "write", "edit", "delete", "search"
	Actor     string // session UUID or subagent agent_id
	ActorType string // "session" or "subagent"
	ToolName  string
}
```

Create `internal/model/subagent.go`:

```go
package model

import "time"

// SubagentMeta holds metadata about a subagent invocation.
type SubagentMeta struct {
	AgentID       string
	Type          string // "Explore", "Plan", "Bash", "general-purpose", etc.
	InitialPrompt string
	ToolUsage     map[string]int
	TokensIn      int64
	TokensOut     int64
	Duration      time.Duration
}
```

Create `internal/model/analytics.go`:

```go
package model

// Analytics holds aggregated analytics computed from all sessions.
type Analytics struct {
	TotalSessions    int
	TotalTokensIn    int64
	TotalTokensOut   int64
	TotalCacheRead   int64
	TotalCacheWrite  int64
	ActiveProjects   int
	ModelsUsed       map[string]int // model → total message count
	ToolsUsed        map[string]int // tool → total call count
	SessionsByDate   map[string]int // "2026-03-03" → count
	WorkModeExplore  int            // Read, Grep, Glob, WebFetch, WebSearch calls
	WorkModeBuild    int            // Write, Edit calls
	WorkModeTest     int            // Bash, Agent, Task calls
}
```

**Step 4: Run tests**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/model/
git commit -m "feat: add domain model types for sessions, timeline, subagents, analytics"
```

---

### Task 3: JSONL line parser — parse raw JSON lines into typed messages

**Files:**
- Create: `internal/parser/message.go`
- Create: `internal/parser/content.go`
- Test: `internal/parser/message_test.go`

**Step 1: Write tests for message parsing**

Create `internal/parser/message_test.go`:

```go
package parser

import (
	"testing"
)

func TestParseMessage_UserMessage(t *testing.T) {
	line := []byte(`{
		"type": "user",
		"uuid": "abc-123",
		"timestamp": "2026-03-03T10:00:00.000Z",
		"sessionId": "sess-1",
		"slug": "cool-slug",
		"cwd": "/Users/r/work",
		"gitBranch": "main",
		"isSidechain": false,
		"message": {
			"role": "user",
			"content": "Hello world"
		}
	}`)

	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "user" {
		t.Errorf("expected type 'user', got %q", msg.Type)
	}
	if msg.UUID != "abc-123" {
		t.Errorf("expected UUID 'abc-123', got %q", msg.UUID)
	}
	if msg.Slug != "cool-slug" {
		t.Errorf("expected slug 'cool-slug', got %q", msg.Slug)
	}
	if msg.GitBranch != "main" {
		t.Errorf("expected gitBranch 'main', got %q", msg.GitBranch)
	}
	if msg.UserContent() != "Hello world" {
		t.Errorf("expected user content 'Hello world', got %q", msg.UserContent())
	}
}

func TestParseMessage_AssistantMessage(t *testing.T) {
	line := []byte(`{
		"type": "assistant",
		"uuid": "def-456",
		"timestamp": "2026-03-03T10:01:00.000Z",
		"sessionId": "sess-1",
		"slug": "cool-slug",
		"gitBranch": "main",
		"isSidechain": false,
		"message": {
			"id": "msg_xxx",
			"role": "assistant",
			"model": "claude-opus-4-6",
			"stop_reason": "end_turn",
			"content": [
				{"type": "text", "text": "Hi there!"},
				{"type": "tool_use", "id": "toolu_01", "name": "Read", "input": {"file_path": "/tmp/foo.txt"}}
			],
			"usage": {
				"input_tokens": 100,
				"output_tokens": 50,
				"cache_creation_input_tokens": 200,
				"cache_read_input_tokens": 5000
			}
		}
	}`)

	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "assistant" {
		t.Errorf("expected type 'assistant', got %q", msg.Type)
	}
	if msg.Model() != "claude-opus-4-6" {
		t.Errorf("expected model 'claude-opus-4-6', got %q", msg.Model())
	}

	usage := msg.Usage()
	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", usage.OutputTokens)
	}
	if usage.CacheCreationInputTokens != 200 {
		t.Errorf("expected 200 cache creation tokens, got %d", usage.CacheCreationInputTokens)
	}
	if usage.CacheReadInputTokens != 5000 {
		t.Errorf("expected 5000 cache read tokens, got %d", usage.CacheReadInputTokens)
	}

	tools := msg.ToolUseBlocks()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool_use block, got %d", len(tools))
	}
	if tools[0].Name != "Read" {
		t.Errorf("expected tool name 'Read', got %q", tools[0].Name)
	}
}

func TestParseMessage_SummaryMessage(t *testing.T) {
	line := []byte(`{
		"type": "summary",
		"summary": "Fix the login bug",
		"leafUuid": "leaf-1"
	}`)

	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "summary" {
		t.Errorf("expected type 'summary', got %q", msg.Type)
	}
	if msg.Summary != "Fix the login bug" {
		t.Errorf("expected summary 'Fix the login bug', got %q", msg.Summary)
	}
}

func TestParseMessage_SkillDetection(t *testing.T) {
	line := []byte(`{
		"type": "assistant",
		"uuid": "skill-1",
		"timestamp": "2026-03-03T10:01:00.000Z",
		"sessionId": "sess-1",
		"slug": "cool-slug",
		"isSidechain": false,
		"message": {
			"id": "msg_xxx",
			"role": "assistant",
			"model": "claude-opus-4-6",
			"content": [
				{"type": "tool_use", "id": "toolu_02", "name": "Skill", "input": {"skill": "superpowers:brainstorming"}}
			],
			"usage": {"input_tokens": 10, "output_tokens": 5, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}
	}`)

	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tools := msg.ToolUseBlocks()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool block, got %d", len(tools))
	}
	if tools[0].Name != "Skill" {
		t.Errorf("expected 'Skill', got %q", tools[0].Name)
	}
	skill := tools[0].SkillName()
	if skill != "superpowers:brainstorming" {
		t.Errorf("expected 'superpowers:brainstorming', got %q", skill)
	}
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	line := []byte(`not valid json`)
	_, err := ParseMessage(line)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseMessage_UnknownType(t *testing.T) {
	line := []byte(`{"type": "progress", "uuid": "x"}`)
	msg, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "progress" {
		t.Errorf("expected type 'progress', got %q", msg.Type)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `make test`
Expected: FAIL — `ParseMessage` undefined

**Step 3: Write content block types**

Create `internal/parser/content.go`:

```go
package parser

import "encoding/json"

// ContentBlock represents a single block in an assistant message's content array.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
}

// SkillName extracts the skill name from a Skill tool_use block's input.
// Returns empty string if not a Skill block or input is missing.
func (cb *ContentBlock) SkillName() string {
	if cb.Name != "Skill" || len(cb.Input) == 0 {
		return ""
	}
	var inp struct {
		Skill string `json:"skill"`
	}
	if err := json.Unmarshal(cb.Input, &inp); err != nil {
		return ""
	}
	return inp.Skill
}

// FileToolPath extracts the file path from a file-related tool_use block's input.
// Works for Read, Write, Edit, Glob, Grep tools.
func (cb *ContentBlock) FileToolPath() string {
	if len(cb.Input) == 0 {
		return ""
	}
	var inp map[string]json.RawMessage
	if err := json.Unmarshal(cb.Input, &inp); err != nil {
		return ""
	}
	// Try common path field names
	for _, key := range []string{"file_path", "path", "glob_pattern", "target_directory"} {
		if raw, ok := inp[key]; ok {
			var s string
			if err := json.Unmarshal(raw, &s); err == nil {
				return s
			}
		}
	}
	return ""
}

// TokenUsage holds token counts from an assistant message's usage field.
type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}
```

**Step 4: Write message parser**

Create `internal/parser/message.go`:

```go
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
```

**Step 5: Run tests**

Run: `make test`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/parser/
git commit -m "feat: add JSONL message parser with content block and token usage types"
```

---

### Task 4: JSONL file reader — read a file line by line, yield parsed messages

**Files:**
- Create: `internal/parser/jsonl.go`
- Test: `internal/parser/jsonl_test.go`

**Step 1: Write test using a temp file**

Create `internal/parser/jsonl_test.go`:

```go
package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSessionFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := `{"type":"summary","summary":"Test session","leafUuid":"leaf-1"}
{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"s1","slug":"test","isSidechain":false,"message":{"role":"user","content":"Hello"}}
{"type":"assistant","uuid":"a1","timestamp":"2026-03-03T10:01:00.000Z","sessionId":"s1","slug":"test","isSidechain":false,"message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
	if messages[0].Type != "summary" {
		t.Errorf("expected first message type 'summary', got %q", messages[0].Type)
	}
	if messages[1].Type != "user" {
		t.Errorf("expected second message type 'user', got %q", messages[1].Type)
	}
	if messages[2].Type != "assistant" {
		t.Errorf("expected third message type 'assistant', got %q", messages[2].Type)
	}
}

func TestReadSessionFile_SkipsBadLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")

	content := `not json at all
{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"s1","slug":"test","isSidechain":false,"message":{"role":"user","content":"Hello"}}
also not json
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 valid message, got %d", len(messages))
	}
}

func TestReadSessionFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}

func TestReadSessionFile_NotFound(t *testing.T) {
	_, err := ReadSessionFile("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `ReadSessionFile` undefined

**Step 3: Implement JSONL file reader**

Create `internal/parser/jsonl.go`:

```go
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
```

**Step 4: Run tests**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/parser/jsonl.go internal/parser/jsonl_test.go
git commit -m "feat: add JSONL file reader with error-tolerant line parsing"
```

---

### Task 5: Session metadata extractor — convert parsed messages into SessionMeta

**Files:**
- Create: `internal/parser/extract.go`
- Test: `internal/parser/extract_test.go`

**Step 1: Write test**

Create `internal/parser/extract_test.go`:

```go
package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSessionMeta(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := `{"type":"summary","summary":"Fix the login bug"}
{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"s1","slug":"happy-cat","cwd":"/work","gitBranch":"main","isSidechain":false,"message":{"role":"user","content":"Fix the login bug please"}}
{"type":"assistant","uuid":"a1","timestamp":"2026-03-03T10:01:00.000Z","sessionId":"s1","slug":"happy-cat","gitBranch":"main","isSidechain":false,"message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_01","name":"Read","input":{"file_path":"/work/login.go"}},{"type":"tool_use","id":"toolu_02","name":"Edit","input":{"file_path":"/work/login.go","old_string":"foo","new_string":"bar"}}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":200,"cache_read_input_tokens":5000}}}
{"type":"assistant","uuid":"a2","timestamp":"2026-03-03T10:02:00.000Z","sessionId":"s1","slug":"happy-cat","gitBranch":"feature","isSidechain":false,"message":{"id":"msg_2","role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"tool_use","id":"toolu_03","name":"Bash","input":{"command":"make test"}}],"usage":{"input_tokens":50,"output_tokens":25,"cache_creation_input_tokens":0,"cache_read_input_tokens":1000}}}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}

	meta := ExtractSessionMeta(messages, "/projects/-work", "s1.jsonl")
	if meta.Slug != "happy-cat" {
		t.Errorf("expected slug 'happy-cat', got %q", meta.Slug)
	}
	if meta.InitialPrompt != "Fix the login bug please" {
		t.Errorf("expected initial prompt, got %q", meta.InitialPrompt)
	}
	if len(meta.SessionTitles) != 1 || meta.SessionTitles[0] != "Fix the login bug" {
		t.Errorf("expected session title 'Fix the login bug', got %v", meta.SessionTitles)
	}
	if meta.TokensIn != 150 {
		t.Errorf("expected 150 input tokens, got %d", meta.TokensIn)
	}
	if meta.TokensOut != 75 {
		t.Errorf("expected 75 output tokens, got %d", meta.TokensOut)
	}
	if meta.CacheRead != 6000 {
		t.Errorf("expected 6000 cache read tokens, got %d", meta.CacheRead)
	}
	if meta.CacheWrite != 200 {
		t.Errorf("expected 200 cache write tokens, got %d", meta.CacheWrite)
	}
	if meta.ToolUsage["Read"] != 1 {
		t.Errorf("expected Read tool count 1, got %d", meta.ToolUsage["Read"])
	}
	if meta.ToolUsage["Edit"] != 1 {
		t.Errorf("expected Edit tool count 1, got %d", meta.ToolUsage["Edit"])
	}
	if meta.ToolUsage["Bash"] != 1 {
		t.Errorf("expected Bash tool count 1, got %d", meta.ToolUsage["Bash"])
	}
	if meta.Models["claude-opus-4-6"] != 1 {
		t.Errorf("expected 1 opus message, got %d", meta.Models["claude-opus-4-6"])
	}
	if meta.Models["claude-sonnet-4-6"] != 1 {
		t.Errorf("expected 1 sonnet message, got %d", meta.Models["claude-sonnet-4-6"])
	}
	if meta.MessageCount != 3 {
		t.Errorf("expected 3 messages (1 user + 2 assistant), got %d", meta.MessageCount)
	}

	// Git branches
	branchSet := make(map[string]bool)
	for _, b := range meta.GitBranches {
		branchSet[b] = true
	}
	if !branchSet["main"] || !branchSet["feature"] {
		t.Errorf("expected branches [main, feature], got %v", meta.GitBranches)
	}

	// File ops
	if meta.FileOps["read"] != 1 {
		t.Errorf("expected 1 read file op, got %d", meta.FileOps["read"])
	}
	if meta.FileOps["edit"] != 1 {
		t.Errorf("expected 1 edit file op, got %d", meta.FileOps["edit"])
	}
}

func TestExtractSessionMeta_EmptyMessages(t *testing.T) {
	meta := ExtractSessionMeta(nil, "/projects/-work", "empty.jsonl")
	if meta.MessageCount != 0 {
		t.Errorf("expected 0 messages, got %d", meta.MessageCount)
	}
	if meta.Slug != "" {
		t.Errorf("expected empty slug, got %q", meta.Slug)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `ExtractSessionMeta` undefined

**Step 3: Implement extractor**

Create `internal/parser/extract.go`:

```go
package parser

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/r/cicada/internal/model"
)

// File tool name to operation type mapping.
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
```

**Step 4: Run tests**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/parser/extract.go internal/parser/extract_test.go
git commit -m "feat: add session metadata extractor from parsed JSONL messages"
```

---

### Task 6: In-memory store with query methods

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

**Step 1: Write tests**

Create `internal/store/store_test.go`:

```go
package store

import (
	"testing"
	"time"

	"github.com/r/cicada/internal/model"
)

func newTestMeta(uuid, slug, project string, start time.Time, tokensIn int64) *model.SessionMeta {
	return &model.SessionMeta{
		UUID:        uuid,
		Slug:        slug,
		ProjectPath: project,
		StartTime:   start,
		EndTime:     start.Add(10 * time.Minute),
		Duration:    10 * time.Minute,
		TokensIn:    tokensIn,
		TokensOut:   tokensIn / 2,
		Models:      map[string]int{"claude-opus-4-6": 1},
		ToolUsage:   map[string]int{"Read": 2, "Edit": 1},
		FileOps:     map[string]int{"read": 2, "edit": 1},
		SkillsUsed:  map[string]int{},
		CommandsUsed: map[string]int{},
		MessageCount: 5,
	}
}

func TestStore_AddAndGet(t *testing.T) {
	s := New()
	now := time.Now()
	meta := newTestMeta("uuid-1", "cool-slug", "/work/project", now, 1000)

	s.Add(meta)

	got := s.Get("uuid-1")
	if got == nil {
		t.Fatal("expected to find session")
	}
	if got.Slug != "cool-slug" {
		t.Errorf("expected slug 'cool-slug', got %q", got.Slug)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	s := New()
	if s.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent session")
	}
}

func TestStore_AllSessions(t *testing.T) {
	s := New()
	now := time.Now()
	s.Add(newTestMeta("u1", "s1", "/p1", now, 100))
	s.Add(newTestMeta("u2", "s2", "/p2", now.Add(-time.Hour), 200))

	all := s.AllSessions()
	if len(all) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(all))
	}
}

func TestStore_Projects(t *testing.T) {
	s := New()
	now := time.Now()
	s.Add(newTestMeta("u1", "s1", "/p1", now, 100))
	s.Add(newTestMeta("u2", "s2", "/p1", now, 200))
	s.Add(newTestMeta("u3", "s3", "/p2", now, 300))

	projects := s.Projects()
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestStore_SessionsByProject(t *testing.T) {
	s := New()
	now := time.Now()
	s.Add(newTestMeta("u1", "s1", "/p1", now, 100))
	s.Add(newTestMeta("u2", "s2", "/p1", now, 200))
	s.Add(newTestMeta("u3", "s3", "/p2", now, 300))

	p1Sessions := s.SessionsByProject("/p1")
	if len(p1Sessions) != 2 {
		t.Fatalf("expected 2 sessions for /p1, got %d", len(p1Sessions))
	}
}

func TestStore_Analytics(t *testing.T) {
	s := New()
	now := time.Now()
	s.Add(newTestMeta("u1", "s1", "/p1", now, 1000))
	s.Add(newTestMeta("u2", "s2", "/p2", now, 2000))

	analytics := s.Analytics()
	if analytics.TotalSessions != 2 {
		t.Errorf("expected 2 total sessions, got %d", analytics.TotalSessions)
	}
	if analytics.TotalTokensIn != 3000 {
		t.Errorf("expected 3000 total tokens in, got %d", analytics.TotalTokensIn)
	}
	if analytics.ActiveProjects != 2 {
		t.Errorf("expected 2 active projects, got %d", analytics.ActiveProjects)
	}
	if analytics.ToolsUsed["Read"] != 4 {
		t.Errorf("expected 4 total Read calls, got %d", analytics.ToolsUsed["Read"])
	}
}

func TestStore_ScanProgress(t *testing.T) {
	s := New()
	s.SetScanProgress(50, 100)

	scanned, total := s.ScanProgress()
	if scanned != 50 || total != 100 {
		t.Errorf("expected 50/100, got %d/%d", scanned, total)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `New` undefined

**Step 3: Implement store**

Create `internal/store/store.go`:

```go
package store

import (
	"sync"

	"github.com/r/cicada/internal/model"
)

// Store holds all session metadata in memory with concurrent-safe access.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*model.SessionMeta // UUID → meta
	byProject map[string][]string          // project path → []UUID

	scanMu      sync.RWMutex
	scanScanned int
	scanTotal   int
}

// New creates a new empty Store.
func New() *Store {
	return &Store{
		sessions:  make(map[string]*model.SessionMeta),
		byProject: make(map[string][]string),
	}
}

// Add inserts a session into the store.
func (s *Store) Add(meta *model.SessionMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[meta.UUID] = meta
	s.byProject[meta.ProjectPath] = append(s.byProject[meta.ProjectPath], meta.UUID)
}

// Get returns a session by UUID, or nil if not found.
func (s *Store) Get(uuid string) *model.SessionMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[uuid]
}

// AllSessions returns all sessions.
func (s *Store) AllSessions() []*model.SessionMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.SessionMeta, 0, len(s.sessions))
	for _, meta := range s.sessions {
		result = append(result, meta)
	}
	return result
}

// Projects returns a list of unique project paths.
func (s *Store) Projects() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]string, 0, len(s.byProject))
	for p := range s.byProject {
		projects = append(projects, p)
	}
	return projects
}

// SessionsByProject returns all sessions for a given project path.
func (s *Store) SessionsByProject(project string) []*model.SessionMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uuids := s.byProject[project]
	result := make([]*model.SessionMeta, 0, len(uuids))
	for _, uuid := range uuids {
		if meta, ok := s.sessions[uuid]; ok {
			result = append(result, meta)
		}
	}
	return result
}

// Analytics computes aggregate analytics from all sessions.
func (s *Store) Analytics() *model.Analytics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a := &model.Analytics{
		ModelsUsed:     make(map[string]int),
		ToolsUsed:      make(map[string]int),
		SessionsByDate: make(map[string]int),
	}

	projectSet := make(map[string]bool)

	for _, meta := range s.sessions {
		a.TotalSessions++
		a.TotalTokensIn += meta.TokensIn
		a.TotalTokensOut += meta.TokensOut
		a.TotalCacheRead += meta.CacheRead
		a.TotalCacheWrite += meta.CacheWrite
		projectSet[meta.ProjectPath] = true

		for m, count := range meta.Models {
			a.ModelsUsed[m] += count
		}
		for tool, count := range meta.ToolUsage {
			a.ToolsUsed[tool] += count

			// Work mode classification
			switch tool {
			case "Read", "Grep", "Glob", "WebFetch", "WebSearch", "LS", "SemanticSearch":
				a.WorkModeExplore += count
			case "Write", "Edit", "StrReplace":
				a.WorkModeBuild += count
			case "Bash", "Agent", "TaskCreate", "TaskUpdate":
				a.WorkModeTest += count
			}
		}

		if !meta.StartTime.IsZero() {
			date := meta.StartTime.Format("2006-01-02")
			a.SessionsByDate[date]++
		}
	}

	a.ActiveProjects = len(projectSet)
	return a
}

// SetScanProgress updates the background scan progress.
func (s *Store) SetScanProgress(scanned, total int) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	s.scanScanned = scanned
	s.scanTotal = total
}

// ScanProgress returns the current scan progress.
func (s *Store) ScanProgress() (scanned, total int) {
	s.scanMu.RLock()
	defer s.scanMu.RUnlock()
	return s.scanScanned, s.scanTotal
}
```

**Step 4: Run tests**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat: add in-memory session store with query methods and analytics"
```

---

### Task 7: Background scanner

**Files:**
- Create: `internal/store/scanner.go`
- Test: `internal/store/scanner_test.go`

**Step 1: Write test**

Create `internal/store/scanner_test.go`:

```go
package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestSession(t *testing.T, dir, uuid string) {
	t.Helper()
	content := `{"type":"user","uuid":"u1","timestamp":"2026-03-03T10:00:00.000Z","sessionId":"` + uuid + `","slug":"test-slug","gitBranch":"main","isSidechain":false,"message":{"role":"user","content":"Hello"}}
{"type":"assistant","uuid":"a1","timestamp":"2026-03-03T10:01:00.000Z","sessionId":"` + uuid + `","slug":"test-slug","gitBranch":"main","isSidechain":false,"message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	path := filepath.Join(dir, uuid+".jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestScanner_DiscoverAndParse(t *testing.T) {
	// Set up a fake ~/.claude/projects structure
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "-Users-r-work-myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeTestSession(t, projectDir, "sess-001")
	writeTestSession(t, projectDir, "sess-002")

	s := New()
	scanner := NewScanner(s, baseDir)

	// Collect messages
	var msgs []ScanMsg
	msgCh := make(chan ScanMsg, 100)
	done := make(chan struct{})
	go func() {
		for msg := range msgCh {
			msgs = append(msgs, msg)
		}
		close(done)
	}()

	scanner.Run(msgCh)
	close(msgCh)
	<-done

	// Verify store has sessions
	all := s.AllSessions()
	if len(all) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(all))
	}

	// Verify we got messages
	if len(msgs) == 0 {
		t.Error("expected at least one scan message")
	}

	// Last message should be ScanComplete
	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Type != ScanComplete {
		t.Errorf("expected last message type ScanComplete, got %v", lastMsg.Type)
	}
}

func TestScanner_EmptyProjectsDir(t *testing.T) {
	baseDir := t.TempDir()

	s := New()
	scanner := NewScanner(s, baseDir)

	msgCh := make(chan ScanMsg, 100)
	done := make(chan struct{})
	go func() {
		for range msgCh {
		}
		close(done)
	}()

	scanner.Run(msgCh)
	close(msgCh)
	<-done

	if len(s.AllSessions()) != 0 {
		t.Error("expected 0 sessions for empty projects dir")
	}
}

func TestScanner_NonexistentDir(t *testing.T) {
	s := New()
	scanner := NewScanner(s, "/nonexistent/path")

	msgCh := make(chan ScanMsg, 100)
	done := make(chan struct{})
	var msgs []ScanMsg
	go func() {
		for msg := range msgCh {
			msgs = append(msgs, msg)
		}
		close(done)
	}()

	scanner.Run(msgCh)
	close(msgCh)
	<-done

	// Should still send ScanComplete even on error
	found := false
	for _, msg := range msgs {
		if msg.Type == ScanComplete {
			found = true
		}
	}
	if !found {
		t.Error("expected ScanComplete message even for nonexistent dir")
	}
}

// Ensure scanner completes in a reasonable time
func TestScanner_Timeout(t *testing.T) {
	baseDir := t.TempDir()
	s := New()
	scanner := NewScanner(s, baseDir)

	msgCh := make(chan ScanMsg, 100)
	done := make(chan struct{})
	go func() {
		for range msgCh {
		}
		close(done)
	}()

	start := time.Now()
	scanner.Run(msgCh)
	close(msgCh)
	<-done

	if time.Since(start) > 5*time.Second {
		t.Error("scanner took too long for empty dir")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `NewScanner`, `ScanMsg` undefined

**Step 3: Implement scanner**

Create `internal/store/scanner.go`:

```go
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
	ProjectsDiscovered ScanMsgType = iota
	SessionsBatch
	ScanComplete
)

// ScanMsg is sent from the scanner to the TUI to report progress.
type ScanMsg struct {
	Type     ScanMsgType
	Projects []string // for ProjectsDiscovered
	Count    int      // sessions in this batch
	Scanned  int      // total scanned so far
	Total    int      // total to scan
}

// Scanner discovers and parses JSONL session files.
type Scanner struct {
	store    *Store
	baseDir  string // path to ~/.claude/projects
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
	// Discover project directories
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

	// Count total JSONL files
	var allFiles []struct {
		project string
		path    string
	}
	for _, projName := range projectDirs {
		projPath := filepath.Join(sc.baseDir, projName)
		files, _ := filepath.Glob(filepath.Join(projPath, "*.jsonl"))
		for _, f := range files {
			allFiles = append(allFiles, struct {
				project string
				path    string
			}{projName, f})
		}
	}

	total := len(allFiles)
	sc.store.SetScanProgress(0, total)

	// Parse sessions in batches
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
```

**Step 4: Run tests**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/scanner.go internal/store/scanner_test.go
git commit -m "feat: add background scanner for discovering and parsing session files"
```

---

### Task 8: Install Bubbletea dependencies

**Step 1: Add dependencies**

```bash
cd /Users/r/work/cicada && go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/lipgloss@latest github.com/charmbracelet/bubbles@latest
```

**Step 2: Tidy**

```bash
go mod tidy
```

**Step 3: Verify build**

Run: `make build`
Expected: builds successfully

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add bubbletea, lipgloss, and bubbles"
```

---

### Task 9: TUI styles and theme

**Files:**
- Create: `internal/tui/styles.go`
- Test: `internal/tui/styles_test.go`

**Step 1: Write test**

Create `internal/tui/styles_test.go`:

```go
package tui

import (
	"testing"
)

func TestThemeColors(t *testing.T) {
	// Verify theme struct is properly initialized
	theme := DefaultTheme()

	if theme.Primary == "" {
		t.Error("expected non-empty Primary color")
	}
	if theme.Secondary == "" {
		t.Error("expected non-empty Secondary color")
	}
	if theme.Muted == "" {
		t.Error("expected non-empty Muted color")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `DefaultTheme` undefined

**Step 3: Implement styles**

Create `internal/tui/styles.go`:

```go
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
		Primary:   "#7C3AED", // purple
		Secondary: "#06B6D4", // cyan
		Accent:    "#F59E0B", // amber
		Muted:     "#6B7280", // gray
		Error:     "#EF4444", // red
		Success:   "#10B981", // green
		Warning:   "#F59E0B", // amber
		BgDark:    "#1F2937", // dark gray
		BgLight:   "#374151", // medium gray
		Fg:        "#F9FAFB", // white
		FgDim:     "#9CA3AF", // light gray
	}
}

// Styles holds pre-built lipgloss styles.
type Styles struct {
	TabBar       lipgloss.Style
	TabActive    lipgloss.Style
	TabInactive  lipgloss.Style
	StatusBar    lipgloss.Style
	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	StatLabel    lipgloss.Style
	StatValue    lipgloss.Style
	Selected     lipgloss.Style
	ViewPort     lipgloss.Style
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
```

**Step 4: Run tests**

Run: `make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI theme and lipgloss styles"
```

---

### Task 10: TUI app shell — root model with tab navigation and status bar

**Files:**
- Create: `internal/tui/app.go`
- Test: `internal/tui/app_test.go`
- Modify: `main.go`

**Step 1: Write test**

Create `internal/tui/app_test.go`:

```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/r/cicada/internal/store"
)

func TestApp_InitialView(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	if app.activeTab != 0 {
		t.Errorf("expected initial tab 0, got %d", app.activeTab)
	}
}

func TestApp_TabNavigation(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	// Press Tab to move to next
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updated.(App)
	if app.activeTab != 1 {
		t.Errorf("expected tab 1 after Tab, got %d", app.activeTab)
	}

	// Press Shift+Tab to go back
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = updated.(App)
	if app.activeTab != 0 {
		t.Errorf("expected tab 0 after Shift+Tab, got %d", app.activeTab)
	}
}

func TestApp_TabWrapAround(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	// Shift+Tab from first tab should wrap to last
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = updated.(App)
	if app.activeTab != len(tabNames)-1 {
		t.Errorf("expected last tab, got %d", app.activeTab)
	}
}

func TestApp_NumberKeyNavigation(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	// Press '3' to go to Sessions tab (index 2)
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	app = updated.(App)
	if app.activeTab != 2 {
		t.Errorf("expected tab 2, got %d", app.activeTab)
	}
}

func TestApp_QuitKey(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestApp_ScanProgress(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	// Simulate scan batch message
	msg := ScanBatchMsg{Scanned: 50, Total: 100}
	updated, _ := app.Update(msg)
	app = updated.(App)

	if app.scanScanned != 50 || app.scanTotal != 100 {
		t.Errorf("expected 50/100, got %d/%d", app.scanScanned, app.scanTotal)
	}
}

func TestApp_ScanComplete(t *testing.T) {
	s := store.New()
	app := NewApp(s)

	msg := ScanCompleteMsg{}
	updated, _ := app.Update(msg)
	app = updated.(App)

	if !app.scanDone {
		t.Error("expected scanDone to be true")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `NewApp` undefined

**Step 3: Implement app shell**

Create `internal/tui/app.go`:

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/r/cicada/internal/store"
)

var tabNames = []string{"Dashboard", "Projects", "Sessions", "Analytics", "Agents", "Tools"}

// Messages from the scanner goroutine
type ScanBatchMsg struct {
	Scanned int
	Total   int
}

type ScanCompleteMsg struct{}

// App is the root Bubbletea model.
type App struct {
	store       *store.Store
	styles      Styles
	activeTab   int
	width       int
	height      int
	scanScanned int
	scanTotal   int
	scanDone    bool
}

// NewApp creates a new App model.
func NewApp(s *store.Store) App {
	theme := DefaultTheme()
	return App{
		store:  s,
		styles: NewStyles(theme),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			a.activeTab = (a.activeTab + 1) % len(tabNames)
			return a, nil
		case tea.KeyShiftTab:
			a.activeTab = (a.activeTab - 1 + len(tabNames)) % len(tabNames)
			return a, nil
		case tea.KeyEsc:
			return a, nil
		case tea.KeyCtrlC:
			return a, tea.Quit
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				return a, tea.Quit
			case "1", "2", "3", "4", "5", "6":
				idx := int(msg.Runes[0]-'0') - 1
				if idx < len(tabNames) {
					a.activeTab = idx
				}
				return a, nil
			}
		}

	case ScanBatchMsg:
		a.scanScanned = msg.Scanned
		a.scanTotal = msg.Total
		return a, nil

	case ScanCompleteMsg:
		a.scanDone = true
		return a, nil
	}

	return a, nil
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Tab bar
	b.WriteString(a.renderTabBar())
	b.WriteString("\n")

	// Content area
	contentHeight := a.height - 4 // tab bar + status bar + borders
	content := a.renderContent()
	contentStyle := lipgloss.NewStyle().Height(contentHeight).Width(a.width)
	b.WriteString(contentStyle.Render(content))
	b.WriteString("\n")

	// Status bar
	b.WriteString(a.renderStatusBar())

	return b.String()
}

func (a App) renderTabBar() string {
	var tabs []string
	for i, name := range tabNames {
		if i == a.activeTab {
			tabs = append(tabs, a.styles.TabActive.Render(name))
		} else {
			tabs = append(tabs, a.styles.TabInactive.Render(name))
		}
	}
	title := a.styles.Title.Render("cicada")
	return title + " " + strings.Join(tabs, "")
}

func (a App) renderContent() string {
	switch a.activeTab {
	case 0:
		return a.renderDashboard()
	default:
		return fmt.Sprintf("  %s view — coming soon", tabNames[a.activeTab])
	}
}

func (a App) renderDashboard() string {
	analytics := a.store.Analytics()

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s %s    %s %s    %s %s\n",
		a.styles.StatLabel.Render("Sessions:"),
		a.styles.StatValue.Render(fmt.Sprintf("%d", analytics.TotalSessions)),
		a.styles.StatLabel.Render("Tokens In:"),
		a.styles.StatValue.Render(formatTokens(analytics.TotalTokensIn)),
		a.styles.StatLabel.Render("Projects:"),
		a.styles.StatValue.Render(fmt.Sprintf("%d", analytics.ActiveProjects)),
	))
	return b.String()
}

func (a App) renderStatusBar() string {
	var status string
	if a.scanDone {
		status = fmt.Sprintf("Ready — %d sessions indexed", a.scanScanned)
	} else if a.scanTotal > 0 {
		status = fmt.Sprintf("Scanning... %d/%d sessions", a.scanScanned, a.scanTotal)
	} else {
		status = "Discovering projects..."
	}

	help := "? help  q quit"
	gap := a.width - len(status) - len(help) - 4
	if gap < 0 {
		gap = 1
	}
	return a.styles.StatusBar.Width(a.width).Render(
		"  " + status + strings.Repeat(" ", gap) + help,
	)
}

func formatTokens(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
```

**Step 4: Run tests**

Run: `make test`
Expected: PASS

**Step 5: Update main.go to launch the TUI with background scanner**

Modify `main.go`:

```go
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
```

**Step 6: Verify build and run**

Run: `make build`
Expected: builds successfully

Run: `make run`
Expected: TUI launches, shows scanning progress, then session count

**Step 7: Commit**

```bash
git add main.go internal/tui/app.go internal/tui/app_test.go
git commit -m "feat: add TUI app shell with tab navigation, status bar, and scanner integration"
```

---

### Task 11: Dashboard view — stats, sparkline, top tools

**Files:**
- Create: `internal/tui/components/sparkline.go`
- Create: `internal/tui/components/barchart.go`
- Test: `internal/tui/components/sparkline_test.go`
- Test: `internal/tui/components/barchart_test.go`
- Modify: `internal/tui/app.go` (renderDashboard)

**Step 1: Write sparkline test**

Create `internal/tui/components/sparkline_test.go`:

```go
package components

import "testing"

func TestSparkline_Render(t *testing.T) {
	data := []int{1, 3, 5, 2, 7, 4, 6}
	result := Sparkline(data, 20)

	if result == "" {
		t.Error("expected non-empty sparkline")
	}
	if len([]rune(result)) > 20 {
		t.Errorf("sparkline too long: %d runes (max 20)", len([]rune(result)))
	}
}

func TestSparkline_Empty(t *testing.T) {
	result := Sparkline(nil, 20)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}
}

func TestSparkline_SingleValue(t *testing.T) {
	result := Sparkline([]int{5}, 20)
	if result == "" {
		t.Error("expected non-empty sparkline for single value")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `Sparkline` undefined

**Step 3: Implement sparkline**

Create `internal/tui/components/sparkline.go`:

```go
package components

var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Sparkline renders a sparkline from data within maxWidth characters.
func Sparkline(data []int, maxWidth int) string {
	if len(data) == 0 {
		return ""
	}

	// If data is wider than maxWidth, downsample
	if len(data) > maxWidth {
		sampled := make([]int, maxWidth)
		ratio := float64(len(data)) / float64(maxWidth)
		for i := range sampled {
			idx := int(float64(i) * ratio)
			if idx >= len(data) {
				idx = len(data) - 1
			}
			sampled[i] = data[idx]
		}
		data = sampled
	}

	minVal, maxVal := data[0], data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	result := make([]rune, len(data))
	span := maxVal - minVal
	for i, v := range data {
		if span == 0 {
			result[i] = sparkBlocks[len(sparkBlocks)/2]
		} else {
			idx := (v - minVal) * (len(sparkBlocks) - 1) / span
			result[i] = sparkBlocks[idx]
		}
	}

	return string(result)
}
```

**Step 4: Run sparkline tests**

Run: `make test`
Expected: PASS

**Step 5: Write barchart test**

Create `internal/tui/components/barchart_test.go`:

```go
package components

import (
	"strings"
	"testing"
)

func TestBarChart_Render(t *testing.T) {
	items := []BarItem{
		{Label: "Read", Value: 100},
		{Label: "Edit", Value: 50},
		{Label: "Bash", Value: 25},
	}

	result := BarChart(items, 40)
	lines := strings.Split(result, "\n")

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestBarChart_Empty(t *testing.T) {
	result := BarChart(nil, 40)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
```

**Step 6: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `BarChart` undefined

**Step 7: Implement barchart**

Create `internal/tui/components/barchart.go`:

```go
package components

import (
	"fmt"
	"strings"
)

// BarItem represents a single bar in a bar chart.
type BarItem struct {
	Label string
	Value int
}

// BarChart renders a horizontal bar chart.
func BarChart(items []BarItem, maxWidth int) string {
	if len(items) == 0 {
		return ""
	}

	// Find max label width and max value
	maxLabel := 0
	maxVal := 0
	for _, item := range items {
		if len(item.Label) > maxLabel {
			maxLabel = len(item.Label)
		}
		if item.Value > maxVal {
			maxVal = item.Value
		}
	}

	barWidth := maxWidth - maxLabel - 10 // label + gap + count
	if barWidth < 5 {
		barWidth = 5
	}

	var lines []string
	for _, item := range items {
		var bar int
		if maxVal > 0 {
			bar = item.Value * barWidth / maxVal
		}
		if bar < 1 && item.Value > 0 {
			bar = 1
		}
		label := fmt.Sprintf("%-*s", maxLabel, item.Label)
		line := fmt.Sprintf("%s %s %d", label, strings.Repeat("█", bar), item.Value)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
```

**Step 8: Run all tests**

Run: `make test`
Expected: PASS

**Step 9: Commit**

```bash
git add internal/tui/components/
git commit -m "feat: add sparkline and bar chart TUI components"
```

**Step 10: Enhance dashboard view**

Update `renderDashboard` in `internal/tui/app.go` to use sparkline and barchart components, showing sessions-by-date sparkline, top tools bar chart, model distribution, and work mode split. Wire up the data from `store.Analytics()`.

This is a rendering-only change — verify visually with `make run`.

**Step 11: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: enhance dashboard with sparkline, tool chart, and model distribution"
```

---

### Task 12: Projects view — list projects with session counts

**Files:**
- Create: `internal/tui/views/projects.go`
- Test: `internal/tui/views/projects_test.go`
- Modify: `internal/tui/app.go` (wire Projects view)

**Step 1: Write test**

Create `internal/tui/views/projects_test.go`:

```go
package views

import (
	"testing"
	"time"

	"github.com/r/cicada/internal/model"
	"github.com/r/cicada/internal/store"
)

func TestProjectsView_Render(t *testing.T) {
	s := store.New()
	now := time.Now()
	s.Add(&model.SessionMeta{
		UUID: "u1", Slug: "s1", ProjectPath: "-Users-r-work-myproject",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 5,
	})
	s.Add(&model.SessionMeta{
		UUID: "u2", Slug: "s2", ProjectPath: "-Users-r-work-myproject",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 3,
	})
	s.Add(&model.SessionMeta{
		UUID: "u3", Slug: "s3", ProjectPath: "-Users-r-work-other",
		StartTime: now, EndTime: now.Add(time.Minute),
		Models: map[string]int{}, ToolUsage: map[string]int{},
		SkillsUsed: map[string]int{}, CommandsUsed: map[string]int{},
		FileOps: map[string]int{}, MessageCount: 1,
	})

	view := NewProjectsView(s)
	content := view.View(80, 24)

	if content == "" {
		t.Error("expected non-empty view")
	}
}

func TestProjectsView_Empty(t *testing.T) {
	s := store.New()
	view := NewProjectsView(s)
	content := view.View(80, 24)

	if content == "" {
		t.Error("expected non-empty view even when empty")
	}
}
```

**Step 2: Run test — expect failure**

Run: `make test`
Expected: FAIL — `NewProjectsView` undefined

**Step 3: Implement projects view**

Create `internal/tui/views/projects.go`:

```go
package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/r/cicada/internal/store"
)

// ProjectRow holds display data for a project.
type ProjectRow struct {
	Name         string
	Path         string
	SessionCount int
	LastActive   string
}

// ProjectsView shows a list of projects.
type ProjectsView struct {
	store    *store.Store
	selected int
}

// NewProjectsView creates a new ProjectsView.
func NewProjectsView(s *store.Store) *ProjectsView {
	return &ProjectsView{store: s}
}

// View renders the projects list.
func (v *ProjectsView) View(width, height int) string {
	projects := v.store.Projects()
	if len(projects) == 0 {
		return "\n  No projects found. Waiting for scan to complete..."
	}

	// Build rows
	rows := make([]ProjectRow, 0, len(projects))
	for _, p := range projects {
		sessions := v.store.SessionsByProject(p)
		lastActive := ""
		for _, s := range sessions {
			if !s.StartTime.IsZero() {
				ts := s.StartTime.Format("2006-01-02 15:04")
				if ts > lastActive {
					lastActive = ts
				}
			}
		}

		// Decode project path: "-Users-r-work-myproject" → "/Users/r/work/myproject"
		decoded := "/" + strings.ReplaceAll(strings.TrimPrefix(p, "-"), "-", "/")

		rows = append(rows, ProjectRow{
			Name:         decoded,
			Path:         p,
			SessionCount: len(sessions),
			LastActive:   lastActive,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].LastActive > rows[j].LastActive
	})

	var b strings.Builder
	b.WriteString("\n")
	header := fmt.Sprintf("  %-50s %10s %20s", "Project", "Sessions", "Last Active")
	b.WriteString(header + "\n")
	b.WriteString("  " + strings.Repeat("─", 82) + "\n")

	for i, row := range rows {
		name := row.Name
		if len(name) > 50 {
			name = "..." + name[len(name)-47:]
		}
		prefix := "  "
		if i == v.selected {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-50s %10d %20s", prefix, name, row.SessionCount, row.LastActive)
		b.WriteString(line + "\n")
	}

	return b.String()
}
```

**Step 4: Run tests**

Run: `make test`
Expected: PASS

**Step 5: Wire projects view into app.go**

Add the projects view case in `renderContent()`. Import `views` package.

**Step 6: Commit**

```bash
git add internal/tui/views/projects.go internal/tui/views/projects_test.go internal/tui/app.go
git commit -m "feat: add projects list view with session counts and last active"
```

---

### Task 13: Sessions list view

**Files:**
- Create: `internal/tui/views/sessions.go`
- Test: `internal/tui/views/sessions_test.go`
- Modify: `internal/tui/app.go`

Follow same pattern as Task 12. Table shows: Slug | Project | Date | Duration | Tokens | Tools count. Sortable by columns.

**Step 1: Write test, Step 2: Run failing, Step 3: Implement, Step 4: Run passing, Step 5: Commit.**

```bash
git commit -m "feat: add sessions list view with sorting and filtering"
```

---

### Task 14: Session detail view with sub-tabs

**Files:**
- Create: `internal/tui/views/session_detail.go`
- Test: `internal/tui/views/session_detail_test.go`
- Modify: `internal/tui/app.go` (navigation from sessions list → detail)
- Modify: `internal/parser/extract.go` (add ExtractSessionDetail for lazy loading)
- Test: `internal/parser/extract_detail_test.go`

This task adds: Overview tab (stats grid, initial prompt, models, branches), Timeline tab (events list), Files tab (file operations), Agents tab (subagents), Tools tab (tool breakdown).

The detail view needs lazy loading: when the user presses Enter on a session in the list, parse the full JSONL file and cache the result in the store.

**Step 1: Write parser test for ExtractSessionDetail**
**Step 2: Run failing**
**Step 3: Implement ExtractSessionDetail**
**Step 4: Run passing**
**Step 5: Write TUI test for session detail**
**Step 6: Run failing**
**Step 7: Implement session detail view**
**Step 8: Wire navigation (Enter from sessions list → detail, Esc back)**
**Step 9: Run passing**
**Step 10: Commit**

```bash
git commit -m "feat: add session detail view with overview, timeline, files, agents, tools tabs"
```

---

### Task 15: Analytics view

**Files:**
- Create: `internal/tui/components/heatmap.go`
- Test: `internal/tui/components/heatmap_test.go`
- Create: `internal/tui/views/analytics.go`
- Test: `internal/tui/views/analytics_test.go`
- Modify: `internal/tui/app.go`

Renders: sessions per day sparkline, temporal heatmap (7x24 grid), tool usage bar chart, model distribution, work mode distribution. Time period filter (7d/30d/90d/all).

**Step 1: Write heatmap test, Step 2: Fail, Step 3: Implement, Step 4: Pass**
**Step 5: Write analytics view test, Step 6: Fail, Step 7: Implement, Step 8: Pass**
**Step 9: Commit**

```bash
git commit -m "feat: add analytics view with heatmap, sparklines, and tool charts"
```

---

### Task 16: Agents view

**Files:**
- Create: `internal/tui/views/agents.go`
- Test: `internal/tui/views/agents_test.go`
- Modify: `internal/tui/app.go`

Lists all agent types used across sessions with run count, last used. Aggregated from `store.AllSessions()` by iterating SubagentCount and tool usage.

Follow standard pattern: test → fail → implement → pass → commit.

```bash
git commit -m "feat: add agents usage view"
```

---

### Task 17: Tools view

**Files:**
- Create: `internal/tui/views/tools.go`
- Test: `internal/tui/views/tools_test.go`
- Modify: `internal/tui/app.go`

Lists all tools (built-in + MCP) with call counts, session counts. MCP tools grouped by server (split on `__`).

Follow standard pattern: test → fail → implement → pass → commit.

```bash
git commit -m "feat: add tools usage view with MCP server grouping"
```

---

### Task 18: Search and filter

**Files:**
- Create: `internal/tui/components/filter.go`
- Test: `internal/tui/components/filter_test.go`
- Modify: `internal/tui/views/sessions.go`
- Modify: `internal/tui/views/projects.go`

Add `/` key to open a text input filter bar. Filter sessions by slug, initial prompt text. Filter projects by name. Esc to clear filter.

Follow standard pattern: test → fail → implement → pass → commit.

```bash
git commit -m "feat: add search/filter with / key shortcut"
```

---

### Task 19: Final integration and polish

**Files:**
- Modify: `internal/tui/app.go` (help overlay with `?`)
- Modify: `internal/tui/styles.go` (finalize colors)
- Test: run full test suite

**Step 1: Add help overlay**

When user presses `?`, show a modal overlay listing all keybindings. Press any key to dismiss.

**Step 2: Run all tests**

Run: `make test`
Expected: all pass

**Step 3: Run lint**

Run: `make lint`
Expected: clean

**Step 4: Manual testing**

Run: `make run`
Verify: TUI starts, scans sessions, all 6 views render, tab navigation works, Enter/Esc navigation works, search works.

**Step 5: Commit**

```bash
git commit -m "feat: add help overlay and final polish"
```

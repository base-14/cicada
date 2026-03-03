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

package analysis

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// extractToolResultContent handles both string content and the
// [{type:"text", text:"..."}] array format from Claude CLI JSONL.
func extractToolResultContent(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if arr, ok := v.([]any); ok {
		var parts []string
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// ToolCall represents a single tool invocation extracted from a Claude CLI JSONL file.
type ToolCall struct {
	Index  int            `json:"index"`
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Input  map[string]any `json:"input"`
	Result string         `json:"result"`
}

// SearchPath returns the path/cwd from the tool call input,
// checking "path" first then falling back to "cwd".
func (tc ToolCall) SearchPath() string {
	if p, _ := tc.Input["path"].(string); p != "" {
		return p
	}
	cwd, _ := tc.Input["cwd"].(string)
	return cwd
}

// HasLumenSearch returns true if the raw JSONL file contains at least one
// mcp__lumen__semantic_search tool call.
func HasLumenSearch(rawPath string) (bool, error) {
	calls, err := ExtractToolCalls(rawPath)
	if err != nil {
		return false, err
	}
	for _, tc := range calls {
		if tc.Name == "mcp__lumen__semantic_search" {
			return true, nil
		}
	}
	return false, nil
}

// ExtractToolCalls parses a Claude CLI raw JSONL file and returns all tool
// calls in order. It handles both top-level events (type:"tool_use") and
// the nested message.content[] format used by the Claude CLI.
func ExtractToolCalls(rawPath string) ([]ToolCall, error) {
	f, err := os.Open(rawPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var calls []ToolCall
	pendingIDs := map[string]int{} // tool_use ID -> index in calls slice
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	idx := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var evt map[string]any
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}

		blocks := extractContentBlocks(evt)

		for _, block := range blocks {
			blockType, _ := block["type"].(string)

			if blockType == "tool_use" {
				id, _ := block["id"].(string)
				name, _ := block["name"].(string)
				inputMap, _ := block["input"].(map[string]any)
				calls = append(calls, ToolCall{
					Index: idx,
					ID:    id,
					Name:  name,
					Input: inputMap,
				})
				if id != "" {
					pendingIDs[id] = len(calls) - 1
				}
				idx++
			}

			if blockType == "tool_result" {
				toolUseID, _ := block["tool_use_id"].(string)
				if i, ok := pendingIDs[toolUseID]; ok {
					calls[i].Result = extractToolResultContent(block["content"])
					delete(pendingIDs, toolUseID)
				}
			}
		}
	}

	return calls, nil
}

// extractContentBlocks collects tool_use/tool_result blocks from a JSONL event.
// It handles both top-level events and nested message.content[] blocks.
func extractContentBlocks(evt map[string]any) []map[string]any {
	var blocks []map[string]any

	evtType, _ := evt["type"].(string)
	if evtType == "tool_use" || evtType == "tool_result" {
		blocks = append(blocks, evt)
	}

	if msg, ok := evt["message"].(map[string]any); ok {
		if content, ok := msg["content"].([]any); ok {
			for _, item := range content {
				if block, ok := item.(map[string]any); ok {
					blocks = append(blocks, block)
				}
			}
		}
	}

	return blocks
}

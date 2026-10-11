package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Metrics struct {
	TestStatus   string  `json:"test_status,omitempty"`
	Agent        string  `json:"agent,omitempty"`
	Runner       string  `json:"runner,omitempty"`
	Model        string  `json:"model,omitempty"`
	Error        string  `json:"error,omitempty"`
	CostUnknown  bool    `json:"cost_unknown,omitempty"`
	CostUSD      float64 `json:"cost_usd"`
	DurationMS   int64   `json:"duration_ms"`
	InputTokens  int64   `json:"input_tokens"`
	CacheRead    int64   `json:"cache_read"`
	CacheCreated int64   `json:"cache_created"`
	OutputTokens int64   `json:"output_tokens"`
}

type resultEvent struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype"`
	IsError bool            `json:"is_error"`
	Error   json.RawMessage `json:"error"`
	Item    struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"item"`
	TotalCost  float64 `json:"total_cost_usd"`
	DurationMS int64   `json:"duration_ms"`
	Result     string  `json:"result"`
	Usage      usage   `json:"usage"`
}

type usage struct {
	InputTokens              int64 `json:"input_tokens"`
	CachedInputTokens        int64 `json:"cached_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
}

// ExtractFromJSONL accepts native CLI and SDK events from both providers.
func ExtractFromJSONL(rawPath string) (*Metrics, error) {
	m, _, err := extract(rawPath)
	return m, err
}

func ExtractAnswer(rawPath string) (string, error) {
	_, answer, err := extract(rawPath)
	return answer, err
}

func extract(rawPath string) (*Metrics, string, error) {
	f, err := os.Open(rawPath)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	var m *Metrics
	var answer string
	var failure error
	for scanner.Scan() {
		var evt resultEvent
		if json.Unmarshal(scanner.Bytes(), &evt) != nil {
			continue
		}
		switch evt.Type {
		case "result":
			m = &Metrics{CostUSD: evt.TotalCost, DurationMS: evt.DurationMS,
				InputTokens: evt.Usage.InputTokens, CacheRead: evt.Usage.CacheReadInputTokens,
				CacheCreated: evt.Usage.CacheCreationInputTokens, OutputTokens: evt.Usage.OutputTokens}
			answer = evt.Result
			if evt.IsError || (evt.Subtype != "" && evt.Subtype != "success") {
				failure = fmt.Errorf("agent result: %s %s", evt.Subtype, evt.Result)
			}
		case "turn.completed":
			if m == nil {
				m = &Metrics{CostUnknown: true}
			}
			// Codex input_tokens INCLUDES cached_input_tokens; normalize to Claude's
			// exclusive input count so cache tokens are not counted twice in reports.
			m.InputTokens += max(0, evt.Usage.InputTokens-evt.Usage.CachedInputTokens)
			m.CacheRead += evt.Usage.CachedInputTokens
			m.OutputTokens += evt.Usage.OutputTokens
		case "item.completed":
			if evt.Item.Type == "agent_message" {
				answer = evt.Item.Text
			}
		case "turn.failed":
			failure = fmt.Errorf("agent turn failed: %s", evt.Error)
		}
	}
	if err := scanner.Err(); err != nil {
		return m, answer, err
	}
	if failure != nil {
		return m, answer, failure
	}
	if m == nil {
		return nil, answer, fmt.Errorf("no result event found in %s", rawPath)
	}
	return m, answer, nil
}

func (m *Metrics) CostLabel() string {
	if m.CostUnknown {
		return "n/a"
	}
	return fmt.Sprintf("$%.4f", m.CostUSD)
}

func (m *Metrics) TotalTokens() int64 {
	return m.InputTokens + m.CacheRead + m.CacheCreated + m.OutputTokens
}

func LoadFromFile(path string) (*Metrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Metrics
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Metrics) SaveToFile(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

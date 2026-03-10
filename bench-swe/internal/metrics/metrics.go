package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Metrics struct {
	CostUSD      float64 `json:"cost_usd"`
	DurationMS   int64   `json:"duration_ms"`
	InputTokens  int64   `json:"input_tokens"`
	CacheRead    int64   `json:"cache_read"`
	CacheCreated int64   `json:"cache_created"`
	OutputTokens int64   `json:"output_tokens"`
}

type resultEvent struct {
	Type       string  `json:"type"`
	TotalCost  float64 `json:"total_cost_usd"`
	DurationMS int64   `json:"duration_ms"`
	Result     string  `json:"result"`
	Usage      usage   `json:"usage"`
}

type usage struct {
	InputTokens              int64 `json:"input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
}

func ExtractFromJSONL(rawPath string) (*Metrics, error) {
	f, err := os.Open(rawPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", rawPath, err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, `"type":"result"`) {
			continue
		}

		var evt resultEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		if evt.Type != "result" {
			continue
		}

		return &Metrics{
			CostUSD:      evt.TotalCost,
			DurationMS:   evt.DurationMS,
			InputTokens:  evt.Usage.InputTokens,
			CacheRead:    evt.Usage.CacheReadInputTokens,
			CacheCreated: evt.Usage.CacheCreationInputTokens,
			OutputTokens: evt.Usage.OutputTokens,
		}, nil
	}

	return nil, fmt.Errorf("no result event found in %s", rawPath)
}

func ExtractAnswer(rawPath string) (string, error) {
	f, err := os.Open(rawPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, `"type":"result"`) {
			continue
		}

		var evt resultEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		if evt.Type == "result" {
			return evt.Result, nil
		}
	}
	return "", fmt.Errorf("no result event found in %s", rawPath)
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

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
)

func TestDiscoverPairs(t *testing.T) {
	t.Run("dir with metrics files", func(t *testing.T) {
		dir := t.TempDir()

		m := &metrics.Metrics{CostUSD: 0.01, DurationMS: 5000, InputTokens: 100, OutputTokens: 50}
		data, _ := json.Marshal(m)
		_ = os.WriteFile(filepath.Join(dir, "task1-baseline-metrics.json"), data, 0o644)
		_ = os.WriteFile(filepath.Join(dir, "task1-with-lumen-metrics.json"), data, 0o644)

		pairs := discoverPairs(dir)
		if len(pairs) != 2 {
			t.Fatalf("got %d pairs, want 2", len(pairs))
		}
		if _, ok := pairs["task1-baseline"]; !ok {
			t.Error("missing key task1-baseline")
		}
		if _, ok := pairs["task1-with-lumen"]; !ok {
			t.Error("missing key task1-with-lumen")
		}
	})

	t.Run("empty dir", func(t *testing.T) {
		dir := t.TempDir()
		pairs := discoverPairs(dir)
		if len(pairs) != 0 {
			t.Errorf("got %d pairs, want 0", len(pairs))
		}
	})
}

func TestFormatDelta(t *testing.T) {
	tests := []struct {
		name  string
		delta float64
		base  float64
		want  string
	}{
		{
			name:  "positive delta",
			delta: 0.005,
			base:  0.01,
			want:  "+",
		},
		{
			name:  "negative delta",
			delta: -0.005,
			base:  0.01,
			want:  "-", // negative sign from the number itself
		},
		{
			name:  "zero base",
			delta: 0.01,
			base:  0,
			want:  "+",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDelta(tt.delta, tt.base, "$%.4f")
			if !strings.Contains(got, tt.want) {
				t.Errorf("formatDelta(%f, %f) = %q, does not contain %q", tt.delta, tt.base, got, tt.want)
			}
		})
	}
}

func TestFormatDeltaTime(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		got := formatDeltaTime(5000, 10000)
		if !strings.Contains(got, "+5.0s") {
			t.Errorf("got %q, want to contain +5.0s", got)
		}
		if !strings.Contains(got, "+50%") {
			t.Errorf("got %q, want to contain +50%%", got)
		}
	})

	t.Run("negative", func(t *testing.T) {
		got := formatDeltaTime(-3000, 10000)
		if !strings.Contains(got, "-3.0s") {
			t.Errorf("got %q, want to contain -3.0s", got)
		}
	})
}

func TestFormatDeltaInt(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		got := formatDeltaInt(500, 1000)
		if !strings.Contains(got, "+500") {
			t.Errorf("got %q, want to contain +500", got)
		}
	})

	t.Run("negative", func(t *testing.T) {
		got := formatDeltaInt(-200, 1000)
		if !strings.Contains(got, "-200") {
			t.Errorf("got %q, want to contain -200", got)
		}
	})
}

func TestRatingToRank(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Perfect", 3},
		{"Good", 2},
		{"Poor", 1},
		{"Unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ratingToRank(tt.input)
			if got != tt.want {
				t.Errorf("ratingToRank(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

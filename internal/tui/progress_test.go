// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tui

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewProgress_ReturnsNonNil(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)
	if p == nil {
		t.Fatal("NewProgress returned nil")
	}
}

func TestProgress_StartStop(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Start("Indexing", 10)
	p.Update(5, "Processing file 5/10: foo.go")
	p.Stop()

	output := buf.String()
	if output == "" {
		t.Fatal("expected output on writer, got empty string")
	}
}

func TestProgress_Info(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Info("Indexing /tmp/project (model: jina-v2, dims: 768)")

	output := buf.String()
	if !strings.Contains(output, "Indexing") {
		t.Errorf("expected output to contain 'Indexing', got %q", output)
	}
}

func TestProgress_AsProgressFunc(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	fn := p.AsProgressFunc()

	fn(0, 5, "Processing file 1/5: a.go")
	fn(3, 5, "Processing file 3/5: c.go")
	fn(5, 5, "Done")

	output := buf.String()
	if output == "" {
		t.Fatal("expected output from progress func callback, got empty string")
	}
}

func TestProgress_ZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	// Should not panic when total is 0
	p.Start("Scanning", 0)
	p.Update(0, "scanning...")
	p.Stop()
}

func TestProgress_RenderClearsLine(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Start("Indexing", 100)
	buf.Reset()

	p.Update(50, "a very long title that should be visible")
	output := buf.String()
	if !strings.Contains(output, "\r\033[K") {
		t.Errorf("expected render to contain \\r\\033[K clear sequence, got %q", output)
	}

	buf.Reset()
	p.Update(60, "short")
	output = buf.String()
	if !strings.Contains(output, "\r\033[K") {
		t.Errorf("expected second render to also contain \\r\\033[K clear sequence, got %q", output)
	}

	p.Stop()
}

func TestProgress_NonTerminal_NoCursorSequences(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Start("Indexing", 10)
	p.Update(5, "Processing")
	p.Stop()

	output := buf.String()
	if strings.Contains(output, "\x1b[?25l") {
		t.Error("non-terminal output should not contain cursor-hide sequence")
	}
	if strings.Contains(output, "\x1b[?25h") {
		t.Error("non-terminal output should not contain cursor-show sequence")
	}
}

func TestProgress_StopIdempotent(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Start("Indexing", 10)
	p.Update(5, "working...")
	p.Stop()
	p.Stop() // second call should not panic
}

func TestProgress_RestoreCursor_SafeWithoutStart(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	// RestoreCursor should not panic even if Start was never called
	p.RestoreCursor()

	output := buf.String()
	if strings.Contains(output, "\x1b[?25h") {
		t.Error("RestoreCursor without Start should not emit cursor-show")
	}
}

func TestProgress_TitleTruncation(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)
	p.widthFunc = func() int { return 60 }

	p.Start("Indexing", 100)
	buf.Reset()

	longTitle := strings.Repeat("x", 200)
	p.Update(50, longTitle)

	output := buf.String()
	// The rendered line (excluding \r\033[K) should not exceed 60 characters
	line := strings.TrimPrefix(output, "\r\033[K")
	line = strings.TrimRight(line, "\n")
	if len(line) > 60 {
		t.Errorf("rendered line length %d exceeds terminal width 60: %q", len(line), line)
	}

	p.Stop()
}

func TestProgress_AsProgressFunc_InfoOnZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	fn := p.AsProgressFunc()
	fn(0, 0, "Scanning directories")

	output := buf.String()
	if !strings.Contains(output, "Scanning") {
		t.Errorf("expected info output for zero-total call, got %q", output)
	}
}

func TestProgress_AsProgressFunc_NonTerminalPlainText(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf) // bytes.Buffer is not a terminal

	fn := p.AsProgressFunc()
	fn(0, 10, "file1.go")
	fn(5, 10, "file5.go")
	fn(10, 10, "done")

	output := buf.String()
	if !strings.Contains(output, "Indexing:") {
		t.Errorf("non-terminal progress should emit plain text with 'Indexing:', got %q", output)
	}
	if strings.Contains(output, "\r") {
		t.Error("non-terminal progress should not use carriage return")
	}
}

func TestProgress_RenderFormat(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)
	p.widthFunc = func() int { return 120 }

	p.Start("Indexing", 100)
	buf.Reset()

	p.Update(42, "Processing file 42/100: main.go")

	output := buf.String()
	line := strings.TrimPrefix(output, "\r\033[K")

	if !strings.Contains(line, "Processing file 42/100: main.go") {
		t.Errorf("expected title in output, got %q", line)
	}
	if !strings.Contains(line, "[042/100]") {
		t.Errorf("expected [042/100] count in output, got %q", line)
	}
	if !strings.Contains(line, "42%") {
		t.Errorf("expected 42%% in output, got %q", line)
	}

	p.Stop()
}

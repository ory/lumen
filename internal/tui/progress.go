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

// Package tui provides terminal UI components for lumen CLI output.
package tui

import (
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// Progress wraps a custom progress renderer and PTerm prefix printers to
// display indexing progress, status messages, and completion summaries.
// All output is written to the configured writer (typically os.Stderr to
// avoid interfering with MCP stdio on stdout).
type Progress struct {
	writer       io.Writer
	isTerminal   bool
	info         pterm.PrefixPrinter
	widthFunc    func() int // overridable for tests; nil means use real terminal width

	// Custom progress bar state (replaces pterm ProgressbarPrinter).
	total        int
	current      int
	active       bool
	cursorHidden bool
}

// NewProgress creates a new Progress that writes to w.
// When w is not a terminal, PTerm styling is disabled to prevent ANSI
// escape sequences from leaking to stdout via PTerm's global output writer.
func NewProgress(w io.Writer) *Progress {
	f, isFile := w.(*os.File)
	isTerm := isFile && term.IsTerminal(int(f.Fd()))
	if !isTerm {
		pterm.DisableStyling()
	}
	return &Progress{
		writer:     w,
		info:       *pterm.Info.WithWriter(w),
		isTerminal: isTerm,
	}
}

// Start initialises and displays a progress bar with the given title and total.
func (p *Progress) Start(title string, total int) {
	if total <= 0 {
		return
	}
	p.total = total
	p.current = 0
	p.active = true
	if p.isTerminal {
		_, _ = fmt.Fprint(p.writer, "\x1b[?25l") // hide cursor
		p.cursorHidden = true
	}
	p.render(title)
}

// Update sets the progress bar to current and updates the title.
func (p *Progress) Update(current int, message string) {
	if !p.active {
		return
	}
	p.current = current
	p.render(message)
}

// render writes a single progress line, clearing the current line first.
func (p *Progress) render(message string) {
	if p.total <= 0 {
		return
	}

	pct := 0
	if p.total > 0 {
		pct = p.current * 100 / p.total
	}

	// Format: message [NNNN/TOTAL] PP%
	padding := 1 + int(math.Log10(float64(p.total)))
	suffix := fmt.Sprintf(" [%0*d/%d] %3d%%", padding, p.current, p.total, pct)

	width := p.termWidth()
	maxMsg := max(width-len(suffix), 0)
	if len(message) > maxMsg {
		if maxMsg > 3 {
			message = message[:maxMsg-3] + "..."
		} else {
			message = ""
		}
	}

	// \r returns to column 0; \033[K clears to end of line.
	_, _ = fmt.Fprintf(p.writer, "\r\033[K%s%s", message, suffix)
}

// Stop stops the progress bar.
func (p *Progress) Stop() {
	if !p.active {
		return
	}
	p.active = false
	// Clear the progress line and move to a new line.
	_, _ = fmt.Fprint(p.writer, "\r\033[K")
	if p.cursorHidden {
		_, _ = fmt.Fprint(p.writer, "\x1b[?25h") // show cursor
		p.cursorHidden = false
	}
}

// RestoreCursor ensures the terminal cursor is visible. Safe to call
// multiple times or even if Start was never called. Intended for use
// with defer to guarantee cursor restoration on all exit paths.
func (p *Progress) RestoreCursor() {
	if p.cursorHidden {
		_, _ = fmt.Fprint(p.writer, "\x1b[?25h")
		p.cursorHidden = false
	}
}

// AsProgressFunc returns a callback compatible with index.ProgressFunc.
// Calls with total=0 print an info line; the progress bar is started on
// the first call with total>0 and stopped when current reaches total.
// When writing to a non-terminal (e.g. a log file), progress bar is skipped
// entirely and a plain-text status line is emitted at most every 5 seconds.
func (p *Progress) AsProgressFunc() func(current, total int, message string) {
	const logInterval = 5 * time.Second
	started := false
	var lastLog time.Time
	return func(current, total int, message string) {
		if total == 0 {
			p.Info(message)
			return
		}
		if !p.isTerminal {
			// Non-terminal: emit a plain status line every logInterval.
			now := time.Now()
			if current < total && now.Sub(lastLog) < logInterval {
				return
			}
			lastLog = now
			pct := current * 100 / total
			_, _ = fmt.Fprintf(p.writer, "Indexing: %d/%d (%d%%)\n", current, total, pct)
			return
		}
		if !started {
			p.Start("Indexing", total)
			started = true
		}
		p.Update(current, message)
		if current >= total {
			p.Stop()
			started = false
		}
	}
}

// Info prints an informational message.
func (p *Progress) Info(msg string) {
	p.info.Println(msg)
}

// termWidth returns the terminal width, falling back to 80.
func (p *Progress) termWidth() int {
	if p.widthFunc != nil {
		return p.widthFunc()
	}
	if f, ok := p.writer.(*os.File); ok {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
			return w
		}
	}
	return 80
}

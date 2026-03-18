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
	"io"
	"os"

	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// Progress wraps PTerm components to display indexing progress, status
// messages, and completion summaries. All output is written to the
// configured writer (typically os.Stderr to avoid interfering with
// MCP stdio on stdout).
type Progress struct {
	writer  io.Writer
	bar     *pterm.ProgressbarPrinter
	info    pterm.PrefixPrinter
	success pterm.PrefixPrinter
	errpr   pterm.PrefixPrinter
}

// NewProgress creates a new Progress that writes to w.
// When w is not a terminal, PTerm styling is disabled to prevent ANSI
// escape sequences (including cursor hide/show) from leaking to stdout
// via PTerm's global output writer.
func NewProgress(w io.Writer) *Progress {
	f, isFile := w.(*os.File)
	if !isFile || !term.IsTerminal(int(f.Fd())) {
		pterm.DisableStyling()
	}
	// Redirect PTerm's global output (used for cursor control etc.) to w
	// so nothing escapes to the default os.Stdout.
	pterm.SetDefaultOutput(w)
	return &Progress{
		writer:  w,
		info:    *pterm.Info.WithWriter(w),
		success: *pterm.Success.WithWriter(w),
		errpr:   *pterm.Error.WithWriter(w),
	}
}

// Start initialises and displays a progress bar with the given title and total.
func (p *Progress) Start(title string, total int) {
	bar, _ := pterm.DefaultProgressbar.
		WithTitle(title).
		WithTotal(total).
		WithWriter(p.writer).
		WithShowCount(true).
		WithShowPercentage(true).
		Start()
	p.bar = bar
}

// Update sets the progress bar to current and updates the title.
func (p *Progress) Update(current int, message string) {
	if p.bar == nil {
		return
	}
	p.bar.UpdateTitle(message)
	// Increment is additive, so compute the delta from the bar's current value.
	delta := current - p.bar.Current
	if delta > 0 {
		p.bar.Add(delta)
	}
}

// Stop stops the progress bar.
func (p *Progress) Stop() {
	if p.bar == nil {
		return
	}
	_, _ = p.bar.Stop()
	p.bar = nil
}

// AsProgressFunc returns a callback compatible with index.ProgressFunc.
// Calls with total=0 print an info line; the progress bar is started on
// the first call with total>0 and stopped when current reaches total.
func (p *Progress) AsProgressFunc() func(current, total int, message string) {
	started := false
	return func(current, total int, message string) {
		if total == 0 {
			p.Info(message)
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

// Complete prints a success/completion message.
func (p *Progress) Complete(msg string) {
	p.success.Println(msg)
}

// UpToDate prints a message indicating the index is already current.
func (p *Progress) UpToDate() {
	p.info.Println("Index is already up to date.")
}

// Error prints an error-styled message.
func (p *Progress) Error(msg string) {
	p.errpr.Println(msg)
}

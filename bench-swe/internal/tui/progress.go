// Package tui provides terminal UI components for bench-swe CLI output.
package tui

import (
	"fmt"
	"io"
	"math"
	"os"

	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// Progress wraps a custom progress renderer and PTerm prefix printers to
// display benchmark progress, status messages, and completion summaries.
// All output is written to the configured writer (typically os.Stderr).
//
// NOTE: NewProgress sets PTerm's global styling state for non-terminals.
// Create only one Progress instance per process.
type Progress struct {
	writer       io.Writer
	isTerminal   bool
	info         pterm.PrefixPrinter
	success      pterm.PrefixPrinter
	warn         pterm.PrefixPrinter
	errPrinter   pterm.PrefixPrinter
	spinner      *pterm.SpinnerPrinter
	widthFunc    func() int // overridable for tests; nil means use real terminal width

	// Custom progress bar state (replaces pterm ProgressbarPrinter).
	total        int
	current      int
	active       bool
	cursorHidden bool
}

// NewProgress creates a new Progress that writes to w.
// When w is not a terminal, PTerm styling is disabled to prevent ANSI
// escape sequences from corrupting piped output.
func NewProgress(w io.Writer) *Progress {
	f, isFile := w.(*os.File)
	isTerm := isFile && term.IsTerminal(int(f.Fd()))
	if !isTerm {
		pterm.DisableStyling()
	}
	return &Progress{
		writer:     w,
		isTerminal: isTerm,
		info:       *pterm.Info.WithWriter(w),
		success:    *pterm.Success.WithWriter(w),
		warn:       *pterm.Warning.WithWriter(w),
		errPrinter: *pterm.Error.WithWriter(w),
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

// StartSpinner shows an indeterminate spinner with the given message.
func (p *Progress) StartSpinner(msg string) {
	sp, err := pterm.DefaultSpinner.WithWriter(p.writer).Start(msg)
	if err != nil {
		p.info.Println(msg)
		return
	}
	p.spinner = sp
}

// StopSpinner stops the active spinner.
func (p *Progress) StopSpinner() {
	if p.spinner == nil {
		return
	}
	_ = p.spinner.Stop()
	p.spinner = nil
}

// PrintTable renders headers and rows as a styled table to the writer.
func (p *Progress) PrintTable(headers []string, rows [][]string) {
	data := pterm.TableData{headers}
	for _, row := range rows {
		data = append(data, row)
	}
	_ = pterm.DefaultTable.WithHasHeader(true).WithWriter(p.writer).WithData(data).Render()
}

// Info prints an informational message.
func (p *Progress) Info(msg string) { p.info.Println(msg) }

// Complete prints a success/completion message.
func (p *Progress) Complete(msg string) { p.success.Println(msg) }

// Warn prints a warning message.
func (p *Progress) Warn(msg string) { p.warn.Println(msg) }

// Error prints an error-styled message.
func (p *Progress) Error(msg string) { p.errPrinter.Println(msg) }

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

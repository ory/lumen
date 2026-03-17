// Package tui provides terminal UI components for bench-swe CLI output.
package tui

import (
	"io"
	"os"

	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// Progress wraps PTerm components to display benchmark progress, status
// messages, and completion summaries. All output is written to the
// configured writer (typically os.Stderr).
//
// NOTE: NewProgress sets PTerm's global output and styling state.
// Create only one Progress instance per process.
type Progress struct {
	writer     io.Writer
	bar        *pterm.ProgressbarPrinter
	spinner    *pterm.SpinnerPrinter
	info       pterm.PrefixPrinter
	success    pterm.PrefixPrinter
	warn       pterm.PrefixPrinter
	errPrinter pterm.PrefixPrinter
}

// NewProgress creates a new Progress that writes to w.
// When w is not a terminal, PTerm styling is disabled to prevent ANSI
// escape sequences from corrupting piped output.
func NewProgress(w io.Writer) *Progress {
	f, isFile := w.(*os.File)
	if !isFile || !term.IsTerminal(int(f.Fd())) {
		pterm.DisableStyling()
	}
	pterm.SetDefaultOutput(w)
	return &Progress{
		writer:     w,
		info:       *pterm.Info.WithWriter(w),
		success:    *pterm.Success.WithWriter(w),
		warn:       *pterm.Warning.WithWriter(w),
		errPrinter: *pterm.Error.WithWriter(w),
	}
}

// Start initialises and displays a progress bar with the given title and total.
func (p *Progress) Start(title string, total int) {
	bar, err := pterm.DefaultProgressbar.
		WithTitle(title).
		WithTotal(total).
		WithWriter(p.writer).
		WithShowCount(true).
		WithShowPercentage(true).
		Start()
	if err != nil {
		p.errPrinter.Println("failed to start progress bar: " + err.Error())
		return
	}
	p.bar = bar
}

// Update sets the progress bar to current and updates the title.
func (p *Progress) Update(current int, message string) {
	if p.bar == nil {
		return
	}
	p.bar.UpdateTitle(message)
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

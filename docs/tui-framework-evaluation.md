# TUI Framework Evaluation for Lumen

## Context

Lumen currently has **no TUI framework** — all terminal output uses plain
`fmt.Fprintf` to stderr/stdout (e.g., `[n/total] filename` progress, completion
summaries). The project uses Cobra for CLI parsing. A TUI framework would
improve the developer experience for CLI commands like `lumen index`, adding
spinners, progress bars, colored output, and potentially interactive features.

**Critical constraint:** MCP stdio transport uses stdout for JSON — any TUI
framework must render to **stderr only** during MCP mode, or be disabled
entirely when running as an MCP server.

---

## Recommendation 1: Charm Stack (Bubble Tea + Lip Gloss + Bubbles)

**What it is:** An Elm-architecture TUI framework. You define a Model (state),
Update (handle messages), and View (render to string). Bubbles provides
pre-built components (spinners, progress bars, text inputs). Lip Gloss handles
styling (colors, borders, padding).

| Attribute    | Detail                                                    |
| ------------ | --------------------------------------------------------- |
| GitHub Stars | ~29k (BubbleTea), ~9k (Lip Gloss), ~6k (Bubbles)         |
| Maturity     | v2.x (2026), production-ready, largest Go TUI ecosystem   |
| License      | MIT                                                       |
| Go module    | `github.com/charmbracelet/bubbletea`                      |

**Pros:**

- Richest ecosystem — spinners, progress bars, tables, file pickers all
  available as Bubbles components
- Full-screen or inline rendering modes (inline fits CLI tools well)
- Excellent community, documentation, and examples
- Can output to any `io.Writer` (stderr) — solves the MCP stdout conflict
- Active development with v2 performance improvements
- Battle-tested in production tools (Glow, Soft Serve, Crush)

**Cons:**

- Heaviest option — pulls in the full Elm runtime loop even for simple progress
  output
- Learning curve: the Elm architecture is unfamiliar to most Go developers
- Anti-pattern to use goroutines directly; must use Commands/Messages
- Overkill if you only need progress bars and spinners (not interactive TUI)
- Three separate packages to import (bubbletea + bubbles + lipgloss)

**Integration with Lumen:**

- Replace `cmd/index.go` progress callback with a BubbleTea program using
  `tea.WithOutput(os.Stderr)`
- Use `progress.Model` from Bubbles for indexing progress bar
- Use `spinner.Model` for embedding generation phases
- Disable entirely in `cmd/stdio.go` (MCP mode) — keep plain text there

**Indexing progress would look like:**

```
 Indexing codebase...
 ████████████████░░░░░░░░ 67% (134/200 files)
 ⣾ Generating embeddings for chunker/parser.go
```

---

## Recommendation 2: PTerm

**What it is:** A lightweight Go library for beautifying console output. Not a
full TUI framework — it's a collection of output printers (spinners, progress
bars, tables, colored text, tree views, panels) that write directly to a writer.
No event loop, no Elm architecture.

| Attribute    | Detail                             |
| ------------ | ---------------------------------- |
| GitHub Stars | ~5k                                |
| Maturity     | v0.12.x, stable API, widely used   |
| License      | MIT                                |
| Go module    | `github.com/pterm/pterm`           |

**Pros:**

- **Simplest integration** — drop-in replacement for `fmt.Fprintf` calls, no
  architecture change
- No event loop or message passing — just call
  `pterm.DefaultSpinner.Start()` / `.Stop()`
- Built-in progress bar, spinner, table, tree, and section printers
- Configurable writer (set to stderr to avoid MCP conflict)
- Minimal learning curve — feels like enhanced `fmt`
- Testable output with `pterm.DisableColor()` and output capture
- Zero dependencies beyond Go stdlib

**Cons:**

- Less polished rendering than Charm stack (no Lip Gloss-level styling)
- Not suitable for full interactive TUIs (no input handling, no cursor
  management)
- Smaller ecosystem and community than Charm
- Some components use ANSI codes that may not work in all terminals
- API is v0.x (though stable in practice)

**Integration with Lumen:**

- Replace `fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", ...)` with
  `pterm.DefaultProgressbar`
- Add `pterm.DefaultSpinner` for embedding generation
- Use `pterm.DefaultTable` for completion summary statistics
- Set `pterm.SetDefaultOutput(os.Stderr)` globally
- No architectural changes needed — works alongside existing Cobra commands

**Indexing progress would look like:**

```
 ✓ Scanning files...
 Indexing:  [████████████████░░░░░░░░] 134/200
 ◓ Embedding chunker/parser.go...
```

---

## Recommendation 3: Lip Gloss Only (Styling Without the TUI Runtime)

**What it is:** Lip Gloss is the styling layer from the Charm stack, usable
**standalone** without BubbleTea. It provides ANSI color, borders, padding,
alignment, and layout — essentially CSS for the terminal. Combined with Go's
`fmt`, it gives you styled output without an event loop.

| Attribute    | Detail                                    |
| ------------ | ----------------------------------------- |
| GitHub Stars | ~9k                                       |
| Maturity     | v1.x, stable, widely adopted independently |
| License      | MIT                                       |
| Go module    | `github.com/charmbracelet/lipgloss`       |

**Pros:**

- **Lightest Charm option** — styled output without BubbleTea's runtime
- Familiar `fmt.Fprintf` pattern — just wrap strings in styles
- Adaptive color (auto-detects terminal capabilities, degrades gracefully)
- Can add a manual spinner/progress bar with a few lines of code
- Same styling quality as full Charm apps
- Pairs well with Cobra (no architectural conflict)
- Can upgrade to full BubbleTea later if needed

**Cons:**

- No built-in components (no spinner, no progress bar — you build your own or
  add Bubbles)
- Manual terminal cursor management if you want updating progress
- Not a framework — more of a styling utility
- Still need to handle concurrent output yourself (e.g., progress + log
  messages)

**Integration with Lumen:**

- Define styles for progress, success, error, and info messages
- Replace plain `fmt.Fprintf` with styled equivalents
- Write a small progress renderer (~30 lines) that uses `\r` for in-place
  updates
- All output to stderr via `lipgloss.NewRenderer(os.Stderr)`

**Indexing progress would look like:**

```
 Indexing   134/200 files  ██████████████░░░░░░  67%
 ✓ Done. Indexed 200 files, 1,847 chunks in 3.2s
```

---

## Comparison Matrix

| Criteria               | Charm (BubbleTea) | PTerm       | Lip Gloss Only    |
| ---------------------- | ----------------- | ----------- | ----------------- |
| Complexity             | High              | Low         | Low               |
| Learning curve         | Steep (Elm arch)  | Minimal     | Minimal           |
| Built-in components    | Many (Bubbles)    | Many        | None              |
| Styling quality        | Excellent         | Good        | Excellent         |
| Interactive features   | Full TUI          | None        | None              |
| Dependency weight      | Heavy (3 pkgs)    | Light (1)   | Light (1)         |
| Future extensibility   | Full TUI apps     | Output only | Upgrade to BubbleTea |
| MCP compatibility      | Configurable writer | Configurable writer | Configurable writer |

## Overall Recommendation

**For Lumen specifically, PTerm is the best fit.** Rationale:

- Lumen is a plugin, not a standalone TUI app — it needs pretty output, not
  interactive UI
- PTerm requires zero architectural changes (no Elm model/update/view)
- Drop-in replacement for existing `fmt.Fprintf` progress output
- Built-in progress bar, spinner, and table cover all current needs
- Lightest integration effort with Cobra

**Runner-up: Lip Gloss Only** — if you want Charm-quality styling but prefer to
stay closer to raw `fmt` with the option to upgrade to full BubbleTea later.

**BubbleTea is overkill** unless Lumen plans to add interactive features (search
result browsing, config wizards, etc.).

## Sources

- [BubbleTea](https://github.com/charmbracelet/bubbletea)
- [PTerm](https://github.com/pterm/pterm)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [TUI Framework Rankings](https://ossinsight.io/collections/tui-framework)

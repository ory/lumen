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

package cmd

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ory/lumen/internal/config"
)

// newDebugLogger opens (or creates) the lumen debug log file and returns a
// JSON slog.Logger that appends to it. The caller must close the returned
// io.Closer when the process exits. If the log directory cannot be created
// the logger writes to os.Stderr and returns a nil Closer.
func newDebugLogger() (*slog.Logger, io.Closer) {
	logDir := filepath.Join(config.XDGDataDir(), "lumen")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return slog.New(slog.NewJSONHandler(os.Stderr, nil)), nil
	}

	f, err := os.OpenFile(filepath.Join(logDir, "debug.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return slog.New(slog.NewJSONHandler(os.Stderr, nil)), nil
	}

	logger := slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv("LUMEN_LOG_LEVEL")),
	}))
	return logger, f
}

// parseLogLevel maps a LUMEN_LOG_LEVEL string to a slog.Level.
// Accepts "debug", "info", "warn", "error" (case-insensitive).
// Defaults to slog.LevelInfo for unrecognised or empty values.
func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

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
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ory/lumen/internal/config"
)

// newDebugLogger opens (or creates) the lumen debug log file and returns a
// JSON slog.Logger that appends to it. The file is written to the lumen data
// directory alongside the index databases so it is easy to find when
// debugging re-index behaviour.
//
// The caller is responsible for closing the returned *os.File when the
// process exits. If the log file cannot be opened the logger writes to
// os.Stderr instead, so callers never need to check for errors.
func newDebugLogger() (*slog.Logger, *os.File) {
	logDir := filepath.Join(config.XDGDataDir(), "lumen")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return slog.New(slog.NewJSONHandler(os.Stderr, nil)), nil
	}

	logPath := filepath.Join(logDir, "debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return slog.New(slog.NewJSONHandler(os.Stderr, nil)), nil
	}

	logger := slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv("LUMEN_LOG_LEVEL")),
	}))
	logger.Info("lumen started", "log_file", logPath, "time", time.Now().UTC().Format(time.RFC3339))
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

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

// Package config loads and validates runtime configuration from environment variables.
package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ory/lumen/internal/git"
	"github.com/ory/lumen/internal/models"
)

const (
	// BackendOllama is the backend identifier for Ollama.
	BackendOllama = "ollama"
	// BackendLMStudio is the backend identifier for LM Studio.
	BackendLMStudio = "lmstudio"
)

// DBPathForProject returns the default int8/512-token collection path. New
// runtime code with non-default settings should use DBPathForProjectProfile.
func DBPathForProject(projectPath, model string) string {
	return DBPathForProjectBase(XDGDataDir(), projectPath, model)
}

// DBPathForProjectBase returns the SQLite database path for a given project
// using an explicit data directory instead of reading XDG_DATA_HOME from the
// environment. Safe to call from parallel goroutines.
func DBPathForProjectBase(dataDir, projectPath, model string) string {
	dimensions, _ := ModelDimensions(model)
	return DBPathForProjectProfileBase(dataDir, projectPath, model, dimensions, "int8", 512)
}

// ModelDimensions resolves dimensions for a model in the built-in registry.
func ModelDimensions(model string) (int, bool) {
	canonical := model
	if resolved, ok := models.ModelAliases[model]; ok {
		canonical = resolved
	}
	spec, ok := models.KnownModels[canonical]
	return spec.Dims, ok
}

// DBPathForProjectProfile returns the repository-scoped collection path for an
// exact embedding and chunking profile. Git worktrees sharing a common Git
// directory resolve to the same collection; non-Git projects remain private.
func DBPathForProjectProfile(projectPath, model string, dimensions int, vectorStorage string, maxChunkTokens int) string {
	return DBPathForProjectProfileBase(XDGDataDir(), projectPath, model, dimensions, vectorStorage, maxChunkTokens)
}

// DBPathForProjectProfileBase is DBPathForProjectProfile with an explicit data
// directory, primarily for tests and callers that must not read process-wide
// environment state.
func DBPathForProjectProfileBase(dataDir, projectPath, model string, dimensions int, vectorStorage string, maxChunkTokens int) string {
	identity := projectPath
	scope := "."
	if abs, err := filepath.Abs(projectPath); err == nil {
		identity = filepath.Clean(abs)
	}
	if commonDir, err := git.CommonDir(projectPath); err == nil {
		identity = commonDir
		// Indexing currently normalizes Git projects to the worktree root. Keep
		// scope explicit in the key so subdirectory collections can be added
		// without another on-disk format change.
		scope = "."
	} else if resolved, resolveErr := filepath.EvalSymlinks(identity); resolveErr == nil {
		identity = filepath.Clean(resolved)
	}
	profile := identity + "\x00" + scope + "\x00" + model + "\x00" +
		strconv.Itoa(dimensions) + "\x00" + vectorStorage + "\x00" +
		strconv.Itoa(maxChunkTokens) + "\x00" + IndexVersion
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(profile)))
	return filepath.Join(dataDir, "lumen", hash[:16], "index.db")
}

// LegacyDBPathForProject returns the final one-database-per-worktree path used
// before shared collections. It exists solely for lazy migration.
func LegacyDBPathForProject(projectPath, model string) string {
	return LegacyDBPathForProjectBase(XDGDataDir(), projectPath, model)
}

// LegacyDBPathForProjectBase is LegacyDBPathForProject with an explicit data
// directory.
func LegacyDBPathForProjectBase(dataDir, projectPath, model string) string {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(projectPath+"\x00"+model+"\x00"+"3")))
	return filepath.Join(dataDir, "lumen", hash[:16], "index.db")
}

// XDGDataDir returns the XDG data home directory, defaulting to
// ~/.local/share if XDG_DATA_HOME is not set.
func XDGDataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

// XDGConfigDir returns XDG_CONFIG_HOME or ~/.config.
func XDGConfigDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	return filepath.Join(XDGConfigDir(), "lumen", "config.yaml")
}

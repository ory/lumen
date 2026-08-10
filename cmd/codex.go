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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const codexSkillsSourceMarker = ".lumen-skills-source"

type codexPaths struct {
	codexHome   string
	pluginRoot  string
	launcher    string
	hookCommand string
	hooksPath   string
	configPath  string
	skillsSrc   string
	skillsDst   string
}

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type realCommandRunner struct{}

func (realCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func init() {
	rootCmd.AddCommand(codexCmd)
	codexCmd.AddCommand(codexInstallCmd)
	codexCmd.AddCommand(codexDoctorCmd)
}

var codexCmd = &cobra.Command{
	Use:   "codex",
	Short: "Manage Codex integration",
}

var codexInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install or repair Codex MCP, skills, and hooks",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCodexInstall(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), realCommandRunner{})
	},
}

var codexDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Report Codex integration status",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCodexDoctor(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), realCommandRunner{})
	},
}

func resolveCodexPaths() (codexPaths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return codexPaths{}, fmt.Errorf("resolve home directory: %w", err)
	}
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}
	pluginRoot, err := resolveLumenPluginRoot()
	if err != nil {
		return codexPaths{}, err
	}
	launcher := codexLauncherPath(pluginRoot)
	return codexPaths{
		codexHome:   codexHome,
		pluginRoot:  pluginRoot,
		launcher:    launcher,
		hookCommand: codexSessionStartCommand(launcher),
		hooksPath:   filepath.Join(codexHome, "hooks.json"),
		configPath:  filepath.Join(codexHome, "config.toml"),
		skillsSrc:   filepath.Join(pluginRoot, "skills"),
		skillsDst:   filepath.Join(home, ".agents", "skills", "lumen"),
	}, nil
}

func resolveLumenPluginRoot() (string, error) {
	for _, name := range []string{"LUMEN_PLUGIN_ROOT", "CLAUDE_PLUGIN_ROOT", "CURSOR_PLUGIN_ROOT"} {
		if value := os.Getenv(name); value != "" {
			return filepath.Abs(value)
		}
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		if filepath.Base(exeDir) == "bin" {
			root := filepath.Dir(exeDir)
			if hasLumenLauncher(root) && fileExists(filepath.Join(root, "skills")) {
				return root, nil
			}
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if hasLumenLauncher(cwd) && fileExists(filepath.Join(cwd, "skills")) {
			return cwd, nil
		}
	}
	return "", fmt.Errorf("could not resolve Lumen plugin root")
}

func hasLumenLauncher(root string) bool {
	return fileExists(filepath.Join(root, "scripts", "run")) ||
		fileExists(filepath.Join(root, "scripts", "run.cmd"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func installCodexHookFile(path, command string) (bool, error) {
	var raw []byte
	if existing, err := os.ReadFile(path); err == nil {
		raw = existing
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("read hooks file: %w", err)
	}

	merged, changed, err := mergeCodexSessionStartHook(raw, command)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	targetPath := path
	if linkTarget, err := os.Readlink(path); err == nil {
		if filepath.IsAbs(linkTarget) {
			targetPath = linkTarget
		} else {
			targetPath = filepath.Join(filepath.Dir(path), linkTarget)
		}
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return false, fmt.Errorf("create hooks directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".hooks-*.json")
	if err != nil {
		return false, fmt.Errorf("create temp hooks file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(merged); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return false, fmt.Errorf("write temp hooks file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return false, fmt.Errorf("close temp hooks file: %w", err)
	}
	if err := os.Rename(tmpName, targetPath); err != nil {
		_ = os.Remove(tmpName)
		return false, fmt.Errorf("replace hooks file: %w", err)
	}
	return true, nil
}

func ensureCodexHooksFeatureFlag(path string) (bool, error) {
	var raw []byte
	if existing, err := os.ReadFile(path); err == nil {
		raw = existing
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("read Codex config: %w", err)
	}

	merged, changed, err := mergeCodexHooksFeatureFlag(raw)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	return writeFilePreservingSymlink(path, merged, ".config-*.toml")
}

func codexHooksFeatureEnabled(path string) (bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return codexHooksFeatureEnabledInConfig(raw)
}

func writeFilePreservingSymlink(path string, data []byte, pattern string) (bool, error) {
	targetPath := path
	if linkTarget, err := os.Readlink(path); err == nil {
		if filepath.IsAbs(linkTarget) {
			targetPath = linkTarget
		} else {
			targetPath = filepath.Join(filepath.Dir(path), linkTarget)
		}
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), pattern)
	if err != nil {
		return false, fmt.Errorf("create temp config file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return false, fmt.Errorf("write temp config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return false, fmt.Errorf("close temp config file: %w", err)
	}
	if err := os.Rename(tmpName, targetPath); err != nil {
		_ = os.Remove(tmpName)
		return false, fmt.Errorf("replace config file: %w", err)
	}
	return true, nil
}

func runCodexInstall(ctx context.Context, stdout, stderr io.Writer, runner commandRunner) error {
	paths, err := resolveCodexPaths()
	if err != nil {
		return err
	}
	featureChanged, err := ensureCodexHooksFeatureFlag(paths.configPath)
	if err != nil {
		return err
	}
	if featureChanged {
		_, _ = fmt.Fprintf(stdout, "Enabled Codex hooks feature in %s\n", paths.configPath)
	} else {
		_, _ = fmt.Fprintf(stdout, "Codex hooks feature already enabled in %s\n", paths.configPath)
	}
	changed, err := installCodexHookFile(paths.hooksPath, paths.hookCommand)
	if err != nil {
		return err
	}
	if changed {
		_, _ = fmt.Fprintf(stdout, "Installed Codex SessionStart hook at %s\n", paths.hooksPath)
	} else {
		_, _ = fmt.Fprintf(stdout, "Codex SessionStart hook already installed at %s\n", paths.hooksPath)
	}
	if err := ensureCodexSkills(paths, stderr); err != nil {
		return err
	}
	if err := ensureCodexMCP(ctx, paths, runner, stdout, stderr); err != nil {
		return err
	}
	return nil
}

func ensureCodexSkills(paths codexPaths, _ io.Writer) error {
	if !fileExists(paths.skillsSrc) {
		return fmt.Errorf("skills source missing: %s", paths.skillsSrc)
	}
	if info, err := os.Lstat(paths.skillsDst); err == nil {
		if skillsPathMatches(paths.skillsDst, paths.skillsSrc, info) {
			return nil
		}
		if err := removeExistingSkillsPath(paths.skillsDst, info); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(paths.skillsDst), 0o755); err != nil {
		return fmt.Errorf("create skills directory: %w", err)
	}
	if err := os.Symlink(paths.skillsSrc, paths.skillsDst); err != nil {
		if copyErr := copyCodexSkills(paths.skillsSrc, paths.skillsDst); copyErr != nil {
			return fmt.Errorf("create skills symlink: %w; copy fallback: %v", err, copyErr)
		}
	}
	return nil
}

func ensureCodexMCP(ctx context.Context, paths codexPaths, runner commandRunner, stdout, _ io.Writer) error {
	out, err := runner.Run(ctx, "codex", "mcp", "get", "lumen")
	if err == nil && codexMCPOutputMatches(out, paths) {
		_, _ = fmt.Fprintln(stdout, "Codex MCP server lumen already configured")
		return nil
	}
	if err == nil {
		if _, removeErr := runner.Run(ctx, "codex", "mcp", "remove", "lumen"); removeErr != nil {
			return fmt.Errorf("remove mismatched Codex MCP server: %w", removeErr)
		}
		if _, addErr := runner.Run(ctx, "codex", "mcp", "add", "lumen", "--", paths.launcher, "stdio"); addErr != nil {
			return fmt.Errorf("add Codex MCP server: %w", addErr)
		}
		_, _ = fmt.Fprintln(stdout, "Updated Codex MCP server lumen")
		return nil
	}
	if _, addErr := runner.Run(ctx, "codex", "mcp", "add", "lumen", "--", paths.launcher, "stdio"); addErr != nil {
		return fmt.Errorf("add Codex MCP server: %w", addErr)
	}
	_, _ = fmt.Fprintln(stdout, "Added Codex MCP server lumen")
	return nil
}

func codexMCPOutputMatches(out []byte, paths codexPaths) bool {
	command, args := parseCodexMCPOutput(out)
	launcher := strings.ReplaceAll(paths.launcher, `\`, "/")
	return command == launcher && len(args) == 1 && args[0] == "stdio"
}

func runCodexDoctor(ctx context.Context, stdout, _ io.Writer, runner commandRunner) error {
	paths, err := resolveCodexPaths()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Codex home: %s\n", paths.codexHome)
	_, _ = fmt.Fprintf(stdout, "Lumen root: %s\n", paths.pluginRoot)

	out, err := runner.Run(ctx, "codex", "mcp", "get", "lumen")
	switch {
	case err != nil && errors.Is(err, exec.ErrNotFound):
		_, _ = fmt.Fprintln(stdout, "MCP lumen: unavailable: codex command not found")
	case err != nil && codexMCPGetOutputReportsMissing(out):
		_, _ = fmt.Fprintln(stdout, "MCP lumen: missing")
	case err != nil:
		if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
			_, _ = fmt.Fprintf(stdout, "MCP lumen: error: %v: %s\n", err, oneLine(trimmed))
		} else {
			_, _ = fmt.Fprintf(stdout, "MCP lumen: error: %v\n", err)
		}
	case codexMCPOutputMatches(out, paths):
		_, _ = fmt.Fprintln(stdout, "MCP lumen: ok")
	default:
		_, _ = fmt.Fprintln(stdout, "MCP lumen: mismatch")
	}

	featureEnabled, featureErr := codexHooksFeatureEnabled(paths.configPath)
	switch {
	case featureErr != nil:
		_, _ = fmt.Fprintf(stdout, "Codex hooks feature: error: %v\n", featureErr)
	case featureEnabled:
		_, _ = fmt.Fprintln(stdout, "Codex hooks feature: ok")
	default:
		_, _ = fmt.Fprintln(stdout, "Codex hooks feature: disabled")
	}

	hookStatus, hookErr := codexHookStatus(paths.hooksPath, paths.hookCommand)
	switch {
	case hookErr != nil:
		_, _ = fmt.Fprintf(stdout, "SessionStart hook: error: %v\n", hookErr)
	case hookStatus.exact == 1 && hookStatus.total == 1 && featureEnabled:
		_, _ = fmt.Fprintln(stdout, "SessionStart hook: ok")
	case hookStatus.exact == 1 && hookStatus.total == 1:
		_, _ = fmt.Fprintln(stdout, "SessionStart hook: installed but disabled")
	case hookStatus.total > 1:
		_, _ = fmt.Fprintf(stdout, "SessionStart hook: duplicate (%d)\n", hookStatus.total)
	case hookStatus.total == 1:
		_, _ = fmt.Fprintln(stdout, "SessionStart hook: mismatch")
	default:
		_, _ = fmt.Fprintln(stdout, "SessionStart hook: missing")
	}

	if skillsLinkOK(paths) {
		_, _ = fmt.Fprintln(stdout, "Skills: ok")
	} else {
		_, _ = fmt.Fprintln(stdout, "Skills: missing")
	}
	return nil
}

type codexHookStatusResult struct {
	exact int
	total int
}

func codexHookStatus(path, expectedCommand string) (codexHookStatusResult, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return codexHookStatusResult{}, nil
	}
	if err != nil {
		return codexHookStatusResult{}, err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return codexHookStatusResult{}, err
	}
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		if _, exists := doc["hooks"]; exists {
			return codexHookStatusResult{}, fmt.Errorf("codex hooks document field %q must be an object", "hooks")
		}
		return codexHookStatusResult{}, nil
	}
	rawGroups, ok := hooks["SessionStart"].([]any)
	if !ok {
		if _, exists := hooks["SessionStart"]; exists {
			return codexHookStatusResult{}, fmt.Errorf("codex hooks document field %q must be an array", "hooks.SessionStart")
		}
		return codexHookStatusResult{}, nil
	}
	status := codexHookStatusResult{}
	for _, rawGroup := range rawGroups {
		group, ok := rawGroup.(map[string]any)
		if !ok {
			return codexHookStatusResult{}, fmt.Errorf("codex hooks document field %q must contain objects", "hooks.SessionStart")
		}
		groupStatus, err := countLumenCodexHooks(group, expectedCommand)
		if err != nil {
			return codexHookStatusResult{}, err
		}
		status.exact += groupStatus.exact
		status.total += groupStatus.total
	}
	return status, nil
}

func countLumenCodexHooks(group map[string]any, expectedCommand string) (codexHookStatusResult, error) {
	hooks, ok := group["hooks"].([]any)
	if !ok {
		return codexHookStatusResult{}, fmt.Errorf("codex hooks document field %q must contain hook arrays", "hooks.SessionStart")
	}
	status := codexHookStatusResult{}
	for _, rawHook := range hooks {
		hook, ok := rawHook.(map[string]any)
		if !ok {
			return codexHookStatusResult{}, fmt.Errorf("codex hooks document field %q must contain hook objects", "hooks.SessionStart.hooks")
		}
		command, _ := hook["command"].(string)
		if command != "" && isOwnedLumenCodexSessionStartCommand(command, hook, expectedCommand) {
			status.total++
			if expectedCommand != "" && command == expectedCommand {
				status.exact++
			}
		}
	}
	return status, nil
}

func codexMCPGetOutputReportsMissing(out []byte) bool {
	lower := strings.ToLower(string(out))
	return strings.Contains(lower, "no mcp server named")
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func skillsLinkOK(paths codexPaths) bool {
	info, err := os.Lstat(paths.skillsDst)
	if err != nil {
		return false
	}
	return skillsPathMatches(paths.skillsDst, paths.skillsSrc, info)
}

func skillsPathMatches(path, want string, info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink == 0 {
		return sameFileTree(path, want) || copiedSkillsMarkerMatches(path, want)
	}
	target, err := os.Readlink(path)
	if err != nil {
		return false
	}
	return target == want || sameFileTree(resolveRelativeLink(path, target), want)
}

func resolveRelativeLink(path, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), target))
}

func sameFileTree(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	realA, errA := filepath.EvalSymlinks(absA)
	realB, errB := filepath.EvalSymlinks(absB)
	if errA != nil || errB != nil {
		return absA == absB
	}
	return realA == realB
}

func removeExistingSkillsPath(path string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return fmt.Errorf("read existing skills link: %w", err)
		}
		targetPath := resolveRelativeLink(path, target)
		if !lumenSkillsDirLooksOwned(targetPath) && !danglingLumenSkillsLinkTarget(targetPath) {
			return fmt.Errorf("%s exists and is not an installer-managed Lumen skills link", path)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale skills link: %w", err)
		}
		return nil
	}

	if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		if copiedSkillsMarkerPresent(path) {
			return os.RemoveAll(path)
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("read existing skills directory: %w", err)
		}
		if len(entries) != 0 {
			return fmt.Errorf("%s exists and is not the Lumen skills directory", path)
		}
		return os.Remove(path)
	}
	return fmt.Errorf("%s exists and is not an installer-managed Lumen skills path", path)
}

func danglingLumenSkillsLinkTarget(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return false
	} else if !errors.Is(err, os.ErrNotExist) {
		return false
	}
	cleaned := filepath.Clean(path)
	return filepath.Base(cleaned) == "skills" && filepath.Base(filepath.Dir(cleaned)) == "lumen"
}

func copyCodexSkills(src, dst string) error {
	if err := filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if rel == codexSkillsSourceMarker {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, codexSkillsSourceMarker), []byte(src+"\n"), 0o644)
}

func copiedSkillsMarkerMatches(path, want string) bool {
	raw, err := os.ReadFile(filepath.Join(path, codexSkillsSourceMarker))
	return err == nil &&
		strings.TrimSpace(string(raw)) == want &&
		lumenSkillsDirLooksOwned(path)
}

func copiedSkillsMarkerPresent(path string) bool {
	raw, err := os.ReadFile(filepath.Join(path, codexSkillsSourceMarker))
	return err == nil && strings.TrimSpace(string(raw)) != "" && lumenSkillsDirLooksOwned(path)
}

func lumenSkillsDirLooksOwned(path string) bool {
	return fileContains(filepath.Join(path, "doctor", "SKILL.md"), "# Lumen Doctor") &&
		fileContains(filepath.Join(path, "reindex", "SKILL.md"), "# Lumen Reindex")
}

func fileContains(path, want string) bool {
	raw, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(raw), want)
}

func parseCodexMCPOutput(out []byte) (string, []string) {
	var parsed struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(out, &parsed); err == nil && parsed.Command != "" {
		return strings.ReplaceAll(parsed.Command, `\`, "/"), parsed.Args
	}

	var command string
	var args []string
	for _, line := range strings.Split(string(out), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "command":
			command = strings.ReplaceAll(strings.Trim(strings.TrimSpace(value), `"`), `\`, "/")
		case "args":
			args = parseCodexMCPArgs(strings.TrimSpace(value))
		}
	}
	return command, args
}

func parseCodexMCPArgs(value string) []string {
	if value == "" {
		return nil
	}
	var parsed []string
	if strings.HasPrefix(value, "[") && json.Unmarshal([]byte(value), &parsed) == nil {
		return parsed
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil
	}
	return fields
}

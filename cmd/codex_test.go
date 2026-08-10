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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls     []string
	getOutput []byte
	getErr    error
	runErrs   map[string]error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call)
	if f.runErrs != nil {
		if err := f.runErrs[call]; err != nil {
			return nil, err
		}
	}
	if call == "codex mcp get lumen" {
		return f.getOutput, f.getErr
	}
	return []byte("ok\n"), nil
}

func TestCodexPathsFromEnv(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	codexHome := filepath.Join(home, "codex-home")
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)

	paths, err := resolveCodexPaths()
	if err != nil {
		t.Fatalf("resolveCodexPaths: %v", err)
	}

	if paths.codexHome != codexHome {
		t.Fatalf("codexHome = %q, want %q", paths.codexHome, codexHome)
	}
	if paths.pluginRoot != pluginRoot {
		t.Fatalf("pluginRoot = %q, want %q", paths.pluginRoot, pluginRoot)
	}
	if paths.launcher != filepath.Join(pluginRoot, "scripts", "run") {
		t.Fatalf("launcher = %q, want run under plugin root", paths.launcher)
	}
	if paths.hooksPath != filepath.Join(codexHome, "hooks.json") {
		t.Fatalf("hooksPath = %q, want hooks.json under CODEX_HOME", paths.hooksPath)
	}
	if paths.configPath != filepath.Join(codexHome, "config.toml") {
		t.Fatalf("configPath = %q, want config.toml under CODEX_HOME", paths.configPath)
	}
	if paths.skillsSrc != filepath.Join(pluginRoot, "skills") {
		t.Fatalf("skillsSrc = %q, want skills under plugin root", paths.skillsSrc)
	}
	if paths.skillsDst != filepath.Join(home, ".agents", "skills", "lumen") {
		t.Fatalf("skillsDst = %q, want ~/.agents/skills/lumen", paths.skillsDst)
	}
	if !strings.Contains(paths.hookCommand, " hook session-start lumen --host codex") {
		t.Fatalf("hookCommand = %q, want Lumen session-start command", paths.hookCommand)
	}
}

func TestInstallCodexHookFile_Idempotent(t *testing.T) {
	hooksPath := filepath.Join(t.TempDir(), "hooks.json")
	command := codexSessionStartCommand("/repo/lumen/scripts/run")

	changed, err := installCodexHookFile(hooksPath, command)
	if err != nil {
		t.Fatalf("first installCodexHookFile: %v", err)
	}
	if !changed {
		t.Fatal("first changed = false, want true")
	}
	first, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read first hooks file: %v", err)
	}

	changed, err = installCodexHookFile(hooksPath, command)
	if err != nil {
		t.Fatalf("second installCodexHookFile: %v", err)
	}
	if changed {
		t.Fatal("second changed = true, want false")
	}
	second, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read second hooks file: %v", err)
	}
	if string(second) != string(first) {
		t.Fatalf("hooks file changed on idempotent install:\n%s\nwant:\n%s", second, first)
	}
}

func TestInstallCodexHookFile_MalformedJSONLeavesFileUntouched(t *testing.T) {
	hooksPath := filepath.Join(t.TempDir(), "hooks.json")
	original := []byte(`{"hooks":`)
	if err := os.WriteFile(hooksPath, original, 0o644); err != nil {
		t.Fatalf("write malformed hooks file: %v", err)
	}

	changed, err := installCodexHookFile(hooksPath, "command")
	if err == nil {
		t.Fatal("installCodexHookFile error = nil, want error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	got, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read hooks file: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("hooks file = %q, want original %q", got, original)
	}
}

func TestRunCodexInstall_WritesHookAndAddsMCPWhenMissing(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	runner := &fakeRunner{getErr: exec.ErrNotFound}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	if err := runCodexInstall(context.Background(), stdout, stderr, runner); err != nil {
		t.Fatalf("runCodexInstall: %v", err)
	}

	hooksPath := filepath.Join(home, ".codex", "hooks.json")
	raw, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read hooks file: %v", err)
	}
	if !strings.Contains(string(raw), "hook session-start lumen --host codex") {
		t.Fatalf("hooks file missing session-start command:\n%s", raw)
	}
	config, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if !strings.Contains(string(config), "[features]\n") || !strings.Contains(string(config), "hooks = true") {
		t.Fatalf("config file missing hooks feature flag:\n%s", config)
	}
	if strings.Contains(string(config), "codex_hooks") {
		t.Fatalf("config file contains deprecated codex_hooks feature flag:\n%s", config)
	}
	if !isSymlinkTo(t, filepath.Join(home, ".agents", "skills", "lumen"), filepath.Join(pluginRoot, "skills")) {
		t.Fatalf("skills link was not installed")
	}
	wantAdd := fmt.Sprintf("codex mcp add lumen -- %s stdio", filepath.Join(pluginRoot, "scripts", "run"))
	if !containsCall(runner.calls, "codex mcp get lumen") || !containsCall(runner.calls, wantAdd) {
		t.Fatalf("calls = %#v, want get and add", runner.calls)
	}
	if !strings.Contains(stdout.String(), "Installed Codex SessionStart hook") {
		t.Fatalf("stdout = %q, want installed hook message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunCodexInstall_IdempotentEndToEnd(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	runner := &fakeRunner{getErr: exec.ErrNotFound}

	if err := runCodexInstall(context.Background(), new(bytes.Buffer), new(bytes.Buffer), runner); err != nil {
		t.Fatalf("first runCodexInstall: %v", err)
	}
	paths, err := resolveCodexPaths()
	if err != nil {
		t.Fatalf("resolveCodexPaths: %v", err)
	}
	firstHooks, err := os.ReadFile(paths.hooksPath)
	if err != nil {
		t.Fatalf("read first hooks file: %v", err)
	}
	firstCallCount := len(runner.calls)

	runner.getErr = nil
	runner.getOutput = []byte(fmt.Sprintf("command: %s\nargs: stdio\n", paths.launcher))
	stdout := new(bytes.Buffer)
	if err := runCodexInstall(context.Background(), stdout, new(bytes.Buffer), runner); err != nil {
		t.Fatalf("second runCodexInstall: %v", err)
	}

	secondHooks, err := os.ReadFile(paths.hooksPath)
	if err != nil {
		t.Fatalf("read second hooks file: %v", err)
	}
	if string(secondHooks) != string(firstHooks) {
		t.Fatalf("hooks changed on second install:\n%s\nwant:\n%s", secondHooks, firstHooks)
	}
	if !isSymlinkTo(t, paths.skillsDst, paths.skillsSrc) {
		t.Fatalf("skills link was not preserved")
	}
	if !strings.Contains(stdout.String(), "Codex SessionStart hook already installed") ||
		!strings.Contains(stdout.String(), "Codex MCP server lumen already configured") {
		t.Fatalf("second stdout = %q, want idempotent messages", stdout.String())
	}
	for _, call := range runner.calls[firstCallCount:] {
		if strings.Contains(call, " mcp add ") || strings.Contains(call, " mcp remove ") {
			t.Fatalf("second run should not add/remove MCP, calls = %#v", runner.calls[firstCallCount:])
		}
	}
	status, err := codexHookStatus(paths.hooksPath, paths.hookCommand)
	if err != nil {
		t.Fatalf("codexHookStatus: %v", err)
	}
	if status.exact != 1 || status.total != 1 {
		t.Fatalf("hook status = %+v, want exactly one Lumen hook", status)
	}
}

func TestRunCodexInstall_FailsWhenMCPAddFails(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	addCall := fmt.Sprintf("codex mcp add lumen -- %s stdio", filepath.Join(pluginRoot, "scripts", "run"))
	runner := &fakeRunner{
		getErr:  exec.ErrNotFound,
		runErrs: map[string]error{addCall: errors.New("add failed")},
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	if err := runCodexInstall(context.Background(), stdout, stderr, runner); err == nil {
		t.Fatal("runCodexInstall error = nil, want MCP add error")
	}
	if strings.Contains(stderr.String(), "Warning: MCP setup incomplete") {
		t.Fatalf("stderr = %q, want no swallowed MCP warning", stderr.String())
	}
}

func TestRunCodexInstall_FailsWhenSkillsSetupFails(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	skillsDst := filepath.Join(home, ".agents", "skills", "lumen")
	if err := os.MkdirAll(skillsDst, 0o755); err != nil {
		t.Fatalf("create skills dst: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsDst, "custom.md"), []byte("custom"), 0o644); err != nil {
		t.Fatalf("write custom skills file: %v", err)
	}
	runner := &fakeRunner{getErr: exec.ErrNotFound}
	stderr := new(bytes.Buffer)

	if err := runCodexInstall(context.Background(), new(bytes.Buffer), stderr, runner); err == nil {
		t.Fatal("runCodexInstall error = nil, want skills setup error")
	}
	if strings.Contains(stderr.String(), "Warning: skills setup incomplete") {
		t.Fatalf("stderr = %q, want no swallowed skills warning", stderr.String())
	}
}

func TestRunCodexInstall_ReplacesMismatchedMCP(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	runner := &fakeRunner{getOutput: []byte("command: /other/lumen\nargs: stdio\n")}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	if err := runCodexInstall(context.Background(), stdout, stderr, runner); err != nil {
		t.Fatalf("runCodexInstall: %v", err)
	}

	wantRemove := "codex mcp remove lumen"
	wantAdd := fmt.Sprintf("codex mcp add lumen -- %s stdio", filepath.Join(pluginRoot, "scripts", "run"))
	if !containsCall(runner.calls, "codex mcp get lumen") || !containsCall(runner.calls, wantRemove) || !containsCall(runner.calls, wantAdd) {
		t.Fatalf("calls = %#v, want get, remove, add", runner.calls)
	}
	if !strings.Contains(stdout.String(), "Updated Codex MCP server lumen") {
		t.Fatalf("stdout = %q, want updated MCP message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestCodexMCPOutputMatchesRequiresCommandAndArgs(t *testing.T) {
	paths := codexPaths{launcher: "/repo/lumen/scripts/run"}
	if !codexMCPOutputMatches([]byte("transport: stdio\ncommand: /repo/lumen/scripts/run\nargs: stdio\n"), paths) {
		t.Fatal("expected command and args output to match")
	}
	if !codexMCPOutputMatches([]byte(`{"command":"/repo/lumen/scripts/run","args":["stdio"]}`), paths) {
		t.Fatal("expected JSON command and args output to match")
	}
	if codexMCPOutputMatches([]byte("transport: stdio\nnote: /repo/lumen/scripts/run\n"), paths) {
		t.Fatal("matched output without command field")
	}
	if codexMCPOutputMatches([]byte("command: /repo/lumen/scripts/run\nnote: stdio\n"), paths) {
		t.Fatal("matched output without args field")
	}
	if codexMCPOutputMatches([]byte("command: /repo/lumen/scripts/run.bak\nargs: stdio\n"), paths) {
		t.Fatal("matched command prefix false positive")
	}
	if codexMCPOutputMatches([]byte("command: /repo/lumen/scripts/run\nargs: stdio-extra\n"), paths) {
		t.Fatal("matched args prefix false positive")
	}
}

func TestRunCodexDoctorReportsHookStatus(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	paths, err := resolveCodexPaths()
	if err != nil {
		t.Fatalf("resolveCodexPaths: %v", err)
	}
	if _, err := installCodexHookFile(paths.hooksPath, paths.hookCommand); err != nil {
		t.Fatalf("installCodexHookFile: %v", err)
	}
	if _, err := ensureCodexHooksFeatureFlag(paths.configPath); err != nil {
		t.Fatalf("ensureCodexHooksFeatureFlag: %v", err)
	}
	runner := &fakeRunner{getOutput: []byte(fmt.Sprintf("command: %s\nargs: stdio\n", paths.launcher))}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	if err := runCodexDoctor(context.Background(), stdout, stderr, runner); err != nil {
		t.Fatalf("runCodexDoctor: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{
		"Codex home: " + paths.codexHome,
		"Lumen root: " + paths.pluginRoot,
		"MCP lumen: ok",
		"Codex hooks feature: ok",
		"SessionStart hook: ok",
		"Skills: missing",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunCodexDoctorReportsMissingStatuses(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	runner := &fakeRunner{
		getOutput: []byte("Error: No MCP server named 'lumen' found."),
		getErr:    errors.New("exit status 1"),
	}
	stdout := new(bytes.Buffer)

	if err := runCodexDoctor(context.Background(), stdout, new(bytes.Buffer), runner); err != nil {
		t.Fatalf("runCodexDoctor: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{"MCP lumen: missing", "Codex hooks feature: disabled", "SessionStart hook: missing", "Skills: missing"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
	}
}

func TestRunCodexDoctorReportsInstalledHookDisabledByFeatureFlag(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	paths, err := resolveCodexPaths()
	if err != nil {
		t.Fatalf("resolveCodexPaths: %v", err)
	}
	if _, err := installCodexHookFile(paths.hooksPath, paths.hookCommand); err != nil {
		t.Fatalf("installCodexHookFile: %v", err)
	}
	runner := &fakeRunner{getOutput: []byte(fmt.Sprintf("command: %s\nargs: stdio\n", paths.launcher))}
	stdout := new(bytes.Buffer)

	if err := runCodexDoctor(context.Background(), stdout, new(bytes.Buffer), runner); err != nil {
		t.Fatalf("runCodexDoctor: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{"Codex hooks feature: disabled", "SessionStart hook: installed but disabled"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
	}
}

func TestRunCodexDoctorReportsUnavailableMCPCommand(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	stdout := new(bytes.Buffer)

	if err := runCodexDoctor(context.Background(), stdout, new(bytes.Buffer), &fakeRunner{getErr: exec.ErrNotFound}); err != nil {
		t.Fatalf("runCodexDoctor: %v", err)
	}

	if !strings.Contains(stdout.String(), "MCP lumen: unavailable: codex command not found") {
		t.Fatalf("stdout = %q, want unavailable MCP command", stdout.String())
	}
}

func TestRunCodexDoctorReportsMismatchedMCP(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	runner := &fakeRunner{getOutput: []byte("command: /other/lumen\nargs: stdio\n")}
	stdout := new(bytes.Buffer)

	if err := runCodexDoctor(context.Background(), stdout, new(bytes.Buffer), runner); err != nil {
		t.Fatalf("runCodexDoctor: %v", err)
	}

	if !strings.Contains(stdout.String(), "MCP lumen: mismatch") {
		t.Fatalf("stdout = %q, want MCP mismatch", stdout.String())
	}
}

func TestRunCodexDoctorReportsStaleHookMismatch(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	paths, err := resolveCodexPaths()
	if err != nil {
		t.Fatalf("resolveCodexPaths: %v", err)
	}
	staleCommand := codexSessionStartCommand(filepath.Join(home, "old-lumen", "scripts", "run"))
	if _, err := installCodexHookFile(paths.hooksPath, staleCommand); err != nil {
		t.Fatalf("installCodexHookFile: %v", err)
	}
	runner := &fakeRunner{getOutput: []byte(fmt.Sprintf("command: %s\nargs: stdio\n", paths.launcher))}
	stdout := new(bytes.Buffer)

	if err := runCodexDoctor(context.Background(), stdout, new(bytes.Buffer), runner); err != nil {
		t.Fatalf("runCodexDoctor: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "SessionStart hook: mismatch") {
		t.Fatalf("stdout = %q, want stale hook mismatch", out)
	}
	if strings.Contains(out, "SessionStart hook: ok") {
		t.Fatalf("stdout = %q, did not want stale hook ok", out)
	}
}

func TestRunCodexDoctorReportsDuplicateHooksAndSkillsOK(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	paths, err := resolveCodexPaths()
	if err != nil {
		t.Fatalf("resolveCodexPaths: %v", err)
	}
	if _, err := installCodexHookFile(paths.hooksPath, paths.hookCommand); err != nil {
		t.Fatalf("installCodexHookFile first: %v", err)
	}
	raw, err := os.ReadFile(paths.hooksPath)
	if err != nil {
		t.Fatalf("read hooks file: %v", err)
	}
	duplicate := strings.Replace(string(raw), `"SessionStart": [`, `"SessionStart": [
      {
        "hooks": [
          {
            "command": "\"/repo/lumen/scripts/run.sh\" hook session-start lumen --host claude",
            "type": "command"
          }
        ],
        "matcher": "startup"
      },`, 1)
	if err := os.WriteFile(paths.hooksPath, []byte(duplicate), 0o644); err != nil {
		t.Fatalf("write duplicate hooks file: %v", err)
	}
	if err := ensureCodexSkills(paths, new(bytes.Buffer)); err != nil {
		t.Fatalf("ensureCodexSkills: %v", err)
	}
	runner := &fakeRunner{getOutput: []byte(fmt.Sprintf("command: %s\nargs: stdio\n", paths.launcher))}
	stdout := new(bytes.Buffer)

	if err := runCodexDoctor(context.Background(), stdout, new(bytes.Buffer), runner); err != nil {
		t.Fatalf("runCodexDoctor: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{"SessionStart hook: duplicate (2)", "Skills: ok"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
	}
}

func TestRunCodexDoctorReportsMalformedHooks(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("LUMEN_PLUGIN_ROOT", pluginRoot)
	paths, err := resolveCodexPaths()
	if err != nil {
		t.Fatalf("resolveCodexPaths: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.hooksPath), 0o755); err != nil {
		t.Fatalf("create codex home: %v", err)
	}
	if err := os.WriteFile(paths.hooksPath, []byte(`{"hooks":`), 0o644); err != nil {
		t.Fatalf("write malformed hooks: %v", err)
	}
	stdout := new(bytes.Buffer)

	if err := runCodexDoctor(context.Background(), stdout, new(bytes.Buffer), &fakeRunner{getErr: exec.ErrNotFound}); err != nil {
		t.Fatalf("runCodexDoctor: %v", err)
	}
	if !strings.Contains(stdout.String(), "SessionStart hook: error:") {
		t.Fatalf("stdout = %q, want hook error", stdout.String())
	}
}

func TestInstallCodexHookFile_PreservesHooksSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "actual-hooks.json")
	link := filepath.Join(dir, "hooks.json")
	if err := os.WriteFile(target, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := installCodexHookFile(link, codexSessionStartCommand("/repo/lumen/scripts/run")); err != nil {
		t.Fatalf("installCodexHookFile: %v", err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link was not preserved; info=%v err=%v", info, err)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !strings.Contains(string(raw), "hook session-start lumen") {
		t.Fatalf("target not updated:\n%s", raw)
	}
}

func TestEnsureCodexSkills_CopiedSkillsAreIdempotent(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	paths := codexPaths{
		skillsSrc: filepath.Join(pluginRoot, "skills"),
		skillsDst: filepath.Join(home, ".agents", "skills", "lumen"),
	}
	if err := copyCodexSkills(paths.skillsSrc, paths.skillsDst); err != nil {
		t.Fatalf("copyCodexSkills: %v", err)
	}
	info, err := os.Lstat(paths.skillsDst)
	if err != nil {
		t.Fatalf("lstat copied skills: %v", err)
	}
	if !skillsPathMatches(paths.skillsDst, paths.skillsSrc, info) {
		t.Fatal("copied skills should match marker")
	}
	if err := ensureCodexSkills(paths, new(bytes.Buffer)); err != nil {
		t.Fatalf("ensureCodexSkills should accept copied skills: %v", err)
	}
	if !skillsLinkOK(paths) {
		t.Fatal("skillsLinkOK should accept copied skills")
	}
}

func TestEnsureCodexSkills_RejectsMarkerOnlyCopy(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	paths := codexPaths{
		skillsSrc: filepath.Join(pluginRoot, "skills"),
		skillsDst: filepath.Join(home, ".agents", "skills", "lumen"),
	}
	if err := os.MkdirAll(paths.skillsDst, 0o755); err != nil {
		t.Fatalf("create skills dst: %v", err)
	}
	if err := os.WriteFile(filepath.Join(paths.skillsDst, codexSkillsSourceMarker), []byte(paths.skillsSrc+"\n"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	info, err := os.Lstat(paths.skillsDst)
	if err != nil {
		t.Fatalf("lstat marker-only skills: %v", err)
	}
	if skillsPathMatches(paths.skillsDst, paths.skillsSrc, info) {
		t.Fatal("marker-only skills directory should not match")
	}
	if skillsLinkOK(paths) {
		t.Fatal("skillsLinkOK should reject marker-only skills")
	}
}

func TestEnsureCodexSkills_RefusesExistingFile(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	paths := codexPaths{
		skillsSrc: filepath.Join(pluginRoot, "skills"),
		skillsDst: filepath.Join(home, ".agents", "skills", "lumen"),
	}
	if err := os.MkdirAll(filepath.Dir(paths.skillsDst), 0o755); err != nil {
		t.Fatalf("create skills parent: %v", err)
	}
	if err := os.WriteFile(paths.skillsDst, []byte("user-owned"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	if err := ensureCodexSkills(paths, new(bytes.Buffer)); err == nil {
		t.Fatal("ensureCodexSkills error = nil, want refusal for existing file")
	}
	if got, err := os.ReadFile(paths.skillsDst); err != nil || string(got) != "user-owned" {
		t.Fatalf("existing file changed: got=%q err=%v", got, err)
	}
}

func TestEnsureCodexSkills_RefusesNonLumenSymlink(t *testing.T) {
	home := t.TempDir()
	pluginRoot := makeCodexPluginRoot(t, home)
	paths := codexPaths{
		skillsSrc: filepath.Join(pluginRoot, "skills"),
		skillsDst: filepath.Join(home, ".agents", "skills", "lumen"),
	}
	customTarget := filepath.Join(home, "custom-skills")
	if err := os.MkdirAll(customTarget, 0o755); err != nil {
		t.Fatalf("create custom target: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.skillsDst), 0o755); err != nil {
		t.Fatalf("create skills parent: %v", err)
	}
	if err := os.Symlink(customTarget, paths.skillsDst); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if err := ensureCodexSkills(paths, new(bytes.Buffer)); err == nil {
		t.Fatal("ensureCodexSkills error = nil, want refusal for non-Lumen symlink")
	}
	if !isSymlinkTo(t, paths.skillsDst, customTarget) {
		t.Fatalf("non-Lumen symlink was changed")
	}
}

func TestEnsureCodexSkills_ReplacesStaleLumenSymlink(t *testing.T) {
	home := t.TempDir()
	oldPluginRoot := makeCodexPluginRoot(t, filepath.Join(home, "old"))
	newPluginRoot := makeCodexPluginRoot(t, filepath.Join(home, "new"))
	paths := codexPaths{
		skillsSrc: filepath.Join(newPluginRoot, "skills"),
		skillsDst: filepath.Join(home, ".agents", "skills", "lumen"),
	}
	if err := os.MkdirAll(filepath.Dir(paths.skillsDst), 0o755); err != nil {
		t.Fatalf("create skills parent: %v", err)
	}
	if err := os.Symlink(filepath.Join(oldPluginRoot, "skills"), paths.skillsDst); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if err := ensureCodexSkills(paths, new(bytes.Buffer)); err != nil {
		t.Fatalf("ensureCodexSkills: %v", err)
	}
	if !isSymlinkTo(t, paths.skillsDst, paths.skillsSrc) {
		t.Fatalf("stale Lumen symlink was not replaced")
	}
}

func TestEnsureCodexSkills_ReplacesDanglingLumenSymlink(t *testing.T) {
	home := t.TempDir()
	oldPluginRoot := makeCodexPluginRoot(t, filepath.Join(home, "old", "lumen"))
	newPluginRoot := makeCodexPluginRoot(t, filepath.Join(home, "new"))
	paths := codexPaths{
		skillsSrc: filepath.Join(newPluginRoot, "skills"),
		skillsDst: filepath.Join(home, ".agents", "skills", "lumen"),
	}
	oldSkills := filepath.Join(oldPluginRoot, "skills")
	if err := os.MkdirAll(filepath.Dir(paths.skillsDst), 0o755); err != nil {
		t.Fatalf("create skills parent: %v", err)
	}
	if err := os.Symlink(oldSkills, paths.skillsDst); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.RemoveAll(oldPluginRoot); err != nil {
		t.Fatalf("remove old plugin root: %v", err)
	}

	if err := ensureCodexSkills(paths, new(bytes.Buffer)); err != nil {
		t.Fatalf("ensureCodexSkills: %v", err)
	}
	if !isSymlinkTo(t, paths.skillsDst, paths.skillsSrc) {
		t.Fatalf("dangling Lumen symlink was not replaced")
	}
}

func makeCodexPluginRoot(t *testing.T, home string) string {
	t.Helper()
	pluginRoot := filepath.Join(home, "lumen")
	if err := os.MkdirAll(filepath.Join(pluginRoot, "scripts"), 0o755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginRoot, "skills"), 0o755); err != nil {
		t.Fatalf("create skills dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginRoot, "skills", "doctor"), 0o755); err != nil {
		t.Fatalf("create doctor skill dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginRoot, "skills", "reindex"), 0o755); err != nil {
		t.Fatalf("create reindex skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "scripts", "run"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write run: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "skills", "doctor", "SKILL.md"), []byte("---\nname: doctor\n---\n\n# Lumen Doctor\n"), 0o644); err != nil {
		t.Fatalf("write doctor skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "skills", "reindex", "SKILL.md"), []byte("---\nname: reindex\n---\n\n# Lumen Reindex\n"), 0o644); err != nil {
		t.Fatalf("write reindex skill: %v", err)
	}
	return pluginRoot
}

func containsCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
}

func isSymlinkTo(t *testing.T, path, wantTarget string) bool {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("read symlink: %v", err)
	}
	return target == wantTarget
}

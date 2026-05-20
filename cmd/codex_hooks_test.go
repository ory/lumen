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
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeCodexSessionStartHook_MissingDocument(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")

	out, changed, err := mergeCodexSessionStartHook(nil, command)
	if err != nil {
		t.Fatalf("mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	doc := decodeCodexHooksDocument(t, out)
	groups := codexHookGroups(t, doc, "SessionStart")
	if len(groups) != 1 {
		t.Fatalf("SessionStart groups = %d, want 1", len(groups))
	}
	if got := codexHookCommand(t, groups[0]); got != command {
		t.Fatalf("command = %q, want %q", got, command)
	}
	if strings.Contains(string(out), "compact") {
		t.Fatalf("output contains compact: %s", out)
	}
}

func TestMergeCodexSessionStartHook_EmptyDocument(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")

	out, changed, err := mergeCodexSessionStartHook([]byte("{}"), command)
	if err != nil {
		t.Fatalf("mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	doc := decodeCodexHooksDocument(t, out)
	groups := codexHookGroups(t, doc, "SessionStart")
	if len(groups) != 1 {
		t.Fatalf("SessionStart groups = %d, want 1", len(groups))
	}
	if got := groups[0]["matcher"]; got != codexSessionStartMatcher {
		t.Fatalf("matcher = %v, want %q", got, codexSessionStartMatcher)
	}
	if _, ok := groups[0]["statusMessage"]; ok {
		t.Fatalf("statusMessage should be on command hook, got group-level value %v", groups[0]["statusMessage"])
	}
	if got := codexHookStatusMessageForGroup(t, groups[0]); got != codexHookStatusMessage {
		t.Fatalf("statusMessage = %q, want %q", got, codexHookStatusMessage)
	}
}

func TestMergeCodexHooksFeatureFlag_MissingDocument(t *testing.T) {
	out, changed, err := mergeCodexHooksFeatureFlag(nil)
	if err != nil {
		t.Fatalf("mergeCodexHooksFeatureFlag: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got, want := string(out), "[features]\nhooks = true\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestMergeCodexHooksFeatureFlag_AppendsFeaturesSection(t *testing.T) {
	raw := []byte("model = \"gpt-5-codex\"\n")

	out, changed, err := mergeCodexHooksFeatureFlag(raw)
	if err != nil {
		t.Fatalf("mergeCodexHooksFeatureFlag: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got, want := string(out), "model = \"gpt-5-codex\"\n\n[features]\nhooks = true\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestMergeCodexHooksFeatureFlag_InsertsIntoExistingFeatures(t *testing.T) {
	raw := []byte("[features]\nexperimental = true\n\n[mcp_servers]\n")

	out, changed, err := mergeCodexHooksFeatureFlag(raw)
	if err != nil {
		t.Fatalf("mergeCodexHooksFeatureFlag: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got, want := string(out), "[features]\nhooks = true\nexperimental = true\n\n[mcp_servers]\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestMergeCodexHooksFeatureFlag_EnablesDisabledFlag(t *testing.T) {
	raw := []byte("[features]\n  hooks = false # keep hooks enabled for Lumen\n")

	out, changed, err := mergeCodexHooksFeatureFlag(raw)
	if err != nil {
		t.Fatalf("mergeCodexHooksFeatureFlag: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got, want := string(out), "[features]\n  hooks = true # keep hooks enabled for Lumen\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestMergeCodexHooksFeatureFlag_Idempotent(t *testing.T) {
	raw := []byte("[features]\nhooks = true\n")

	out, changed, err := mergeCodexHooksFeatureFlag(raw)
	if err != nil {
		t.Fatalf("mergeCodexHooksFeatureFlag: %v", err)
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if string(out) != string(raw) {
		t.Fatalf("config = %q, want original %q", out, raw)
	}
}

func TestMergeCodexHooksFeatureFlag_RejectsInvalidValue(t *testing.T) {
	out, changed, err := mergeCodexHooksFeatureFlag([]byte("[features]\nhooks = \"yes\"\n"))
	if err == nil {
		t.Fatal("error = nil, want invalid value error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if out != nil {
		t.Fatalf("out = %q, want nil", out)
	}
}

func TestMergeCodexHooksFeatureFlag_MigratesDeprecatedCodexHooksFlag(t *testing.T) {
	raw := []byte("[features]\n  codex_hooks = true # keep hooks enabled for Lumen\n")

	out, changed, err := mergeCodexHooksFeatureFlag(raw)
	if err != nil {
		t.Fatalf("mergeCodexHooksFeatureFlag: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got, want := string(out), "[features]\n  hooks = true # keep hooks enabled for Lumen\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestMergeCodexHooksFeatureFlag_RemovesDeprecatedCodexHooksWhenHooksExists(t *testing.T) {
	raw := []byte("[features]\nhooks = true\ncodex_hooks = true\n")

	out, changed, err := mergeCodexHooksFeatureFlag(raw)
	if err != nil {
		t.Fatalf("mergeCodexHooksFeatureFlag: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got, want := string(out), "[features]\nhooks = true\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestCodexHooksFeatureEnabledInConfig(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     string
		want    bool
		wantErr bool
	}{
		{name: "missing", raw: "[features]\n", want: false},
		{name: "disabled", raw: "[features]\nhooks = false\n", want: false},
		{name: "enabled", raw: "[features]\nhooks = true\n", want: true},
		{name: "commented", raw: "[features]\nhooks = true # enabled\n", want: true},
		{name: "deprecated only", raw: "[features]\ncodex_hooks = true\n", want: false},
		{name: "wrong table", raw: "[other]\nhooks = true\n", want: false},
		{name: "invalid", raw: "[features]\nhooks = \"yes\"\n", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := codexHooksFeatureEnabledInConfig([]byte(tc.raw))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("enabled = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCodexLauncherPathForGOOS(t *testing.T) {
	pluginRoot := filepath.Join("tmp", "lumen")

	if got, want := codexLauncherPathForGOOS(pluginRoot, "darwin"), filepath.Join(pluginRoot, "scripts", "run"); got != want {
		t.Fatalf("darwin launcher = %q, want %q", got, want)
	}
	if got, want := codexLauncherPathForGOOS(`C:\Users\franz\lumen`, "windows"), `C:\Users\franz\lumen\scripts\run.cmd`; got != want {
		t.Fatalf("windows launcher = %q, want %q", got, want)
	}
}

func TestCodexSessionStartCommandQuotesLauncher(t *testing.T) {
	launcher := "/Users/franz/Code/lumen plugin/scripts/run"

	got := codexSessionStartCommand(launcher)
	if !strings.HasPrefix(got, `'`+launcher+`'`) {
		t.Fatalf("command = %q, want quoted launcher prefix", got)
	}
	if !strings.Contains(got, " hook session-start lumen --host codex") {
		t.Fatalf("command = %q, want session-start invocation", got)
	}
}

func TestCodexSessionStartCommandEscapesShellExpansion(t *testing.T) {
	launcher := "/Users/franz/Code/lumen $(touch nope)/scripts/run"

	got := codexSessionStartCommand(launcher)
	if !strings.HasPrefix(got, `'`+launcher+`'`) {
		t.Fatalf("command = %q, want single-quoted launcher prefix", got)
	}
	if strings.Contains(got, `"`) {
		t.Fatalf("command = %q, want no double-quoted shell interpolation", got)
	}
}

func TestCodexSessionStartCommandEscapesSingleQuotes(t *testing.T) {
	launcher := "/Users/franz/Code/lumen's fork/scripts/run"

	got := codexSessionStartCommand(launcher)
	if want := `'/Users/franz/Code/lumen'\''s fork/scripts/run' hook session-start lumen --host codex`; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestCodexSessionStartCommandEscapesWindowsExpansion(t *testing.T) {
	launcher := `C:\Users\%USERNAME%\lumen\scripts\run.cmd`

	got := codexSessionStartCommand(launcher)
	if want := `"C:\Users\^%USERNAME^%\lumen\scripts\run.cmd" hook session-start lumen --host codex`; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestMergeCodexSessionStartHook_PreservesUnrelatedHooks(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")
	raw := []byte(`{
		"hooks": {
			"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "\"/repo/lumen/scripts/run\" hook session-start lumen --host claude"}]}],
			"SessionStart": [{"matcher": "startup", "hooks": [{"type": "command", "command": "echo foreign"}]}]
		}
	}`)

	out, changed, err := mergeCodexSessionStartHook(raw, command)
	if err != nil {
		t.Fatalf("mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	doc := decodeCodexHooksDocument(t, out)
	if got := codexHookCommand(t, codexHookGroups(t, doc, "PreToolUse")[0]); got != `"/repo/lumen/scripts/run" hook session-start lumen --host claude` {
		t.Fatalf("PreToolUse command = %q, want preserved command", got)
	}
	sessionStart := codexHookGroups(t, doc, "SessionStart")
	if len(sessionStart) != 2 {
		t.Fatalf("SessionStart groups = %d, want 2", len(sessionStart))
	}
	if got := codexHookCommand(t, sessionStart[0]); got != "echo foreign" {
		t.Fatalf("foreign command = %q, want preserved command", got)
	}
	if got := codexHookCommand(t, sessionStart[1]); got != command {
		t.Fatalf("Lumen command = %q, want %q", got, command)
	}
}

func TestMergeCodexSessionStartHook_ReplacesStaleLumenHook(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")
	raw := []byte(`{
		"hooks": {
			"SessionStart": [
				{"matcher": "startup|resume|clear|compact", "hooks": [{"type": "command", "command": "\"/repo/lumen/scripts/run.sh\" hook session-start lumen --host claude"}]},
				{"matcher": "startup", "hooks": [{"type": "command", "command": "echo foreign"}]}
			]
		}
	}`)

	out, changed, err := mergeCodexSessionStartHook(raw, command)
	if err != nil {
		t.Fatalf("mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	groups := codexHookGroups(t, decodeCodexHooksDocument(t, out), "SessionStart")
	if len(groups) != 2 {
		t.Fatalf("SessionStart groups = %d, want 2", len(groups))
	}
	lumenCount := 0
	for _, group := range groups {
		if codexHookGroupRunsLumenSessionStart(group) {
			lumenCount++
			if got := group["matcher"]; got != codexSessionStartMatcher {
				t.Fatalf("Lumen matcher = %v, want %q", got, codexSessionStartMatcher)
			}
		}
	}
	if lumenCount != 1 {
		t.Fatalf("Lumen hook groups = %d, want 1", lumenCount)
	}
	if strings.Contains(string(out), "compact") {
		t.Fatalf("output contains compact: %s", out)
	}
}

func TestMergeCodexSessionStartHook_DeduplicatesLumenHooks(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")
	raw := []byte(`{
		"hooks": {
			"SessionStart": [
				{"matcher": "startup", "hooks": [{"type": "command", "command": "\"/repo/lumen/scripts/run.sh\" hook session-start lumen --host claude"}]},
				{"matcher": "resume", "hooks": [{"type": "command", "command": "\"/repo/lumen/scripts/run\" hook session-start lumen --host claude"}]}
			]
		}
	}`)

	out, changed, err := mergeCodexSessionStartHook(raw, command)
	if err != nil {
		t.Fatalf("mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	groups := codexHookGroups(t, decodeCodexHooksDocument(t, out), "SessionStart")
	if len(groups) != 1 {
		t.Fatalf("SessionStart groups = %d, want 1", len(groups))
	}
	if got := codexHookCommand(t, groups[0]); got != command {
		t.Fatalf("command = %q, want %q", got, command)
	}
}

func TestMergeCodexSessionStartHook_PreservesForeignSessionStartCommands(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")
	foreign := `"/tmp/not-lumen/scripts/run" hook session-start lumen --host claude`
	raw, err := json.Marshal(map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"matcher": "startup",
					"hooks": []any{
						map[string]any{"type": "command", "command": foreign},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	out, changed, err := mergeCodexSessionStartHook(raw, command)
	if err != nil {
		t.Fatalf("mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	groups := codexHookGroups(t, decodeCodexHooksDocument(t, out), "SessionStart")
	if len(groups) != 2 {
		t.Fatalf("SessionStart groups = %d, want 2", len(groups))
	}
	if got := codexHookCommand(t, groups[0]); got != foreign {
		t.Fatalf("foreign command = %q, want %q", got, foreign)
	}
}

func TestMergeCodexSessionStartHook_PreservesForeignHookInMixedGroup(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")
	raw := []byte(`{
		"hooks": {
			"SessionStart": [
				{
					"matcher": "startup",
					"hooks": [
						{"type": "command", "command": "\"/repo/lumen/scripts/run\" hook session-start lumen --host claude"},
						{"type": "command", "command": "echo foreign"}
					]
				}
			]
		}
	}`)

	out, changed, err := mergeCodexSessionStartHook(raw, command)
	if err != nil {
		t.Fatalf("mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	groups := codexHookGroups(t, decodeCodexHooksDocument(t, out), "SessionStart")
	if len(groups) != 2 {
		t.Fatalf("SessionStart groups = %d, want 2", len(groups))
	}
	if got := codexHookCommands(t, groups[0]); len(got) != 1 || got[0] != "echo foreign" {
		t.Fatalf("first group commands = %#v, want preserved foreign hook only", got)
	}
	if got := codexHookCommand(t, groups[1]); got != command {
		t.Fatalf("Lumen command = %q, want %q", got, command)
	}
}

func TestMergeCodexSessionStartHook_ReplacesMovedUnmarkedLumenHook(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")
	raw := []byte(`{
		"hooks": {
			"SessionStart": [
				{"matcher": "startup", "hooks": [{"type": "command", "command": "\"/repo/lumen fork/scripts/run.sh\" hook session-start lumen --host claude"}]}
			]
		}
	}`)

	out, changed, err := mergeCodexSessionStartHook(raw, command)
	if err != nil {
		t.Fatalf("mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	groups := codexHookGroups(t, decodeCodexHooksDocument(t, out), "SessionStart")
	if len(groups) != 1 {
		t.Fatalf("SessionStart groups = %d, want 1", len(groups))
	}
	if got := codexHookCommand(t, groups[0]); got != command {
		t.Fatalf("command = %q, want %q", got, command)
	}
}

func TestMergeCodexSessionStartHook_MalformedJSON(t *testing.T) {
	out, changed, err := mergeCodexSessionStartHook([]byte(`{"hooks":`), "command")
	if err == nil {
		t.Fatal("error = nil, want error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if out != nil {
		t.Fatalf("out = %q, want nil", out)
	}
}

func TestMergeCodexSessionStartHook_NullDocument(t *testing.T) {
	out, changed, err := mergeCodexSessionStartHook([]byte(`null`), "command")
	if err == nil {
		t.Fatal("error = nil, want error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if out != nil {
		t.Fatalf("out = %q, want nil", out)
	}
}

func TestMergeCodexSessionStartHook_Idempotent(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen/scripts/run")

	first, changed, err := mergeCodexSessionStartHook(nil, command)
	if err != nil {
		t.Fatalf("first mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("first changed = false, want true")
	}

	second, changed, err := mergeCodexSessionStartHook(first, command)
	if err != nil {
		t.Fatalf("second mergeCodexSessionStartHook: %v", err)
	}
	if changed {
		t.Fatal("second changed = true, want false")
	}
	if string(second) != string(first) {
		t.Fatalf("second output changed:\n%s\nwant:\n%s", second, first)
	}
}

func TestMergeCodexSessionStartHook_IdempotentOutsideLumenDirectory(t *testing.T) {
	command := codexSessionStartCommand("/repo/lumen fork/scripts/run")

	first, changed, err := mergeCodexSessionStartHook(nil, command)
	if err != nil {
		t.Fatalf("first mergeCodexSessionStartHook: %v", err)
	}
	if !changed {
		t.Fatal("first changed = false, want true")
	}

	second, changed, err := mergeCodexSessionStartHook(first, command)
	if err != nil {
		t.Fatalf("second mergeCodexSessionStartHook: %v", err)
	}
	if changed {
		t.Fatal("second changed = true, want false")
	}
	if string(second) != string(first) {
		t.Fatalf("second output changed:\n%s\nwant:\n%s", second, first)
	}
}

func TestMergeCodexSessionStartHook_PreservesMalformedHooksShape(t *testing.T) {
	raw := []byte(`{"hooks": []}`)
	out, changed, err := mergeCodexSessionStartHook(raw, "command")
	if err == nil {
		t.Fatal("error = nil, want error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if out != nil {
		t.Fatalf("out = %q, want nil", out)
	}
}

func TestMergeCodexSessionStartHook_PreservesMalformedSessionStartShape(t *testing.T) {
	raw := []byte(`{"hooks": {"SessionStart": {}}}`)
	out, changed, err := mergeCodexSessionStartHook(raw, "command")
	if err == nil {
		t.Fatal("error = nil, want error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if out != nil {
		t.Fatalf("out = %q, want nil", out)
	}
}

func TestMergeCodexSessionStartHook_PreservesMalformedSessionStartGroup(t *testing.T) {
	raw := []byte(`{"hooks": {"SessionStart": ["bad"]}}`)
	out, changed, err := mergeCodexSessionStartHook(raw, "command")
	if err == nil {
		t.Fatal("error = nil, want error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if out != nil {
		t.Fatalf("out = %q, want nil", out)
	}
}

func TestMergeCodexSessionStartHook_PreservesMalformedSessionStartGroupHooks(t *testing.T) {
	raw := []byte(`{"hooks": {"SessionStart": [{"hooks": {}}]}}`)
	out, changed, err := mergeCodexSessionStartHook(raw, "command")
	if err == nil {
		t.Fatal("error = nil, want error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if out != nil {
		t.Fatalf("out = %q, want nil", out)
	}
}

func TestMergeCodexSessionStartHook_PreservesMalformedSessionStartHook(t *testing.T) {
	raw := []byte(`{"hooks": {"SessionStart": [{"hooks": ["bad"]}]}}`)
	out, changed, err := mergeCodexSessionStartHook(raw, "command")
	if err == nil {
		t.Fatal("error = nil, want error")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if out != nil {
		t.Fatalf("out = %q, want nil", out)
	}
}

func decodeCodexHooksDocument(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("json.Unmarshal: %v\n%s", err, raw)
	}
	return doc
}

func codexHookGroups(t *testing.T, doc map[string]any, event string) []map[string]any {
	t.Helper()
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks missing or wrong type: %#v", doc["hooks"])
	}
	rawGroups, ok := hooks[event].([]any)
	if !ok {
		t.Fatalf("%s hooks missing or wrong type: %#v", event, hooks[event])
	}
	groups := make([]map[string]any, 0, len(rawGroups))
	for _, rawGroup := range rawGroups {
		group, ok := rawGroup.(map[string]any)
		if !ok {
			t.Fatalf("hook group wrong type: %#v", rawGroup)
		}
		groups = append(groups, group)
	}
	return groups
}

func codexHookCommand(t *testing.T, group map[string]any) string {
	t.Helper()
	hook := codexSingleHook(t, group)
	command, ok := hook["command"].(string)
	if !ok {
		t.Fatalf("command missing or wrong type: %#v", hook["command"])
	}
	return command
}

func codexHookCommands(t *testing.T, group map[string]any) []string {
	t.Helper()
	rawHooks, ok := group["hooks"].([]any)
	if !ok {
		t.Fatalf("hooks = %#v, want hook array", group["hooks"])
	}
	commands := make([]string, 0, len(rawHooks))
	for _, rawHook := range rawHooks {
		hook, ok := rawHook.(map[string]any)
		if !ok {
			t.Fatalf("hook wrong type: %#v", rawHook)
		}
		command, ok := hook["command"].(string)
		if !ok {
			t.Fatalf("command missing or wrong type: %#v", hook["command"])
		}
		commands = append(commands, command)
	}
	return commands
}

func codexHookStatusMessageForGroup(t *testing.T, group map[string]any) string {
	t.Helper()
	hook := codexSingleHook(t, group)
	statusMessage, ok := hook["statusMessage"].(string)
	if !ok {
		t.Fatalf("statusMessage missing or wrong type: %#v", hook["statusMessage"])
	}
	return statusMessage
}

func codexSingleHook(t *testing.T, group map[string]any) map[string]any {
	t.Helper()
	rawHooks, ok := group["hooks"].([]any)
	if !ok || len(rawHooks) != 1 {
		t.Fatalf("hooks = %#v, want one hook", group["hooks"])
	}
	hook, ok := rawHooks[0].(map[string]any)
	if !ok {
		t.Fatalf("hook wrong type: %#v", rawHooks[0])
	}
	return hook
}

// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

// Command mutate is the curated production-mutation gate for the worktree
// integration suite. For every mutant in mutants.json it works in a disposable
// copy of the source tree:
//
//  1. baseline: every designated test passes on unmodified source;
//  2. applies one exact production-code edit that must still compile;
//  3. runs the designated test (which rebuilds the Lumen binary from the
//     mutated copy) and requires it to fail with its designated assertion
//     marker;
//  4. restores the file and requires the designated test to pass again.
//
// Compilation failures, timeouts, failures without the designated marker,
// and edits to test code all count as surviving mutants and fail the gate.
// Diffs, exact commands, and logs are kept in the output directory.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type mutant struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	File        string `json:"file"`
	Find        string `json:"find"`
	Replace     string `json:"replace"`
	Test        string `json:"test"`
	Marker      string `json:"marker"`
}

type outcome struct {
	ID       string `json:"id"`
	Test     string `json:"test"`
	Marker   string `json:"marker"`
	Verdict  string `json:"verdict"` // killed, survived, unevaluable
	Reason   string `json:"reason,omitempty"`
	Command  string `json:"command"`
	Restored bool   `json:"restored_pass"`
	Duration string `json:"duration"`
}

// Diagnostic counters are the observation channel. Mutating them would test
// the counters rather than production behavior, so such mutants are rejected.
var forbiddenInFind = []string{"FilesParsed", "RevisionsAttached", "FilesHashed", "FilesReused", "logIndexPass", "scan completed"}

func main() {
	manifest := flag.String("manifest", "e2e/worktrees/mutants.json", "mutant manifest relative to the module root")
	only := flag.String("only", "", "comma-separated mutant IDs to run (default: all)")
	out := flag.String("out", "", "artifact directory (default: a new temporary directory)")
	timeout := flag.Duration("timeout", 10*time.Minute, "per designated test run")
	keep := flag.Bool("keep", false, "keep the disposable source copy")
	flag.Parse()
	if err := run(*manifest, *only, *out, *timeout, *keep); err != nil {
		fmt.Fprintln(os.Stderr, "mutation gate:", err)
		os.Exit(1)
	}
}

func run(manifestPath, only, outDir string, timeout time.Duration, keep bool) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, manifestPath))
	if err != nil {
		return err
	}
	var mutants []mutant
	if err := json.Unmarshal(data, &mutants); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	if only != "" {
		ids := strings.Split(only, ",")
		mutants = slices.DeleteFunc(mutants, func(m mutant) bool { return !slices.Contains(ids, m.ID) })
	}
	if len(mutants) == 0 {
		return errors.New("no mutants selected")
	}
	if outDir == "" {
		if outDir, err = os.MkdirTemp("", "lumen-mutations-"); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "lumen-mutation-src-")
	if err != nil {
		return err
	}
	if keep {
		fmt.Println("source copy:", work)
	} else {
		defer func() { _ = os.RemoveAll(work) }()
	}
	if err := copySource(root, work); err != nil {
		return fmt.Errorf("copy source: %w", err)
	}
	fmt.Printf("artifacts: %s\n", outDir)

	var tests []string
	for _, m := range mutants {
		if !slices.Contains(tests, m.Test) {
			tests = append(tests, m.Test)
		}
	}
	for _, test := range tests {
		fmt.Printf("baseline %s ... ", test)
		cmd, output, err := runTest(work, test, timeout)
		_ = os.WriteFile(filepath.Join(outDir, "baseline-"+test+".log"), []byte(cmd+"\n\n"+output), 0o644)
		if err != nil {
			fmt.Println("FAIL")
			return fmt.Errorf("baseline %s is not green; see %s", test, outDir)
		}
		fmt.Println("ok")
	}

	var results []outcome
	for _, m := range mutants {
		r := evaluate(work, outDir, m, timeout)
		results = append(results, r)
		fmt.Printf("%-34s %-11s restored=%-5v %s\n", m.ID, r.Verdict, r.Restored, r.Reason)
	}
	summary, _ := json.MarshalIndent(results, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "summary.json"), summary, 0o644); err != nil {
		return err
	}
	failed := 0
	for _, r := range results {
		if r.Verdict != "killed" || !r.Restored {
			failed++
		}
	}
	fmt.Printf("\n%d/%d mutants killed with designated assertions and restored green\n", len(results)-failed, len(results))
	if failed > 0 {
		return fmt.Errorf("%d mutants survived, were unevaluable, or did not restore green", failed)
	}
	return nil
}

func evaluate(work, outDir string, m mutant, timeout time.Duration) outcome {
	start := time.Now()
	r := outcome{ID: m.ID, Test: m.Test, Marker: m.Marker, Verdict: "unevaluable"}
	defer func() { r.Duration = time.Since(start).Round(time.Second).String() }()
	dir := filepath.Join(outDir, m.ID)
	_ = os.MkdirAll(dir, 0o755)
	if reason := validate(m); reason != "" {
		r.Reason = reason
		return r
	}
	path := filepath.Join(work, m.File)
	original, err := os.ReadFile(path)
	if err != nil {
		r.Reason = err.Error()
		return r
	}
	if n := strings.Count(string(original), m.Find); n != 1 {
		r.Reason = fmt.Sprintf("find text occurs %d times in %s", n, m.File)
		return r
	}
	mutated := strings.Replace(string(original), m.Find, m.Replace, 1)
	if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
		r.Reason = err.Error()
		return r
	}
	restore := func() { _ = os.WriteFile(path, original, 0o644) }
	_ = os.WriteFile(filepath.Join(dir, "mutant.diff"), []byte(unifiedDiff(m.File, string(original), mutated)), 0o644)

	build := exec.Command("go", "build", "-tags=fts5", "-o", os.DevNull, ".")
	build.Dir, build.Env = work, append(os.Environ(), "CGO_ENABLED=1")
	if output, err := build.CombinedOutput(); err != nil {
		restore()
		_ = os.WriteFile(filepath.Join(dir, "build.log"), output, 0o644)
		r.Reason = "mutant does not compile"
		return r
	}

	cmd, output, err := runTest(work, m.Test, timeout)
	r.Command = cmd
	_ = os.WriteFile(filepath.Join(dir, "command.txt"), []byte(cmd+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "mutated.log"), []byte(output), 0o644)
	restore()
	switch {
	case err == nil:
		r.Verdict, r.Reason = "survived", "designated test passed"
	case errors.Is(err, context.DeadlineExceeded) || strings.Contains(output, "panic: test timed out"):
		r.Reason = "designated test timed out"
	case strings.Contains(output, "[build failed]") || strings.Contains(output, "build lumen from"):
		r.Reason = "test or binary build failed"
	case !strings.Contains(output, "--- FAIL: "+m.Test):
		r.Reason = "designated test did not report a failure"
	case !strings.Contains(output, "["+m.Marker+"]"):
		r.Verdict, r.Reason = "survived", "failed without designated marker "+m.Marker+": "+firstFailure(output)
	default:
		r.Verdict, r.Reason = "killed", firstFailure(output)
	}
	_ = os.WriteFile(filepath.Join(dir, "designated-failure.txt"), []byte(r.Reason+"\n"), 0o644)

	_, restored, err := runTest(work, m.Test, timeout)
	_ = os.WriteFile(filepath.Join(dir, "restored.log"), []byte(restored), 0o644)
	r.Restored = err == nil
	return r
}

func validate(m mutant) string {
	switch {
	case m.ID == "" || m.File == "" || m.Find == "" || m.Test == "" || m.Marker == "":
		return "incomplete manifest entry"
	case m.Find == m.Replace:
		return "replacement does not change the source"
	case strings.HasSuffix(m.File, "_test.go") || strings.HasPrefix(filepath.ToSlash(m.File), "e2e/"):
		return "mutants must change production code, not tests"
	case !strings.HasSuffix(m.File, ".go"):
		return "mutants must change Go production code"
	}
	for _, word := range forbiddenInFind {
		if strings.Contains(m.Find, word) || strings.Contains(m.Replace, word) {
			return "mutants must not alter diagnostic counters (" + word + ")"
		}
	}
	return ""
}

func runTest(dir, test string, timeout time.Duration) (string, string, error) {
	args := []string{"test", "-tags=fts5,worktree_e2e", "-count=1", "-v", "-timeout=" + timeout.String(), "-run", "^" + test + "$", "./e2e/worktrees/"}
	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir, cmd.Env = dir, append(os.Environ(), "CGO_ENABLED=1")
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	return "(cd " + dir + " && CGO_ENABLED=1 go " + strings.Join(args, " ") + ")", buf.String(), err
}

func firstFailure(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "[A:") {
			return strings.TrimSpace(line)
		}
	}
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "--- FAIL") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func unifiedDiff(name, before, after string) string {
	dir, err := os.MkdirTemp("", "lumen-mutant-diff-")
	if err != nil {
		return ""
	}
	defer func() { _ = os.RemoveAll(dir) }()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	_ = os.WriteFile(a, []byte(before), 0o644)
	_ = os.WriteFile(b, []byte(after), 0o644)
	out, _ := exec.Command("git", "diff", "--no-index", "--no-color", "--src-prefix=a/"+name+"#", "--dst-prefix=b/"+name+"#", a, b).Output()
	return string(out)
}

func moduleRoot() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", err
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		return "", errors.New("run inside the lumen module")
	}
	return filepath.Dir(gomod), nil
}

// copySource copies tracked and untracked, non-ignored files, so uncommitted
// work is evaluated exactly as it is in the checkout.
func copySource(root, dst string) error {
	cmd := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	for rel := range strings.SplitSeq(strings.TrimRight(string(out), "\x00"), "\x00") {
		src := filepath.Join(root, rel)
		info, err := os.Lstat(src)
		if os.IsNotExist(err) {
			continue // deleted in the working tree
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(src)
			if err != nil {
				return err
			}
			if err := os.Symlink(link, target); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := copyFile(src, target, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

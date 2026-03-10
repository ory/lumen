package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTask_applyDefaults(t *testing.T) {
	tests := []struct {
		name        string
		task        Task
		wantTimeout int
	}{
		{
			name:        "zero values get defaults",
			task:        Task{},
			wantTimeout: 900,
		},
		{
			name:        "non-zero timeout preserved",
			task:        Task{TimeoutSeconds: 600},
			wantTimeout: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.task.applyDefaults()
			if tt.task.TimeoutSeconds != tt.wantTimeout {
				t.Errorf("TimeoutSeconds = %d, want %d", tt.task.TimeoutSeconds, tt.wantTimeout)
			}
		})
	}
}

func TestTask_Validate(t *testing.T) {
	// Create a temp dir with a dummy patch file for the valid case.
	tmpDir := t.TempDir()
	patchFile := "gold.patch"
	if err := os.WriteFile(filepath.Join(tmpDir, patchFile), []byte("diff content"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		task    Task
		baseDir string
		wantErr string
	}{
		{
			name:    "missing id",
			task:    Task{},
			baseDir: tmpDir,
			wantErr: "missing id",
		},
		{
			name:    "missing repo",
			task:    Task{ID: "t1"},
			baseDir: tmpDir,
			wantErr: "missing repo",
		},
		{
			name:    "missing base_commit",
			task:    Task{ID: "t1", Repo: "https://example.com/repo"},
			baseDir: tmpDir,
			wantErr: "missing base_commit",
		},
		{
			name:    "missing issue_body",
			task:    Task{ID: "t1", Repo: "https://example.com/repo", BaseCommit: "abc123"},
			baseDir: tmpDir,
			wantErr: "missing issue_body",
		},
		{
			name:    "missing gold_patch_file",
			task:    Task{ID: "t1", Repo: "https://example.com/repo", BaseCommit: "abc123", IssueBody: "fix bug"},
			baseDir: tmpDir,
			wantErr: "missing gold_patch_file",
		},
		{
			name:    "gold patch file does not exist",
			task:    Task{ID: "t1", Repo: "https://example.com/repo", BaseCommit: "abc123", IssueBody: "fix bug", GoldPatchFile: "nonexistent.patch"},
			baseDir: tmpDir,
			wantErr: "gold patch not found",
		},
		{
			name:    "all fields valid",
			task:    Task{ID: "t1", Repo: "https://example.com/repo", BaseCommit: "abc123", IssueBody: "fix bug", GoldPatchFile: patchFile},
			baseDir: tmpDir,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate(tt.baseDir)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestTask_GoldPatch(t *testing.T) {
	tmpDir := t.TempDir()
	content := "--- a/file.go\n+++ b/file.go\n@@ -1 +1 @@\n-old\n+new\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "fix.patch"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("patch file exists", func(t *testing.T) {
		task := Task{ID: "t1", GoldPatchFile: "fix.patch"}
		got, err := task.GoldPatch(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != content {
			t.Errorf("got %q, want %q", got, content)
		}
	})

	t.Run("patch file missing", func(t *testing.T) {
		task := Task{ID: "t1", GoldPatchFile: "missing.patch"}
		_, err := task.GoldPatch(tmpDir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestLoadTasks(t *testing.T) {
	writeTask := func(dir, name string, task Task) {
		data, err := json.Marshal(task)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("single valid JSON returns 1 task with defaults", func(t *testing.T) {
		dir := t.TempDir()
		writeTask(dir, "task1.json", Task{ID: "t1", Language: "go", Repo: "r", BaseCommit: "c", IssueBody: "b", GoldPatchFile: "p"})

		tasks, err := LoadTasks(dir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("got %d tasks, want 1", len(tasks))
		}
		if tasks[0].TimeoutSeconds != 900 {
			t.Errorf("TimeoutSeconds = %d, want 900", tasks[0].TimeoutSeconds)
		}
	})

	t.Run("multiple JSON files in nested dirs", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "sub")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		writeTask(dir, "task1.json", Task{ID: "t1", Language: "go", Repo: "r", BaseCommit: "c", IssueBody: "b", GoldPatchFile: "p"})
		writeTask(sub, "task2.json", Task{ID: "t2", Language: "python", Repo: "r", BaseCommit: "c", IssueBody: "b", GoldPatchFile: "p"})

		tasks, err := LoadTasks(dir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("got %d tasks, want 2", len(tasks))
		}
	})

	t.Run("filter by language", func(t *testing.T) {
		dir := t.TempDir()
		writeTask(dir, "go.json", Task{ID: "t1", Language: "go", Repo: "r", BaseCommit: "c", IssueBody: "b", GoldPatchFile: "p"})
		writeTask(dir, "py.json", Task{ID: "t2", Language: "python", Repo: "r", BaseCommit: "c", IssueBody: "b", GoldPatchFile: "p"})

		tasks, err := LoadTasks(dir, []string{"go"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("got %d tasks, want 1", len(tasks))
		}
		if tasks[0].Language != "go" {
			t.Errorf("Language = %q, want %q", tasks[0].Language, "go")
		}
	})

	t.Run("filter by multiple languages", func(t *testing.T) {
		dir := t.TempDir()
		writeTask(dir, "go.json", Task{ID: "t1", Language: "go", Repo: "r", BaseCommit: "c", IssueBody: "b", GoldPatchFile: "p"})
		writeTask(dir, "py.json", Task{ID: "t2", Language: "python", Repo: "r", BaseCommit: "c", IssueBody: "b", GoldPatchFile: "p"})
		writeTask(dir, "ts.json", Task{ID: "t3", Language: "typescript", Repo: "r", BaseCommit: "c", IssueBody: "b", GoldPatchFile: "p"})

		tasks, err := LoadTasks(dir, []string{"go", "python"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("got %d tasks, want 2", len(tasks))
		}
	})

	t.Run("empty dir returns error", func(t *testing.T) {
		dir := t.TempDir()
		_, err := LoadTasks(dir, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no tasks found") {
			t.Errorf("error %q does not contain %q", err.Error(), "no tasks found")
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{invalid json"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadTasks(dir, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "parsing") {
			t.Errorf("error %q does not contain %q", err.Error(), "parsing")
		}
	})
}

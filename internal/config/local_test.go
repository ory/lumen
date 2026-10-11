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

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/models"
)

func TestLocalBackend_EnvDefaults(t *testing.T) {
	clearServerEnv(t)
	t.Setenv("LUMEN_BACKEND", "local")

	svc, err := NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	s := svc.Servers()[0]
	if s.Backend != BackendLocal {
		t.Errorf("Backend = %q, want %q", s.Backend, BackendLocal)
	}
	if s.Model != models.DefaultLocalModel {
		t.Errorf("Model = %q, want %q", s.Model, models.DefaultLocalModel)
	}
	if s.Host != "" {
		t.Errorf("Host = %q, want empty for the in-process backend", s.Host)
	}
	if s.ModelPath != "" {
		t.Errorf("ModelPath = %q, want empty (managed download)", s.ModelPath)
	}
	if got := svc.ServerDims(0); got != 3584 {
		t.Errorf("ServerDims = %d, want 3584", got)
	}
}

func TestLocalBackend_EnvModelPath(t *testing.T) {
	clearServerEnv(t)
	path := filepath.Join(t.TempDir(), "nomic.gguf")
	t.Setenv("LUMEN_BACKEND", "local")
	t.Setenv("LUMEN_MODEL_PATH", path)
	// Remote host variables must not leak into the in-process backend.
	t.Setenv("OLLAMA_HOST", "http://ollama:1")

	svc, err := NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	s := svc.Servers()[0]
	if s.ModelPath != path {
		t.Errorf("ModelPath = %q, want %q", s.ModelPath, path)
	}
	if s.Host != "" {
		t.Errorf("Host = %q, want empty", s.Host)
	}
}

func TestLocalBackend_YAMLSurvivesSelectionAndOverride(t *testing.T) {
	clearServerEnv(t)
	path := filepath.Join(t.TempDir(), "nomic.gguf")
	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(`
servers:
  - backend: local
    model: nomic-ai/nomic-embed-code-GGUF
    model_path: `+path+`
  - backend: ollama
    host: http://localhost:11434
    model: all-minilm
`), 0o644); err != nil {
		t.Fatal(err)
	}

	for name, opt := range map[string]Option{
		"selection": WithServerSelection("", BackendLocal),
		"override":  WithModelOverride(models.DefaultLocalModel),
	} {
		t.Run(name, func(t *testing.T) {
			svc, err := NewConfigService(cfgFile, opt)
			if err != nil {
				t.Fatalf("NewConfigService: %v", err)
			}
			s := svc.Servers()[0]
			if s.Backend != BackendLocal || s.ModelPath != path {
				t.Errorf("server[0] = %+v, want local backend with model_path %q", s, path)
			}
		})
	}
}

func TestLocalBackend_Validation(t *testing.T) {
	for name, tc := range map[string]struct {
		yaml string
		want string
	}{
		"unsupported model": {
			yaml: `servers: [{backend: local, model: all-minilm}]`,
			want: "not supported by the local backend",
		},
		"relative model path": {
			yaml: `servers: [{backend: local, model: nomic-ai/nomic-embed-code-GGUF, model_path: models/x.gguf}]`,
			want: "must be an absolute path",
		},
		"model path on remote backend": {
			yaml: `servers: [{backend: ollama, host: "http://x:1", model: all-minilm, model_path: /tmp/x.gguf}]`,
			want: "model_path is only supported by the local backend",
		},
		"dims mismatch": {
			yaml: `servers: [{backend: local, model: nomic-ai/nomic-embed-code-GGUF, dims: 768}]`,
			want: "dims",
		},
	} {
		t.Run(name, func(t *testing.T) {
			clearServerEnv(t)
			f := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(f, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := NewConfigService(f)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("NewConfigService error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestLocalBackend_HomeRelativeModelPath(t *testing.T) {
	clearServerEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	f := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(f, []byte(`servers: [{backend: local, model: nomic-ai/nomic-embed-code-GGUF, model_path: ~/models/nomic.gguf}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	svc, err := NewConfigService(f)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if got, want := svc.Servers()[0].ModelPath, filepath.Join(home, "models", "nomic.gguf"); got != want {
		t.Errorf("ModelPath = %q, want %q", got, want)
	}
}

func TestXDGCacheDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/custom/cache")
	if got := XDGCacheDir(); got != "/custom/cache" {
		t.Errorf("XDGCacheDir() = %q, want /custom/cache", got)
	}
	t.Setenv("XDG_CACHE_HOME", "")
	home, _ := os.UserHomeDir()
	if got, want := XDGCacheDir(), filepath.Join(home, ".cache"); got != want {
		t.Errorf("XDGCacheDir() = %q, want %q", got, want)
	}
}

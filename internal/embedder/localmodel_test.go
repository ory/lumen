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

package embedder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/models"
)

func testArtifact(t *testing.T, url string, content []byte) modelArtifact {
	t.Helper()
	sum := sha256.Sum256(content)
	return modelArtifact{
		URL:    url,
		SHA256: hex.EncodeToString(sum[:]),
		Size:   int64(len(content)),
		Path:   filepath.Join(t.TempDir(), "model.gguf"),
	}
}

func modelContent() []byte {
	return bytes.Repeat([]byte("GGUF model bytes "), 64<<10)
}

// rangeServer serves content with Range support and records requests.
type rangeServer struct {
	content  []byte
	requests atomic.Int32
	ranges   []string
	mu       sync.Mutex
}

func (s *rangeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.requests.Add(1)
	s.mu.Lock()
	s.ranges = append(s.ranges, r.Header.Get("Range"))
	s.mu.Unlock()
	http.ServeContent(w, r, "model.gguf", time.Time{}, bytes.NewReader(s.content))
}

func TestModelArtifact_DownloadsVerifiesAndInstalls(t *testing.T) {
	content := modelContent()
	srv := &rangeServer{content: content}
	ts := httptest.NewServer(srv)
	defer ts.Close()
	a := testArtifact(t, ts.URL, content)

	var reports []int64
	err := a.ensure(context.Background(), ts.Client(), func(done, total int64) {
		if total != int64(len(content)) {
			t.Errorf("progress total = %d, want %d", total, len(content))
		}
		reports = append(reports, done)
	})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got, err := os.ReadFile(a.Path)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("installed file differs from served content (err=%v)", err)
	}
	if _, err := os.Stat(a.Path + ".partial"); !os.IsNotExist(err) {
		t.Errorf("partial file left behind: %v", err)
	}
	if len(reports) == 0 || reports[len(reports)-1] != int64(len(content)) {
		t.Errorf("progress reports = %v, want final report of %d", reports, len(content))
	}
	if len(reports) > 105 {
		t.Errorf("progress reported %d times, want at most about one per percent", len(reports))
	}
}

func TestModelArtifact_ChecksumMismatchIsNotInstalled(t *testing.T) {
	content := modelContent()
	srv := &rangeServer{content: append([]byte("tampered"), content[8:]...)}
	ts := httptest.NewServer(srv)
	defer ts.Close()
	a := testArtifact(t, ts.URL, content)

	err := a.ensure(context.Background(), ts.Client(), nil)
	if !errors.Is(err, errChecksum) {
		t.Fatalf("ensure error = %v, want checksum mismatch", err)
	}
	if _, err := os.Stat(a.Path); !os.IsNotExist(err) {
		t.Errorf("model installed despite checksum mismatch")
	}
	if _, err := os.Stat(a.Path + ".partial"); !os.IsNotExist(err) {
		t.Errorf("corrupt partial download kept")
	}
	if n := srv.requests.Load(); n != 1 {
		t.Errorf("requests = %d, want 1 (checksum failures are not retried)", n)
	}
}

func TestModelArtifact_ResumesPartialDownload(t *testing.T) {
	content := modelContent()
	srv := &rangeServer{content: content}
	ts := httptest.NewServer(srv)
	defer ts.Close()
	a := testArtifact(t, ts.URL, content)

	half := len(content) / 2
	if err := os.WriteFile(a.Path+".partial", content[:half], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.ensure(context.Background(), ts.Client(), nil); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got, _ := os.ReadFile(a.Path)
	if !bytes.Equal(got, content) {
		t.Fatal("resumed download differs from served content")
	}
	if want := "bytes=" + strconv.Itoa(half) + "-"; len(srv.ranges) != 1 || srv.ranges[0] != want {
		t.Errorf("Range headers = %q, want [%q]", srv.ranges, want)
	}
}

func TestModelArtifact_InterruptedDownloadIsNotInstalled(t *testing.T) {
	content := modelContent()
	var requests atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content[:len(content)/3])
		// Drop the connection mid-body.
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("response writer cannot hijack")
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer ts.Close()
	a := testArtifact(t, ts.URL, content)

	if err := a.ensure(context.Background(), ts.Client(), nil); err == nil {
		t.Fatal("ensure succeeded on a truncated download")
	}
	if _, err := os.Stat(a.Path); !os.IsNotExist(err) {
		t.Errorf("truncated download installed as a complete model")
	}
	if a.complete() {
		t.Error("complete() = true after interrupted download")
	}
	if n := requests.Load(); n != downloadAttempts {
		t.Errorf("requests = %d, want %d attempts", n, downloadAttempts)
	}
}

func TestModelArtifact_CancelKeepsPartial(t *testing.T) {
	content := modelContent()
	ctx, cancel := context.WithCancel(context.Background())
	var partial string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content[:len(content)/2])
		w.(http.Flusher).Flush()
		// Cancel only once the client has persisted some bytes.
		for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
			if st, err := os.Stat(partial); err == nil && st.Size() > 0 {
				break
			}
		}
		cancel()
		<-r.Context().Done()
	}))
	defer ts.Close()
	a := testArtifact(t, ts.URL, content)
	partial = a.Path + ".partial"

	err := a.ensure(ctx, ts.Client(), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ensure error = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(a.Path); !os.IsNotExist(err) {
		t.Error("cancelled download installed")
	}
	if st, err := os.Stat(a.Path + ".partial"); err != nil || st.Size() == 0 {
		t.Errorf("partial download not kept for resume: %v", err)
	}
}

func TestModelArtifact_CachedModelNeedsNoNetwork(t *testing.T) {
	content := modelContent()
	a := testArtifact(t, "http://127.0.0.1:1/unreachable", content)
	if err := os.WriteFile(a.Path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.ensure(context.Background(), http.DefaultClient, nil); err != nil {
		t.Fatalf("ensure with cached model: %v", err)
	}
}

func TestModelArtifact_ReplacesTruncatedInstall(t *testing.T) {
	content := modelContent()
	srv := &rangeServer{content: content}
	ts := httptest.NewServer(srv)
	defer ts.Close()
	a := testArtifact(t, ts.URL, content)
	if err := os.WriteFile(a.Path, content[:100], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.ensure(context.Background(), ts.Client(), nil); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got, _ := os.ReadFile(a.Path)
	if !bytes.Equal(got, content) {
		t.Fatal("truncated install was not replaced")
	}
}

func TestModelArtifact_ConcurrentCallersDownloadOnce(t *testing.T) {
	content := modelContent()
	srv := &rangeServer{content: content}
	ts := httptest.NewServer(srv)
	defer ts.Close()
	a := testArtifact(t, ts.URL, content)

	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := range errs {
		wg.Go(func() { errs[i] = a.ensure(context.Background(), ts.Client(), nil) })
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("caller %d: %v", i, err)
		}
	}
	if n := srv.requests.Load(); n != 1 {
		t.Errorf("requests = %d, want exactly one download", n)
	}
}

func TestEnsureLocalModel_UserPath(t *testing.T) {
	srv := config.ServerConfig{Backend: config.BackendLocal, Model: models.DefaultLocalModel}

	srv.ModelPath = filepath.Join(t.TempDir(), "missing.gguf")
	if _, err := EnsureLocalModel(context.Background(), srv, nil); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing model_path error = %v, want not-exist", err)
	}

	srv.ModelPath = t.TempDir()
	if _, err := EnsureLocalModel(context.Background(), srv, nil); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("directory model_path error = %v", err)
	}

	srv.ModelPath = filepath.Join(t.TempDir(), "model.gguf")
	if err := os.WriteFile(srv.ModelPath, []byte("GGUF"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := EnsureLocalModel(context.Background(), srv, nil)
	if err != nil || got != srv.ModelPath {
		t.Errorf("EnsureLocalModel = %q, %v; want %q", got, err, srv.ModelPath)
	}
}

func TestLocalModel_ReportsManagedStateWithoutDownloading(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	srv := config.ServerConfig{Backend: config.BackendLocal, Model: models.DefaultLocalModel}
	info, err := LocalModel(srv)
	if err != nil {
		t.Fatal(err)
	}
	spec := models.LocalModels[models.DefaultLocalModel]
	if !info.Managed || info.Present || info.Size != spec.Size {
		t.Errorf("LocalModel = %+v, want managed, absent, size %d", info, spec.Size)
	}
	if want := filepath.Join(ManagedModelDir(), spec.File); info.Path != want {
		t.Errorf("Path = %q, want %q", info.Path, want)
	}
	if entries, _ := os.ReadDir(ManagedModelDir()); len(entries) != 0 {
		t.Errorf("status check created files: %v", entries)
	}
}

func TestLocalModelIdentity(t *testing.T) {
	base := config.ServerConfig{Backend: config.BackendLocal, Model: models.DefaultLocalModel}
	managed := LocalModelIdentity(base)
	if !strings.HasPrefix(managed, models.DefaultLocalModel+"@local-sha256-") {
		t.Errorf("managed identity = %q", managed)
	}
	if managed != LocalModelIdentity(base) {
		t.Error("managed identity is not stable")
	}

	dir := t.TempDir()
	write := func(name string, content []byte) config.ServerConfig {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, content, 0o644); err != nil {
			t.Fatal(err)
		}
		s := base
		s.ModelPath = p
		return s
	}
	content := bytes.Repeat([]byte{1}, 9<<20)
	a := write("a.gguf", content)
	modified := append([]byte(nil), content...)
	modified[len(modified)-1] = 2 // same size, different tensor data at the end
	b := write("b.gguf", modified)

	idA, idB := LocalModelIdentity(a), LocalModelIdentity(b)
	if idA == managed || idB == managed {
		t.Error("user file shares the managed artifact identity")
	}
	if idA == idB {
		t.Error("different model files produced the same identity")
	}
	if idA != LocalModelIdentity(write("c.gguf", content)) {
		t.Error("identical model files produced different identities")
	}
}

func TestLocalModelIdentity_CoversEntireFile(t *testing.T) {
	dir := t.TempDir()
	content := make([]byte, 20<<20)
	for i := range content {
		content[i] = byte(i * 7)
	}
	modified := append([]byte(nil), content...)
	// Inside the tensor data, between the regions a sampling fingerprint
	// would read.
	modified[(8<<20)+(200<<10)] ^= 0xff
	srv := func(name string, b []byte) config.ServerConfig {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return config.ServerConfig{Backend: config.BackendLocal, Model: models.DefaultLocalModel, ModelPath: p}
	}
	if LocalModelIdentity(srv("a.gguf", content)) == LocalModelIdentity(srv("b.gguf", modified)) {
		t.Error("files differing outside sampled regions share an identity")
	}
}

func TestLocalModelIdentity_DetectsReplacedFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "model.gguf")
	if err := os.WriteFile(p, []byte("GGUF weights A"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := config.ServerConfig{Backend: config.BackendLocal, Model: models.DefaultLocalModel, ModelPath: p}
	before := LocalModelIdentity(srv)
	replacement := p + ".new"
	if err := os.WriteFile(replacement, []byte("GGUF weights B"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, p); err != nil {
		t.Fatal(err)
	}
	if LocalModelIdentity(srv) == before {
		t.Error("identity did not change after the model file was replaced")
	}
}

func TestModelArtifact_ReplacesCorruptedSameSizeInstall(t *testing.T) {
	content := modelContent()
	srv := &rangeServer{content: content}
	ts := httptest.NewServer(srv)
	defer ts.Close()
	a := testArtifact(t, ts.URL, content)

	corrupt := append([]byte(nil), content...)
	corrupt[len(corrupt)/2] ^= 0xff
	if err := os.WriteFile(a.Path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.ensure(context.Background(), ts.Client(), nil); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if got, _ := os.ReadFile(a.Path); !bytes.Equal(got, content) {
		t.Fatal("corrupted same-size install was accepted")
	}

	// Corruption after a verified install is detected as well.
	if err := os.WriteFile(a.Path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.ensure(context.Background(), ts.Client(), nil); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if got, _ := os.ReadFile(a.Path); !bytes.Equal(got, content) {
		t.Fatal("corruption after a verified install was accepted")
	}
}

// replacePreservingTimestamp atomically replaces path with content, restoring
// the original modification time.
func replacePreservingTimestamp(t *testing.T, path string, content []byte) {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	tmp := path + ".replacement"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(tmp, st.ModTime(), st.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestFileDigest_DetectsChangesWithPreservedTimestamp(t *testing.T) {
	for _, tc := range []struct {
		name    string
		replace func(t *testing.T, path string, content []byte)
	}{
		{"atomic replacement", replacePreservingTimestamp},
		{"in-place rewrite", func(t *testing.T, path string, content []byte) {
			st, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			// Let coarse filesystem clocks advance the status-change time.
			time.Sleep(50 * time.Millisecond)
			if err := os.WriteFile(path, content, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(path, st.ModTime(), st.ModTime()); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "model.gguf")
			if err := os.WriteFile(path, []byte("GGUF AAAA"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := fileDigest(path); err != nil {
				t.Fatal(err)
			}
			tc.replace(t, path, []byte("GGUF BBBB"))

			// Both the in-memory and the persisted (other process) cache.
			for _, cache := range []string{"memory", "disk"} {
				if cache == "disk" {
					digests.Delete(path)
				}
				got, err := fileDigest(path)
				if err != nil {
					t.Fatal(err)
				}
				if want := sha256Hex([]byte("GGUF BBBB")); got != want {
					t.Errorf("%s cache: digest = %s, want %s", cache, got, want)
				}
			}
		})
	}
}

func TestFileDigest_ReusesPersistedDigestForUnchangedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model.gguf")
	if err := os.WriteFile(path, []byte("GGUF AAAA"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fileDigest(path); err != nil {
		t.Fatal(err)
	}
	record, err := os.ReadFile(digestRecordPath(path))
	if err != nil {
		t.Skipf("file versions are not supported on this platform: %v", err)
	}

	// A later process reads the persisted record instead of rehashing.
	fake := strings.Repeat("ab", sha256.Size)
	fields := strings.Fields(string(record))
	if err := os.WriteFile(digestRecordPath(path), []byte(fields[0]+" sha256 "+fake+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	digests.Delete(path)
	if got, _ := fileDigest(path); got != fake {
		t.Errorf("digest = %s, want persisted %s", got, fake)
	}
}

func TestModelArtifact_RejectsReplacementWithPreservedTimestamp(t *testing.T) {
	content := []byte("GGUF AAAA")
	a := testArtifact(t, "http://127.0.0.1:1/unreachable", content)
	if err := os.WriteFile(a.Path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.ensure(context.Background(), http.DefaultClient, nil); err != nil {
		t.Fatal(err)
	}

	replacePreservingTimestamp(t, a.Path, []byte("GGUF BBBB"))
	for _, cache := range []string{"memory", "disk"} {
		if cache == "disk" {
			digests.Delete(a.Path)
		}
		if a.verified() {
			t.Errorf("%s cache: replaced content verified against the pinned checksum", cache)
		}
	}
	if err := a.ensure(context.Background(), http.DefaultClient, nil); err == nil {
		t.Fatal("ensure accepted content that does not match the pinned checksum")
	}
}

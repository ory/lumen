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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/models"
)

// localProfileVersion identifies the local backend's preprocessing: GGUF
// tokenization with special tokens, role-aware query instruction, pooling,
// L2 normalization, and truncation at localContextSize tokens. Change it when
// any of these change so stored vectors are not reused across profiles.
const localProfileVersion = "1"

// DownloadProgress reports managed model download progress in bytes.
type DownloadProgress func(done, total int64)

// LocalModelInfo describes the model file a local-backend server uses.
type LocalModelInfo struct {
	// Path is the GGUF file used for inference.
	Path string
	// Managed is true when Path is Lumen's checksum-pinned download.
	Managed bool
	// Present is true when Path exists (for managed files: is complete).
	Present bool
	// Size is the expected (managed) or actual (user-supplied) size in bytes.
	Size int64
}

// ManagedModelDir returns the directory holding managed model downloads.
func ManagedModelDir() string {
	return filepath.Join(config.XDGCacheDir(), "lumen", "models")
}

// localSpec returns the local-backend spec for a server's model.
func localSpec(srv config.ServerConfig) (models.LocalModelSpec, error) {
	name := srv.Model
	if canonical, ok := models.ModelAliases[name]; ok {
		name = canonical
	}
	spec, ok := models.LocalModels[name]
	if !ok {
		return models.LocalModelSpec{}, fmt.Errorf("model %q is not supported by the local backend", srv.Model)
	}
	return spec, nil
}

func managedArtifact(spec models.LocalModelSpec) modelArtifact {
	return modelArtifact{
		URL:    spec.URL,
		SHA256: spec.SHA256,
		Size:   spec.Size,
		Path:   filepath.Join(ManagedModelDir(), spec.File),
	}
}

// LocalModel inspects the model file of a local-backend server without
// downloading or loading it.
func LocalModel(srv config.ServerConfig) (LocalModelInfo, error) {
	spec, err := localSpec(srv)
	if err != nil {
		return LocalModelInfo{}, err
	}
	if srv.ModelPath != "" {
		info := LocalModelInfo{Path: srv.ModelPath}
		if st, err := os.Stat(srv.ModelPath); err == nil && st.Mode().IsRegular() {
			info.Present = true
			info.Size = st.Size()
		}
		return info, nil
	}
	a := managedArtifact(spec)
	return LocalModelInfo{Path: a.Path, Managed: true, Present: a.complete(), Size: a.Size}, nil
}

// EnsureLocalModel returns the GGUF path for a local-backend server. A
// user-supplied model_path must exist. The managed model is downloaded,
// checksum-verified, and atomically installed when missing; a cached copy
// that matches the pinned digest is used without network access, and a
// corrupted one is replaced.
func EnsureLocalModel(ctx context.Context, srv config.ServerConfig, progress DownloadProgress) (string, error) {
	spec, err := localSpec(srv)
	if err != nil {
		return "", err
	}
	if srv.ModelPath != "" {
		st, err := os.Stat(srv.ModelPath)
		if err != nil {
			return "", fmt.Errorf("local model file: %w", err)
		}
		if !st.Mode().IsRegular() {
			return "", fmt.Errorf("local model file %s is not a regular file", srv.ModelPath)
		}
		return srv.ModelPath, nil
	}
	a := managedArtifact(spec)
	if err := a.ensure(ctx, http.DefaultClient, progress); err != nil {
		return "", err
	}
	return a.Path, nil
}

// LocalModelIdentity returns the name under which vectors produced by a
// local-backend server are stored. It combines the model name with the
// SHA-256 of the exact artifact and the preprocessing profile, so a different
// quantization, file, or preprocessing never shares a collection with
// incompatible vectors even though the dimensionality is the same. It never
// downloads or loads the model; the managed artifact is identified by its
// pinned digest, which is verified before it is loaded.
func LocalModelIdentity(srv config.ServerConfig) string {
	spec, err := localSpec(srv)
	if err != nil {
		return srv.Model
	}
	artifact := "sha256-" + spec.SHA256[:16]
	if srv.ModelPath != "" {
		digest, err := fileDigest(srv.ModelPath)
		if err != nil {
			digest = "unavailable"
		}
		artifact = "file-" + digest[:min(len(digest), 16)]
	}
	return srv.Model + "@local-" + artifact + "-p" + localProfileVersion
}

type digestEntry struct {
	version string
	digest  string
}

var digests sync.Map // path -> digestEntry

// fileDigest returns the SHA-256 of a file's entire content. Hashing a
// multi-gigabyte model takes seconds, so digests are cached in memory and on
// disk for later processes, keyed by the file's version (see fileVersion).
// Where the version cannot be established, the file is always rehashed.
func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	version, versioned := fileVersion(f)
	if versioned {
		if v, ok := digests.Load(path); ok && v.(digestEntry).version == version {
			return v.(digestEntry).digest, nil
		}
		if b, err := os.ReadFile(digestRecordPath(path)); err == nil {
			if fields := strings.Fields(string(b)); len(fields) == 3 && fields[0] == version && fields[1] == "sha256" && len(fields[2]) == sha256.Size*2 {
				digests.Store(path, digestEntry{version: version, digest: fields[2]})
				return fields[2], nil
			}
		}
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	digest := hex.EncodeToString(h.Sum(nil))
	if after, ok := fileVersion(f); versioned && ok && after == version {
		rememberDigest(path, version, digest)
	}
	return digest, nil
}

// rememberDigest caches the digest of the given version of the file at path.
func rememberDigest(path, version, digest string) {
	digests.Store(path, digestEntry{version: version, digest: digest})
	record := digestRecordPath(path)
	if err := os.MkdirAll(filepath.Dir(record), 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(record), ".digest-*")
	if err != nil {
		return
	}
	_, werr := fmt.Fprintf(tmp, "%s sha256 %s\n", version, digest)
	if cerr := tmp.Close(); werr != nil || cerr != nil || os.Rename(tmp.Name(), record) != nil {
		_ = os.Remove(tmp.Name())
	}
}

// digestRecordPath returns where the digest of the file at path is cached.
func digestRecordPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	sum := sha256.Sum256([]byte(path))
	return filepath.Join(config.XDGCacheDir(), "lumen", "digests", hex.EncodeToString(sum[:16]))
}

// ggufMagic is the first four bytes of every GGUF file.
var ggufMagic = []byte("GGUF")

// checkGGUF verifies that path is a readable GGUF file.
func checkGGUF(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	magic := make([]byte, len(ggufMagic))
	if _, err := io.ReadFull(f, magic); err != nil || !bytes.Equal(magic, ggufMagic) {
		return fmt.Errorf("%s is not a GGUF model file", path)
	}
	return nil
}

// modelArtifact is a checksum-pinned download installed at Path.
type modelArtifact struct {
	URL    string
	SHA256 string
	Size   int64
	Path   string
}

// errChecksum marks a download whose content does not match the pinned digest.
var errChecksum = errors.New("checksum mismatch")

const downloadAttempts = 3

// complete reports whether the installed file exists with the pinned size.
// It is a cheap status check; verified also checks the content.
func (a modelArtifact) complete() bool {
	st, err := os.Stat(a.Path)
	return err == nil && st.Mode().IsRegular() && st.Size() == a.Size
}

// verified reports whether the installed file matches the pinned digest. The
// digest is cached per file version, so an unchanged file is hashed once.
func (a modelArtifact) verified() bool {
	if !a.complete() {
		return false
	}
	digest, err := fileDigest(a.Path)
	return err == nil && digest == a.SHA256
}

// ensure installs the artifact unless it is already complete. Concurrent
// callers, including other processes, are serialized by a lock file; the
// download is written to Path+".partial" (resumed when possible), verified,
// and renamed into place, so an interrupted download is never mistaken for a
// complete model.
func (a modelArtifact) ensure(ctx context.Context, client *http.Client, progress DownloadProgress) error {
	if a.verified() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(a.Path), 0o755); err != nil {
		return fmt.Errorf("create model directory: %w", err)
	}
	lock, err := indexlock.Acquire(ctx, a.Path+".lock")
	if err != nil {
		return fmt.Errorf("wait for model download lock: %w", err)
	}
	defer lock.Release()

	// Another process may have finished while we waited.
	if a.verified() {
		return nil
	}
	if _, err := os.Stat(a.Path); err == nil {
		// Truncated, corrupted, or replaced. Managed files are ours to replace.
		if err := os.Remove(a.Path); err != nil {
			return fmt.Errorf("remove incomplete model file: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt < downloadAttempts; attempt++ {
		lastErr = a.download(ctx, client, progress)
		if lastErr == nil || errors.Is(lastErr, errChecksum) || ctx.Err() != nil {
			break
		}
	}
	if lastErr != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("download %s: %w (partial download kept for resume)", a.URL, ctx.Err())
		}
		return fmt.Errorf("download %s: %w", a.URL, lastErr)
	}
	return nil
}

// download fetches the remaining bytes of the artifact into the partial file,
// verifies the complete content, and renames it into place.
func (a modelArtifact) download(ctx context.Context, client *http.Client, progress DownloadProgress) error {
	partial := a.Path + ".partial"
	f, err := os.OpenFile(partial, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("open partial download: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	offset, err := io.Copy(h, f)
	if err != nil {
		return fmt.Errorf("read partial download: %w", err)
	}
	if offset > a.Size {
		if err := f.Truncate(0); err != nil {
			return fmt.Errorf("reset partial download: %w", err)
		}
		h.Reset()
		offset = 0
	}

	if offset < a.Size {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
		if err != nil {
			return err
		}
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		switch {
		case resp.StatusCode == http.StatusPartialContent && offset > 0:
		case resp.StatusCode == http.StatusOK:
			// Full content (server ignored the range): start over.
			if err := f.Truncate(0); err != nil {
				return fmt.Errorf("reset partial download: %w", err)
			}
			h.Reset()
			offset = 0
		default:
			return fmt.Errorf("unexpected HTTP status %s", resp.Status)
		}
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return err
		}

		pw := &progressWriter{done: offset, total: a.Size, report: progress}
		pw.emit()
		remaining := a.Size - offset
		n, err := io.Copy(io.MultiWriter(f, h, pw), io.LimitReader(resp.Body, remaining))
		if err != nil {
			return err
		}
		if n < remaining {
			return fmt.Errorf("download ended after %d of %d bytes", offset+n, a.Size)
		}
		pw.emit()
	}

	if got := hex.EncodeToString(h.Sum(nil)); got != a.SHA256 {
		_ = f.Close()
		_ = os.Remove(partial)
		return fmt.Errorf("%w: got sha256 %s, want %s", errChecksum, got, a.SHA256)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync model file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close model file: %w", err)
	}
	if err := os.Rename(partial, a.Path); err != nil {
		return fmt.Errorf("install model file: %w", err)
	}
	if installed, err := os.Open(a.Path); err == nil {
		if version, ok := fileVersion(installed); ok {
			rememberDigest(a.Path, version, a.SHA256)
		}
		_ = installed.Close()
	}
	return nil
}

// progressWriter reports progress at most once per percent.
type progressWriter struct {
	done, total, last int64
	report            DownloadProgress
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if p.total > 0 && (p.done-p.last)*100 >= p.total {
		p.emit()
	}
	return len(b), nil
}

func (p *progressWriter) emit() {
	p.last = p.done
	if p.report != nil {
		p.report(p.done, p.total)
	}
}

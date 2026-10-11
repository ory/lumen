// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

//go:build worktree_e2e

package worktrees

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// Scenario 1: identical linked worktrees share one repository/profile
// collection. New siblings hash their files once but parse and embed nothing.
func TestIdenticalWorktreesShareOneCollection(t *testing.T) {
	for _, backend := range []string{"ollama", "lmstudio"} {
		t.Run(backend, func(t *testing.T) {
			e := newEnv(t, withBackend(backend))
			main := e.repo("main repo", baseFiles())
			sibling := e.worktree(main, filepath.Join(filepath.Dir(main), "sibling wt"), "-b", "sibling")
			third := e.worktree(main, filepath.Join(filepath.Dir(main), "third"), "-b", "third")

			// Positive control: a cold index parses and embeds everything.
			m, l := e.p.mark(), e.logMark()
			e.index(main)
			cold, coldDocs := e.work(l, main), e.p.documents(m)
			if cold.hashed != baseFileCount || cold.parsed != baseFileCount || len(coldDocs) == 0 {
				t.Fatalf("[A:cold-control] cold index hashed=%d parsed=%d docs=%d", cold.hashed, cold.parsed, len(coldDocs))
			}
			requireEachOnce(t, "cold-each-once", coldDocs)

			m, l = e.p.mark(), e.logMark()
			e.index(sibling)
			w := e.work(l, sibling)
			if w.hashed != baseFileCount {
				t.Fatalf("[A:sibling-initial-hash] new sibling hashed %d files, want %d", w.hashed, baseFileCount)
			}
			if w.parsed != 0 {
				t.Fatalf("[A:sibling-zero-parse] identical sibling parsed %d files", w.parsed)
			}
			if w.attached != baseFileCount {
				t.Fatalf("[A:sibling-attached] identical sibling attached %d revisions, want %d", w.attached, baseFileCount)
			}
			if docs := e.p.documents(m); len(docs) != 0 {
				t.Fatalf("[A:sibling-zero-embed] identical sibling embedded %d inputs", len(docs))
			}

			// The MCP server is the path agent hosts use for a new worktree.
			s := e.mcp()
			m, l = e.p.mark(), e.logMark()
			files, _ := s.search("how are totals computed", third, third)
			w = e.work(l, third)
			if w.parsed != 0 || len(e.p.documents(m)) != 0 {
				t.Fatalf("[A:sibling-zero-parse] MCP sibling parsed=%d embedded=%d", w.parsed, len(e.p.documents(m)))
			}
			if q := e.p.queryRequests(m); q != 1 {
				t.Fatalf("[A:query-count] MCP search issued %d query requests, want 1", q)
			}
			if !slices.Equal(files, []string{"alpha.go", "beta.go", "pkg/delta.go", "pkg/gamma.go"}) {
				t.Fatalf("[A:sibling-results] MCP sibling results = %v", files)
			}
			s.close()

			snap := e.inspect(e.onlyCollection())
			if len(snap.projects) != 3 {
				t.Fatalf("[A:memberships] projects = %v", snap.projects)
			}
			want := revisionSet(snap.members[main])
			for _, project := range []string{main, sibling, third} {
				members, ok := snap.members[project]
				if !ok || len(members) != baseFileCount {
					t.Fatalf("[A:memberships] %s has %d memberships", project, len(members))
				}
				if got := revisionSet(members); !mapsEqual(got, want) {
					t.Fatalf("[A:shared-revisions] %s revisions %v differ from %v", project, got, want)
				}
			}
			if distinct := len(multiplicity(coldDocs)); snap.vectors != distinct || snap.chunkDefs != distinct {
				t.Fatalf("[A:shared-vectors] vectors=%d chunkDefs=%d, want %d", snap.vectors, snap.chunkDefs, distinct)
			}
			requireNoGarbage(t, "no-garbage", snap)
		})
	}
}

// Scenario 2: warm searches inside the freshness TTL do not scan; after TTL
// expiry or a restart, stable files reuse hashes and nothing is parsed or
// embedded. Revision and vector IDs remain stable.
func TestWarmSearchesAndRestartsReuseWork(t *testing.T) {
	const ttl = 4 * time.Second
	e := newEnv(t, withEnv("LUMEN_FRESHNESS_TTL="+ttl.String()))
	main := e.repo("warm", baseFiles())
	time.Sleep(settleDelay)
	e.index(main)
	indexedAt := time.Now()
	db := e.onlyCollection()
	before := e.inspect(db)

	s := e.mcp()
	m, l := e.p.mark(), e.logMark()
	s.search("first warm query", main, main)
	s.search("second warm query", main, main)
	if time.Since(indexedAt) < ttl {
		if w := e.work(l, main); w.scans != 0 {
			t.Fatalf("[A:ttl-no-scan] searches inside the freshness TTL scanned %d times", w.scans)
		}
	} else {
		t.Logf("SKIPPED [A:ttl-no-scan]: searches finished %s after indexing, beyond the %s TTL", time.Since(indexedAt), ttl)
	}
	if q, d := e.p.queryRequests(m), len(e.p.documents(m)); q != 2 || d != 0 {
		t.Fatalf("[A:warm-zero-embed] warm searches: queries=%d documents=%d", q, d)
	}

	time.Sleep(ttl + 200*time.Millisecond)
	m, l = e.p.mark(), e.logMark()
	s.search("after ttl query", main, main)
	w := e.work(l, main)
	if w.scans != 1 || w.hashed != 0 || w.reused != baseFileCount {
		t.Fatalf("[A:warm-reuse-hashes] after TTL: scans=%d hashed=%d reused=%d", w.scans, w.hashed, w.reused)
	}
	if w.parsed != 0 || w.passes != 0 || len(e.p.documents(m)) != 0 {
		t.Fatalf("[A:warm-zero-embed] after TTL: parsed=%d passes=%d documents=%d", w.parsed, w.passes, len(e.p.documents(m)))
	}
	s.close()

	// Restart: a new server process loads the scan sidecar.
	s = e.mcp()
	m, l = e.p.mark(), e.logMark()
	s.search("after restart query", main, main)
	w = e.work(l, main)
	if w.scans != 1 || w.hashed != 0 || w.reused != baseFileCount {
		t.Fatalf("[A:warm-reuse-hashes] after restart: scans=%d hashed=%d reused=%d", w.scans, w.hashed, w.reused)
	}
	if w.parsed != 0 || len(e.p.documents(m)) != 0 {
		t.Fatalf("[A:warm-zero-embed] after restart: parsed=%d documents=%d", w.parsed, len(e.p.documents(m)))
	}
	s.close()

	l = e.logMark()
	r := e.index(main)
	if !strings.Contains(r.stdout, "already up to date") {
		t.Fatalf("[A:warm-zero-embed] CLI restart reindexed:\n%s", r.stdout)
	}
	if w := e.work(l, main); w.hashed != 0 || w.reused != baseFileCount {
		t.Fatalf("[A:warm-reuse-hashes] CLI restart hashed=%d reused=%d", w.hashed, w.reused)
	}
	after := e.inspect(db)
	if !mapsEqual(after.members[main], before.members[main]) || !mapsEqual(after.vectorIDs, before.vectorIDs) {
		t.Fatalf("[A:stable-ids] revision or vector IDs changed across warm restarts")
	}
}

// Scenario 3: a one-file edit parses only that file and embeds only its
// genuinely missing inputs; switching branches back to a retained revision
// needs no parsing or embedding.
func TestEditsAndBranchSwitchesDoMinimalWork(t *testing.T) {
	e := newEnv(t)
	main := e.repo("edits", baseFiles())
	e.index(main)
	db := e.onlyCollection()
	cold := multiplicity(e.p.documents(0))
	betaChunks := 0
	for in := range cold {
		if strings.HasPrefix(in, "// beta.go\n") {
			betaChunks++
		}
	}
	if betaChunks < 3 {
		t.Fatalf("[A:cold-control] beta.go produced %d chunks", betaChunks)
	}
	time.Sleep(settleDelay)

	e.git(main, "checkout", "-q", "-b", "feature")
	editFile(t, filepath.Join(main, "beta.go"), "return total // BetaTwo\n", "return total * 2 // BetaTwo edited\n")
	// Rendezvous results must not mask the durable vector reuse path.
	e.expireRendezvous(db)
	m, l := e.p.mark(), e.logMark()
	e.index(main)
	w := e.work(l, main)
	if w.hashed != 1 || w.reused != baseFileCount-1 {
		t.Fatalf("[A:edit-hash-one] one-file edit hashed=%d reused=%d", w.hashed, w.reused)
	}
	if w.parsed != 1 {
		t.Fatalf("[A:edit-parse-one] one-file edit parsed %d files", w.parsed)
	}
	if w.attached != 0 {
		t.Fatalf("[A:edit-no-unchanged-work] one-file edit revisited %d unchanged files", w.attached)
	}
	docs := e.p.documents(m)
	requireEachOnce(t, "edit-each-once", docs)
	if len(docs) == 0 {
		t.Fatalf("[A:edit-embed-only-missing] edited chunk was not embedded")
	}
	for _, in := range docs {
		if !strings.HasPrefix(in, "// beta.go\n") || !strings.Contains(in, "BetaTwo edited") || cold[in] != 0 {
			t.Fatalf("[A:edit-embed-only-missing] embedded an input that was not missing: %q", firstLine(in))
		}
	}
	if len(docs) >= betaChunks {
		t.Fatalf("[A:edit-embed-only-missing] embedded %d of beta.go's %d chunks", len(docs), betaChunks)
	}
	e.git(main, "commit", "-q", "-am", "edit beta")

	for _, branch := range []string{"main", "feature"} {
		e.expireRendezvous(db)
		e.git(main, "checkout", "-q", branch)
		m, l = e.p.mark(), e.logMark()
		e.index(main)
		w = e.work(l, main)
		if w.parsed != 0 {
			t.Fatalf("[A:switch-no-parse] switching to %s parsed %d files", branch, w.parsed)
		}
		if w.attached != 1 {
			t.Fatalf("[A:switch-attach] switching to %s attached %d revisions, want 1", branch, w.attached)
		}
		if docs := e.p.documents(m); len(docs) != 0 {
			t.Fatalf("[A:switch-no-embed] switching to %s embedded %d inputs", branch, len(docs))
		}
	}
	snap := e.inspect(db)
	if snap.retained != 1 || snap.revisions != baseFileCount+1 {
		t.Fatalf("[A:switch-retained] retained=%d revisions=%d", snap.retained, snap.revisions)
	}
	requireNoGarbage(t, "no-garbage", snap)
}

// Scenario 4: uncommitted edits, additions, deletions, renames, ignore-rule
// changes, and same-size edits with restored mtimes across restarts.
func TestFilesystemChangesAreDetected(t *testing.T) {
	e := newEnv(t)
	main := e.repo("filesystem", baseFiles())
	e.index(main)
	db := e.onlyCollection()
	members := func() []string {
		var paths []string
		for rel := range e.inspect(db).members[main] {
			paths = append(paths, rel)
		}
		slices.Sort(paths)
		return paths
	}

	writeFiles(t, main, map[string]string{"epsilon.go": goFile("fixture", "EpsilonOne")})
	l := e.logMark()
	e.index(main)
	if w := e.work(l, main); w.parsed != 1 || !slices.Contains(members(), "epsilon.go") {
		t.Fatalf("[A:fs-addition] untracked addition parsed=%d members=%v", w.parsed, members())
	}

	if err := os.Remove(filepath.Join(main, "alpha.go")); err != nil {
		t.Fatal(err)
	}
	m, l := e.p.mark(), e.logMark()
	e.index(main)
	if w := e.work(l, main); w.parsed != 0 || len(e.p.documents(m)) != 0 || slices.Contains(members(), "alpha.go") {
		t.Fatalf("[A:fs-deletion] deletion parsed=%d members=%v", w.parsed, members())
	}

	e.git(main, "mv", "beta.go", "beta_renamed.go")
	m, l = e.p.mark(), e.logMark()
	e.index(main)
	w := e.work(l, main)
	docs := e.p.documents(m)
	if w.parsed != 1 || len(docs) < 3 || slices.Contains(members(), "beta.go") || !slices.Contains(members(), "beta_renamed.go") {
		t.Fatalf("[A:fs-rename] rename parsed=%d docs=%d members=%v", w.parsed, len(docs), members())
	}
	for _, in := range docs {
		// Embedding inputs are filepath-aware, so a rename embeds new inputs.
		if !strings.HasPrefix(in, "// beta_renamed.go\n") {
			t.Fatalf("[A:fs-rename] unexpected input %q", firstLine(in))
		}
	}

	writeFiles(t, main, map[string]string{".gitignore": "*.log\npkg/\n"})
	m, l = e.p.mark(), e.logMark()
	e.index(main)
	if got := members(); slices.Contains(got, "pkg/gamma.go") || slices.Contains(got, "pkg/delta.go") {
		t.Fatalf("[A:fs-ignore] ignored files remain members: %v", got)
	}
	if w := e.work(l, main); w.parsed != 0 || len(e.p.documents(m)) != 0 {
		t.Fatalf("[A:fs-ignore] ignoring files parsed=%d embedded=%d", w.parsed, len(e.p.documents(m)))
	}
	writeFiles(t, main, map[string]string{".gitignore": "*.log\n"})
	m, l = e.p.mark(), e.logMark()
	e.index(main)
	if w := e.work(l, main); w.parsed != 0 || w.attached != 2 || len(e.p.documents(m)) != 0 || !slices.Contains(members(), "pkg/gamma.go") {
		t.Fatalf("[A:fs-ignore] un-ignoring parsed=%d attached=%d members=%v", w.parsed, w.attached, members())
	}

	// Same-size edit with the original mtime, observed by a new process after
	// the timestamp race window: only identity/ctime reveal the change.
	time.Sleep(settleDelay)
	e.index(main)
	gamma := filepath.Join(main, "pkg", "gamma.go")
	info, err := os.Stat(gamma)
	if err != nil {
		t.Fatal(err)
	}
	editFile(t, gamma, "total += i * 56", "total += i * 65")
	if after, _ := os.Stat(gamma); after.Size() != info.Size() {
		t.Fatal("fixture edit changed the file size")
	}
	if err := os.Chtimes(gamma, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(settleDelay)
	l = e.logMark()
	e.index(main)
	if w := e.work(l, main); w.parsed != 1 || w.hashed != 1 {
		t.Fatalf("[A:fs-same-size-edit] same-size edit with restored mtime: hashed=%d parsed=%d", w.hashed, w.parsed)
	}

	s := e.mcp()
	files, _ := s.search("filesystem state", main, main)
	if want := []string{"beta_renamed.go", "epsilon.go", "pkg/delta.go", "pkg/gamma.go"}; !slices.Equal(files, want) {
		t.Fatalf("[A:fs-search] results %v, want %v", files, want)
	}
	requireNoGarbage(t, "no-garbage", e.inspect(db))
}

// Scenario 5: external and nested worktrees, detached HEAD, subdirectory
// working directories, paths with spaces, and symlink aliases. Separate clones
// and profiles remain isolated.
func TestTopologiesKeepWorktreesIsolated(t *testing.T) {
	e := newEnv(t)
	main := e.repo("topology main", baseFiles())
	parent := filepath.Dir(main)
	nested := e.worktree(main, filepath.Join(main, ".worktrees", "nested"), "-b", "nested")
	writeFiles(t, nested, map[string]string{"nested_only.go": goFile("fixture", "NestedOnly")})
	external := e.worktree(main, filepath.Join(parent, "external wt"), "-b", "external")
	writeFiles(t, external, map[string]string{"external_only.go": goFile("fixture", "ExternalOnly")})
	detached := e.worktree(main, filepath.Join(parent, "detached"), "--detach")
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(main, alias); err != nil {
		t.Fatal(err)
	}
	base := []string{"alpha.go", "beta.go", "pkg/delta.go", "pkg/gamma.go"}

	e.index(main)
	db := e.onlyCollection()
	for rel := range e.inspect(db).members[main] {
		if strings.HasPrefix(rel, ".worktrees/") {
			t.Fatalf("[A:main-excludes-nested] main project indexed nested worktree file %s", rel)
		}
	}

	s := e.mcp()
	pkg := filepath.Join(main, "pkg")
	m, l := e.p.mark(), e.logMark()
	files, _ := s.search("subdirectory query", pkg, pkg)
	if w := e.work(l, ""); w.parsed != 0 || len(e.p.documents(m)) != 0 {
		t.Fatalf("[A:subdir-reuses-root] subdirectory cwd parsed=%d embedded=%d", w.parsed, len(e.p.documents(m)))
	}
	if !slices.Equal(files, []string{"pkg/delta.go", "pkg/gamma.go"}) && !slices.Equal(files, []string{"delta.go", "gamma.go"}) {
		t.Fatalf("[A:subdir-results] subdirectory results = %v", files)
	}

	m, l = e.p.mark(), e.logMark()
	files, _ = s.search("nested worktree query", nested, main)
	if w := e.work(l, ""); w.parsed != 1 || w.attached != baseFileCount || !allFrom(e.p.documents(m), "nested_only.go") {
		t.Fatalf("[A:nested-own-root] nested worktree parsed=%d attached=%d documents=%d", w.parsed, w.attached, len(e.p.documents(m)))
	}
	if want := append([]string{"nested_only.go"}, base...); !sameSet(files, want) {
		t.Fatalf("[A:nested-own-root] nested results = %v, want %v", files, want)
	}

	files, _ = s.search("main isolation query", main, main)
	if !sameSet(files, base) {
		t.Fatalf("[A:no-sibling-leak] main results leak sibling content: %v", files)
	}

	m, l = e.p.mark(), e.logMark()
	files, _ = s.search("external worktree query", external, external)
	if !sameSet(files, append([]string{"external_only.go"}, base...)) {
		t.Fatalf("[A:no-sibling-leak] external results = %v", files)
	}
	if w := e.work(l, external); w.parsed != 1 || !allFrom(e.p.documents(m), "external_only.go") {
		t.Fatalf("[A:external-results] external parsed %d files, want 1; documents=%d", w.parsed, len(e.p.documents(m)))
	}

	m, l = e.p.mark(), e.logMark()
	files, _ = s.search("detached query", detached, detached)
	if w := e.work(l, detached); w.parsed != 0 || len(e.p.documents(m)) != 0 || !sameSet(files, base) {
		t.Fatalf("[A:detached-shares] detached HEAD parsed=%d embedded=%d results=%v", w.parsed, len(e.p.documents(m)), files)
	}
	s.close()

	// A fresh server reached through a symlink alias reuses the canonical
	// worktree: no new membership, parsing, or embedding.
	s = e.mcp()
	m, l = e.p.mark(), e.logMark()
	files, _ = s.search("alias query", alias, alias)
	if w := e.work(l, ""); w.parsed != 0 || len(e.p.documents(m)) != 0 || !sameSet(files, base) {
		t.Fatalf("[A:alias-no-rework] alias parsed=%d embedded=%d results=%v", w.parsed, len(e.p.documents(m)), files)
	}
	s.close()
	snap := e.inspect(db)
	want := map[string]bool{main: true, nested: true, external: true, detached: true}
	for project := range snap.projects {
		if !want[project] {
			t.Fatalf("[A:canonical-roots] unexpected membership %s (all: %v)", project, keys(snap.projects))
		}
	}

	// Separate clones and profiles never share a collection.
	clone := filepath.Join(parent, "clone")
	e.git(parent, "clone", "-q", main, clone)
	l = e.logMark()
	e.index(clone)
	if w := e.work(l, clone); w.parsed != baseFileCount {
		t.Fatalf("[A:clone-isolated] clone parsed %d files, want %d", w.parsed, baseFileCount)
	}
	if n := len(e.collections()); n != 2 {
		t.Fatalf("[A:clone-isolated] %d collections after indexing a clone, want 2", n)
	}
	e.extra = append(e.extra, "LUMEN_MAX_CHUNK_TOKENS=256")
	e.index(main)
	if n := len(e.collections()); n != 3 {
		t.Fatalf("[A:profile-isolated] %d collections after a profile change, want 3", n)
	}
	if got := e.inspect(db); !mapsEqual(got.members[main], snap.members[main]) {
		t.Fatalf("[A:profile-isolated] profile change modified the original collection")
	}
}

// Scenario 6a: independent cold processes compute each unique document input
// once, and same-worktree startup avoids duplicate passes.
func TestConcurrentProcessesComputeOnce(t *testing.T) {
	e := newEnv(t)
	main := e.repo("concurrent", baseFiles())
	wt1 := e.worktree(main, filepath.Join(filepath.Dir(main), "c1"), "-b", "c1")
	wt2 := e.worktree(main, filepath.Join(filepath.Dir(main), "c2"), "-b", "c2")

	g := e.p.block(documentsMentioning(""))
	m, l := e.p.mark(), e.logMark()
	a, b := e.start("index", wt1), e.start("index", wt2)
	waitGate(t, g, "first document request")
	for _, wt := range []string{wt1, wt2} {
		e.waitEvent(l, time.Minute, func(ev event) bool { return ev.msg() == "scan completed" && ev.str("project") == wt })
	}
	// Both processes have scanned; let the second reach its claims while the
	// first request is still held open.
	time.Sleep(500 * time.Millisecond)
	g.Release()
	if err := a.wait(t); err != nil {
		t.Fatalf("index %s: %v\n%s", wt1, err, a.stderr.String())
	}
	if err := b.wait(t); err != nil {
		t.Fatalf("index %s: %v\n%s", wt2, err, b.stderr.String())
	}
	docs := e.p.documents(m)
	requireEachOnce(t, "concurrent-cold-once", docs)
	snap := e.inspect(e.onlyCollection())
	if snap.vectors != len(docs) || !mapsEqual(revisionSet(snap.members[wt1]), revisionSet(snap.members[wt2])) || len(snap.members[wt1]) != baseFileCount {
		t.Fatalf("[A:concurrent-cold-once] vectors=%d docs=%d members=%d/%d", snap.vectors, len(docs), len(snap.members[wt1]), len(snap.members[wt2]))
	}
	if peak := e.p.peakInFlight(); peak != 1 {
		t.Fatalf("[A:single-flight] peak in-flight embedding requests = %d", peak)
	}

	// Same worktree: CLI, SessionStart background child, and MCP all start
	// while the first indexer holds its request open.
	same := e.repo("same", map[string]string{"same_a.go": goFile("same", "SameOne", "SameTwo"), "same_b.go": goFile("same", "SameThree")})
	g = e.p.block(documentsMentioning("// same_"))
	m, l = e.p.mark(), e.logMark()
	first := e.start("index", same)
	waitGate(t, g, "same-worktree document request")
	second := e.lumen("index", same)
	if second.err != nil || !strings.Contains(second.stderr, "already running") {
		t.Fatalf("[A:same-worktree-skip] second CLI indexer was not skipped: err=%v\n%s", second.err, second.stderr)
	}
	_, hookPID := e.sessionStart(same)
	ev := e.waitEvent(l, time.Minute, func(ev event) bool {
		return ev.msg() == "index skipped: another indexer is already running" && ev.str("project") == same
	})
	if ev.pid() == hookPID || ev.pid() == first.cmd.Process.Pid {
		t.Fatalf("[A:same-worktree-skip] skip was not reported by the spawned hook child")
	}
	s := e.mcp()
	_, text := s.search("same worktree query", same, same)
	if !strings.Contains(text, "being updated in the background") {
		t.Fatalf("[A:same-worktree-skip] MCP did not defer to the running indexer:\n%s", text)
	}
	g.Release()
	if err := first.wait(t); err != nil {
		t.Fatalf("index %s: %v", same, err)
	}
	e.waitNoIndexers()
	docs = e.p.documents(m)
	requireEachOnce(t, "same-worktree-once", docs)
	if w := e.work(l, same); w.passes != 1 || w.parsed != 2 {
		t.Fatalf("[A:same-worktree-one-pass] passes=%d parsed=%d", w.passes, w.parsed)
	}
}

// Scenario 6b: the real SessionStart hook warms the index through the binary
// it spawns, and a following search in the session does no indexing work.
func TestSessionStartWarmsThroughSpawnedIndexer(t *testing.T) {
	e := newEnv(t)
	main := e.repo("hook", baseFiles())
	m, l := e.p.mark(), e.logMark()
	text, hookPID := e.sessionStart(main)
	if !strings.Contains(text, "indexing in background") {
		t.Fatalf("[A:hook-warms] unexpected hook context: %s", text)
	}
	ev := e.waitEventTagged("hook-warms", l, 2*time.Minute, func(ev event) bool {
		return ev.msg() == "index pass completed" && ev.str("project") == main
	})
	if ev.pid() == hookPID || ev.pid() == 0 {
		t.Fatalf("[A:hook-warms] index pass ran in pid %d, not a spawned child of hook %d", ev.pid(), hookPID)
	}
	e.waitNoIndexers()
	if w := e.work(l, main); w.parsed != baseFileCount || len(e.p.documents(m)) == 0 {
		t.Fatalf("[A:hook-warms] background child parsed=%d embedded=%d", w.parsed, len(e.p.documents(m)))
	}
	if root := e.inspect(e.onlyCollection()).meta[main]["root_hash"]; root == "" {
		t.Fatalf("[A:hook-warms] background child did not complete the index")
	}

	s := e.mcp()
	m, l = e.p.mark(), e.logMark()
	files, _ := s.search("hook warmed query", main, main)
	if w := e.work(l, main); w.scans != 0 || w.parsed != 0 || len(e.p.documents(m)) != 0 {
		t.Fatalf("[A:hook-warm-search] first search after warmup scans=%d parsed=%d embedded=%d", w.scans, w.parsed, len(e.p.documents(m)))
	}
	if len(files) != baseFileCount {
		t.Fatalf("[A:hook-warm-search] results = %v", files)
	}

	l = e.logMark()
	text, _ = e.sessionStart(main)
	if !strings.Contains(text, "Lumen index ready") {
		t.Fatalf("[A:hook-ready] unexpected hook context: %s", text)
	}
	time.Sleep(500 * time.Millisecond)
	e.waitNoIndexers()
	if w := e.work(l, main); w.scans != 0 || w.passes != 0 {
		t.Fatalf("[A:hook-ready] ready session start triggered scans=%d passes=%d", w.scans, w.passes)
	}
}

// Scenario 7: one embedding HTTP request at a time per data home across
// repositories and processes; queries precede the next background request;
// terminated, failed, and expired owners recover without false completion.
func TestComputeSchedulingAcrossProcesses(t *testing.T) {
	e := newEnv(t)
	names := make([]string, 80)
	for i := range names {
		names[i] = fmt.Sprintf("BigFunction%02d", i)
	}
	big := e.repo("big", map[string]string{"big.go": goFile("big", names...)})
	other := e.repo("other", map[string]string{"other.go": goFile("other", "OtherOne", "OtherTwo")})
	small := e.repo("small", baseFiles())
	e.index(small)
	s := e.mcp("LUMEN_FRESHNESS_TTL=10m")
	s.search("warm the search session", small, small)

	m, l := e.p.mark(), e.logMark()
	g := e.p.block(documentsMentioning("// big.go\n"))
	a := e.start("index", big)
	waitGate(t, g, "first big.go request")
	// A different repository indexed by a different process in the same data
	// home must not overlap the held request.
	c := e.start("index", other)
	e.waitEvent(l, time.Minute, func(ev event) bool { return ev.msg() == "scan completed" && ev.str("project") == other })
	time.Sleep(500 * time.Millisecond)
	if peak := e.p.peakInFlight(); peak != 1 {
		t.Fatalf("[A:single-flight] another process's request overlapped the held one: peak in-flight = %d", peak)
	}

	const query = "which function handles the priority query"
	queryDone := make(chan error, 1)
	go func() { queryDone <- s.trySearch(query, small, small) }()
	waitFor(t, 30*time.Second, "query intent", e.queryIntentHeld)
	g.Release()
	if err := <-queryDone; err != nil {
		t.Fatal(err)
	}
	if err := a.wait(t); err != nil {
		t.Fatalf("index big: %v\n%s", err, a.stderr.String())
	}
	if err := c.wait(t); err != nil {
		t.Fatalf("index other: %v\n%s", err, c.stderr.String())
	}
	if peak := e.p.peakInFlight(); peak != 1 {
		t.Fatalf("[A:single-flight] peak in-flight embedding requests = %d", peak)
	}
	reqs := e.p.since(m)
	bigRequests := 0
	for _, r := range reqs {
		if !r.Query && strings.HasPrefix(r.Inputs[0], "// big.go\n") {
			bigRequests++
			if len(r.Inputs) > 32 {
				t.Fatalf("[A:request-size] provider request carried %d inputs", len(r.Inputs))
			}
		}
	}
	if bigRequests < 3 {
		t.Fatalf("[A:multi-request-control] big.go used %d HTTP requests; the scenario needs more than 32 inputs", bigRequests)
	}
	if len(reqs) < 2 || !reqs[1].Query || reqs[1].Inputs[0] != query {
		order := make([]string, len(reqs))
		for i, r := range reqs {
			order[i] = firstLine(r.Inputs[0])
		}
		t.Fatalf("[A:query-before-next-batch] query was not served before the next background request: %v", order)
	}

	// Terminating the owner of a running request releases capacity, and the
	// interrupted index never receives a completed root marker.
	deadNames := make([]string, 40)
	for i := range deadNames {
		deadNames[i] = fmt.Sprintf("DeadFunction%02d", i)
	}
	dead := e.repo("dead", map[string]string{"dead.go": goFile("dead", deadNames...)})
	g = e.p.block(documentsMentioning("// dead.go\n"))
	d := e.start("index", dead)
	waitGate(t, g, "dead.go request")
	_ = d.cmd.Process.Kill()
	_ = d.wait(t)
	started := time.Now()
	if err := s.trySearch("query after the owner died", small, small); err != nil {
		t.Fatalf("[A:killed-owner-releases-capacity] %v", err)
	}
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("[A:killed-owner-releases-capacity] query waited %s after the owner died", elapsed)
	}
	deadDB := e.collectionFor(dead)
	snap := e.inspect(deadDB)
	if snap.meta[dead]["root_hash"] != "" || snap.claimsPending == 0 {
		t.Fatalf("[A:no-false-completion] killed index: root_hash=%q pending claims=%d", snap.meta[dead]["root_hash"], snap.claimsPending)
	}
	// Simulate the two-minute lease passing; the next indexer recovers.
	e.fixtureExec(deadDB, `UPDATE embedding_claims SET expires_at = 1 WHERE result IS NULL`)
	m = e.p.mark()
	e.index(dead)
	if snap = e.inspect(deadDB); snap.meta[dead]["root_hash"] == "" || snap.claimsPending != 0 {
		t.Fatalf("[A:expired-claims-recover] recovery: root_hash=%q pending=%d", snap.meta[dead]["root_hash"], snap.claimsPending)
	}
	requireEachOnce(t, "expired-claims-recover", e.p.documents(m))

	// A failed request releases its claims immediately.
	failing := e.repo("failing", map[string]string{"failing.go": goFile("failing", "FailOne", "FailTwo")})
	e.p.fail(documentsMentioning("// failing.go\n"), http.StatusBadRequest)
	if r := e.lumen("index", failing); r.err == nil {
		t.Fatalf("[A:failure-control] index succeeded despite an injected provider failure")
	}
	failDB := e.collectionFor(failing)
	if snap = e.inspect(failDB); snap.meta[failing]["root_hash"] != "" || snap.claimsPending != 0 {
		t.Fatalf("[A:failed-claims-released] failed index: root_hash=%q pending=%d", snap.meta[failing]["root_hash"], snap.claimsPending)
	}
	e.index(failing)
	if snap = e.inspect(failDB); snap.meta[failing]["root_hash"] == "" {
		t.Fatalf("[A:failed-claims-released] retry did not complete")
	}
	if peak := e.p.peakInFlight(); peak != 1 {
		t.Fatalf("[A:single-flight] peak in-flight embedding requests = %d", peak)
	}
}

// Scenario 8: edits and disposable worktree churn stay within the retention
// policy, active data is protected, and cleanup leaves no garbage or orphan
// scan sidecars. Zero-policy and TTL expiry run through real maintenance.
func TestRetentionAndCleanupBoundStorage(t *testing.T) {
	// Realistic dimensions make vector bytes dominate per-chunk metadata, so
	// the accounting oracle below cannot be satisfied by metadata alone.
	e := newEnv(t, withDims(768))
	main := e.repo("retention", baseFiles())
	e.index(main)
	db := e.onlyCollection()

	edit := func(dir string, n int) {
		t.Helper()
		writeFiles(t, dir, map[string]string{"beta.go": goFile("fixture", "BetaOne", "BetaTwo", fmt.Sprintf("BetaEdit%03d", n))})
	}
	edit(main, 0)
	e.index(main)
	snap := e.inspect(db)
	if snap.retained != 1 || snap.retainedBytes <= 0 {
		t.Fatalf("[A:retention-control] first replacement retained=%d bytes=%d", snap.retained, snap.retainedBytes)
	}
	// Independent oracle: every retained revision is charged at least one
	// int8 vector (dims bytes) per chunk it owns.
	if under := e.countRows(db, `SELECT count(*) FROM revision_cache rc WHERE rc.bytes < ? * (SELECT count(*) FROM chunk_defs cd WHERE cd.file_revision_id = rc.revision_id)`, e.p.dims); under != 0 {
		t.Fatalf("[A:retention-accounting] %d retained revisions are charged less than their vectors", under)
	}
	budget := snap.retainedBytes*2 + snap.retainedBytes/2
	e.setRetention(db, main, budget, time.Hour)

	var physicalHigh int64
	for i := 1; i <= 6; i++ {
		edit(main, i)
		e.index(main)
		if i%2 == 0 {
			// Disposable worktree: index, diverge, then delete it.
			wt := e.worktree(main, filepath.Join(filepath.Dir(main), fmt.Sprintf("disposable-%d", i)), "-b", fmt.Sprintf("disposable-%d", i))
			e.index(wt)
			writeFiles(t, wt, map[string]string{"alpha.go": goFile("fixture", "AlphaOne", fmt.Sprintf("Disposable%03d", i))})
			e.index(wt)
			e.git(main, "worktree", "remove", "--force", wt)
		}
		physicalHigh = max(physicalHigh, physicalBytes(t, db))
		snap = e.inspect(db)
		if snap.retainedBytes > budget || snap.retained > 2 {
			t.Fatalf("[A:retention-budget] step %d retained %d revisions, %d bytes > budget %d", i, snap.retained, snap.retainedBytes, budget)
		}
		active := map[int64]bool{}
		for _, members := range snap.members {
			for _, id := range members {
				active[id] = true
			}
		}
		if snap.revisions != len(active)+snap.retained {
			t.Fatalf("[A:retention-gc] step %d revisions=%d active=%d retained=%d", i, snap.revisions, len(active), snap.retained)
		}
		requireNoGarbage(t, "retention-gc", snap)
	}

	projectLocks, _ := filepath.Glob(db + ".project-*.lock")
	if len(projectLocks) == 0 {
		t.Fatalf("[A:locks-control] no per-project lock files exist before clean")
	}
	e.mustLumen("clean")
	snap = e.inspect(db)
	if len(snap.projects) != 1 || snap.projects[main] == 0 {
		t.Fatalf("[A:cleanup-memberships] projects after clean: %v", keys(snap.projects))
	}
	requireNoGarbage(t, "cleanup-no-garbage", snap)
	if scans := scanFiles(t, db); len(scans) != 1 || filepath.Base(scans[0]) != fmt.Sprintf("index.db.scan-%x.json", sha256.Sum256([]byte(main))) {
		t.Fatalf("[A:no-orphan-scan] scan sidecars after clean: %v", scans)
	}
	for _, lock := range append([]string{db + ".lock"}, projectLocks...) {
		if _, err := os.Stat(lock); err != nil {
			t.Fatalf("[A:locks-preserved] cleanup removed %s: %v", lock, err)
		}
	}

	// Active content is intact: searching does no indexing work.
	s := e.mcp()
	m, l := e.p.mark(), e.logMark()
	files, _ := s.search("retention survivors", main, main)
	if w := e.work(l, main); w.parsed != 0 || len(e.p.documents(m)) != 0 || len(files) != baseFileCount {
		t.Fatalf("[A:active-protected] parsed=%d embedded=%d results=%v", w.parsed, len(e.p.documents(m)), files)
	}
	s.close()

	// Zero policy retains nothing after the next replacement.
	e.setRetention(db, main, 0, 0)
	edit(main, 100)
	e.index(main)
	if snap = e.inspect(db); snap.retained != 0 || snap.revisions != baseFileCount {
		t.Fatalf("[A:zero-policy] retained=%d revisions=%d", snap.retained, snap.revisions)
	}
	requireNoGarbage(t, "zero-policy", snap)

	// TTL expiry through ordinary replacement maintenance.
	e.setRetention(db, main, 64<<20, time.Hour)
	edit(main, 101)
	e.index(main)
	if snap = e.inspect(db); snap.retained != 1 {
		t.Fatalf("[A:ttl-expiry] retained=%d after one replacement", snap.retained)
	}
	e.fixtureExec(db, `UPDATE revision_cache SET unused_at = unused_at - 7200`)
	edit(main, 102)
	e.index(main)
	if snap = e.inspect(db); snap.retained != 1 || snap.revisions != baseFileCount+1 {
		t.Fatalf("[A:ttl-expiry] expired revision survived: retained=%d revisions=%d", snap.retained, snap.revisions)
	}
	requireNoGarbage(t, "ttl-expiry", snap)
	t.Logf("physical high-water during churn: %d bytes; logical retention budget %d bytes", physicalHigh, budget)
}

// The serialized embedding-result rendezvous cache expires and is bounded by
// its 64 MiB budget independently of revision retention.
func TestRendezvousCacheExpiryAndByteBudget(t *testing.T) {
	e := newEnv(t)
	main := e.repo("rendezvous", baseFiles())
	e.index(main)
	db := e.onlyCollection()
	if snap := e.inspect(db); snap.claimsResults == 0 {
		t.Fatalf("[A:rendezvous-control] cold index published no rendezvous results")
	}
	edit := func(n int) {
		writeFiles(t, main, map[string]string{"beta.go": goFile("fixture", "BetaOne", fmt.Sprintf("Rendezvous%03d", n))})
		e.index(main)
	}

	e.fixtureExec(db, `UPDATE embedding_claims SET expires_at = 1000 WHERE result IS NOT NULL`)
	edit(1)
	if n := e.countRows(db, `SELECT count(*) FROM embedding_claims WHERE expires_at <= ?`, time.Now().UnixMilli()); n != 0 {
		t.Fatalf("[A:rendezvous-expiry] %d expired results survived publication", n)
	}

	planted := sha256.Sum256([]byte("oversized rendezvous fixture"))
	e.fixtureExec(db, `INSERT INTO embedding_claims(input_hash, owner, expires_at, result) VALUES (?, 'fixture', ?, zeroblob(?))`,
		planted[:], time.Now().Add(5*time.Minute).UnixMilli(), 65<<20)
	edit(2)
	if n := e.countRows(db, `SELECT count(*) FROM embedding_claims WHERE owner = 'fixture'`); n != 0 {
		t.Fatalf("[A:rendezvous-byte-budget] oversized result survived the 64 MiB budget")
	}
	if n := e.countRows(db, `SELECT COALESCE(sum(length(result)),0) FROM embedding_claims`); n > 64<<20 {
		t.Fatalf("[A:rendezvous-byte-budget] rendezvous cache holds %d bytes", n)
	}
	if snap := e.inspect(db); snap.claimsResults == 0 {
		t.Fatalf("[A:rendezvous-byte-budget] newest results were pruned too")
	}
}

// Physical storage for a vector-heavy fixture: five identical worktrees stay
// within twice a complete single-worktree collection.
func TestPhysicalStorageForFiveIdenticalWorktrees(t *testing.T) {
	files := map[string]string{}
	for f := range 6 {
		names := make([]string, 25)
		for i := range names {
			names[i] = fmt.Sprintf("Heavy%dFunction%02d", f, i)
		}
		files[fmt.Sprintf("heavy%d.go", f)] = goFile("heavy", names...)
	}

	single := newEnv(t, withDims(768))
	repo := single.repo("physical", files)
	single.index(repo)
	singleDB := single.onlyCollection()
	baseline := physicalBytes(t, singleDB)
	one := single.inspect(singleDB)

	five := newEnv(t, withDims(768))
	repo = five.repo("physical", files)
	roots := []string{repo}
	for i := range 4 {
		roots = append(roots, five.worktree(repo, filepath.Join(filepath.Dir(repo), fmt.Sprintf("physical-%d", i)), "-b", fmt.Sprintf("p%d", i)))
	}
	var high int64
	var fiveDB string
	for _, root := range roots {
		five.index(root)
		fiveDB = five.onlyCollection()
		high = max(high, physicalBytes(t, fiveDB))
	}
	total := physicalBytes(t, fiveDB)
	all := five.inspect(fiveDB)
	if all.vectors != one.vectors || all.revisions != one.revisions || len(all.projects) != 5 {
		t.Fatalf("[A:five-unique-data] vectors %d vs %d, revisions %d vs %d, projects %d", all.vectors, one.vectors, all.revisions, one.revisions, len(all.projects))
	}
	if total > 2*baseline {
		t.Fatalf("[A:five-physical-bound] five worktrees use %d bytes, more than twice the single-worktree %d", total, baseline)
	}
	t.Logf("single worktree %d bytes; five worktrees %d bytes (%.2fx); high-water %d bytes", baseline, total, float64(total)/float64(baseline), high)
}

// Scenario 9: force does full parsing and embedding for the selected worktree
// while sibling memberships, IDs, and results are preserved.
func TestForceReindexPreservesSiblings(t *testing.T) {
	e := newEnv(t)
	main := e.repo("force", baseFiles())
	wt := e.worktree(main, filepath.Join(filepath.Dir(main), "force-wt"), "-b", "force")
	e.index(main)
	e.index(wt)
	db := e.onlyCollection()
	before := e.inspect(db)

	m, l := e.p.mark(), e.logMark()
	e.index(wt, "--force")
	w := e.work(l, wt)
	if w.forcedScans != 1 || w.hashed != baseFileCount {
		t.Fatalf("[A:force-rehash] forced scans=%d hashed=%d", w.forcedScans, w.hashed)
	}
	if w.parsed != baseFileCount {
		t.Fatalf("[A:force-parse] force parsed %d files, want %d", w.parsed, baseFileCount)
	}
	docs := e.p.documents(m)
	requireEachOnce(t, "force-embed-all", docs)
	if len(docs) != before.vectors {
		t.Fatalf("[A:force-embed-all] force embedded %d inputs, want %d", len(docs), before.vectors)
	}
	after := e.inspect(db)
	if !mapsEqual(after.members[main], before.members[main]) || !mapsEqual(after.vectorIDs, before.vectorIDs) {
		t.Fatalf("[A:force-preserves-siblings] sibling memberships or vector IDs changed")
	}
	requireNoGarbage(t, "no-garbage", after)
	s := e.mcp()
	m, l = e.p.mark(), e.logMark()
	files, _ := s.search("force sibling query", main, main)
	if w := e.work(l, main); w.parsed != 0 || len(e.p.documents(m)) != 0 || len(files) != baseFileCount {
		t.Fatalf("[A:force-preserves-siblings] sibling search parsed=%d embedded=%d results=%v", w.parsed, len(e.p.documents(m)), files)
	}
}

// ---------------------------------------------------------------------------

func editFile(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("fixture %s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), old, replacement, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// allFrom reports whether docs is non-empty and every input belongs to file.
func allFrom(docs []string, file string) bool {
	for _, in := range docs {
		if !strings.HasPrefix(in, "// "+file+"\n") {
			return false
		}
	}
	return len(docs) > 0
}

func mapsEqual[K comparable, V comparable](a, b map[K]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

func sameSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(slices.Compact(a), slices.Compact(b))
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// collectionFor finds the collection holding a membership for project.
func (e *env) collectionFor(project string) string {
	e.t.Helper()
	for _, db := range e.collections() {
		if _, ok := e.inspect(db).projects[project]; ok {
			return db
		}
	}
	e.t.Fatalf("no collection has a membership for %s", project)
	return ""
}

func (e *env) countRows(dbPath, query string, args ...any) int64 {
	e.t.Helper()
	db := openReadOnly(e.t, dbPath)
	defer func() { _ = db.Close() }()
	var n int64
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

// trySearch is search without t.Fatal, for use off the test goroutine.
func (s *mcpSession) trySearch(query, path, cwd string) error {
	s.e.p.registerQuery(query)
	ctx, cancel := contextWithTimeout(2 * time.Minute)
	defer cancel()
	res, err := s.session.CallTool(ctx, callParams(query, path, cwd))
	if err != nil {
		return err
	}
	if res.IsError {
		return fmt.Errorf("semantic_search returned an error result")
	}
	return nil
}

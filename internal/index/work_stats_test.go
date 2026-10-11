// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package index

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestIndexerReportsActualScanParseAndAttachWork(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	for _, dir := range []string{first, second} {
		writeGoFile(t, dir, "a.go", "package p\n\nfunc A() {}\n")
		writeGoFile(t, dir, "b.go", "package p\n\nfunc B() {}\n")
	}
	var logs bytes.Buffer
	idx, err := NewIndexerForProject(filepath.Join(t.TempDir(), "index.db"), &mockEmbedder{dims: 4, model: "m"}, 0, "int8", first)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()
	idx.SetLogger(slog.New(slog.NewJSONHandler(&logs, nil)))

	check := func(name string, got Stats, hashed, parsed, attached int) {
		t.Helper()
		if got.FilesHashed != hashed || got.FilesParsed != parsed || got.RevisionsAttached != attached {
			t.Fatalf("%s: hashed=%d parsed=%d attached=%d, want %d/%d/%d", name, got.FilesHashed, got.FilesParsed, got.RevisionsAttached, hashed, parsed, attached)
		}
	}
	stats, err := idx.Index(context.Background(), first, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	check("cold", stats, 2, 2, 0)
	stats, err = idx.Index(context.Background(), second, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	check("identical sibling", stats, 2, 0, 2)
	writeGoFile(t, first, "a.go", "package p\n\nfunc A() { _ = 1 }\n")
	stats, err = idx.Index(context.Background(), first, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.FilesParsed != 1 || stats.RevisionsAttached != 0 || stats.FilesHashed+stats.FilesReused != 2 {
		t.Fatalf("one-file edit: %+v", stats)
	}

	events := map[string]int{}
	for _, line := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
		var entry struct {
			Msg string `json:"msg"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatal(err)
		}
		events[entry.Msg]++
	}
	if events["scan completed"] != 3 || events["index pass completed"] != 3 {
		t.Fatalf("diagnostic events = %v", events)
	}
}

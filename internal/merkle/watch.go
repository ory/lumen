// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package merkle

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watchLocked is best effort. Metadata reconciliation remains active even
// when the OS cannot watch a directory or events overflow.
func (c *Cache) watchLocked(root string) {
	if c.closed {
		return
	}
	if c.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return
		}
		c.watcher = watcher
		c.watched = map[string]bool{}
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					rel, err := filepath.Rel(root, event.Name)
					if err != nil {
						continue
					}
					c.mu.Lock()
					delete(c.Files, rel)
					if c.watched[event.Name] {
						for path := range c.Files {
							if strings.HasPrefix(path, rel+string(os.PathSeparator)) {
								delete(c.Files, path)
							}
						}
					}
					if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
						for path := range c.watched {
							if path == event.Name || strings.HasPrefix(path, event.Name+string(os.PathSeparator)) {
								delete(c.watched, path)
							}
						}
					}
					c.mu.Unlock()
				case _, ok := <-watcher.Errors:
					if !ok {
						return
					}
					c.mu.Lock()
					c.Reconciled = time.Time{}
					c.mu.Unlock()
				}
			}
		}()
	}
}

func (c *Cache) watchDirectoryLocked(path string) {
	if c.watcher != nil && !c.closed && !c.watched[path] {
		if c.watcher.Add(path) == nil {
			c.watched[path] = true
		}
	}
}

// Close releases OS watches. Future builds still use metadata reconciliation.
func (c *Cache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.watcher != nil {
		return c.watcher.Close()
	}
	return nil
}

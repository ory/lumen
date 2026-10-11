//go:build !darwin && !linux

// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package merkle

import "os"

// Unknown stat formats conservatively disable metadata-only reuse.
func fileIdentity(os.FileInfo) (string, int64) { return "", 0 }

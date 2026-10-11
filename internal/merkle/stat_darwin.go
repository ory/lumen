// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package merkle

import (
	"fmt"
	"os"
	"syscall"
)

func fileIdentity(info os.FileInfo) (string, int64) {
	s, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", 0
	}
	return fmt.Sprintf("%d:%d:%d:%d", s.Dev, s.Ino, s.Ctimespec.Sec, s.Ctimespec.Nsec), int64(s.Ctimespec.Sec)*1e9 + int64(s.Ctimespec.Nsec)
}

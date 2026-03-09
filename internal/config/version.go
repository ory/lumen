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

import "runtime/debug"

// BinaryVersion is the git commit hash of this binary. It is injected at build
// time via ldflags:
//
//	-ldflags "-X github.com/ory/lumen/internal/config.BinaryVersion=$(git rev-parse HEAD)"
//
// If not set by ldflags, init() falls back to the VCS revision embedded
// automatically by the Go toolchain (available for both `go build` and
// `go run .` when run inside a git repository). When no VCS information is
// available at all — e.g. building in a CI container without git history or
// outside a VCS directory — the value is set to "dev".
//
// BinaryVersion is included in the DB path hash (see DBPathForProject), so
// each distinct binary version gets its own isolated index. This ensures that
// a binary built with a new embedding format never reads vectors produced by an
// older binary.
var BinaryVersion = ""

func init() {
	if BinaryVersion != "" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				BinaryVersion = s.Value
				return
			}
		}
	}
	BinaryVersion = "dev"
}

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

// IndexVersion is a manually-managed integer that determines the DB path hash.
// Increment this constant whenever a change to the chunker, embedder, or index
// format would make existing indexes incompatible with the current binary.
//
// Using a static version (rather than the git commit hash) means users only
// need to re-index when we deliberately break compatibility, not on every
// release. When you increment this value, document the reason in the commit
// message.
//
// Current version history:
//
//	1 — initial stable version; removed markdown/YAML from chunker pipeline
//	2 — leading comments included in tree-sitter chunk content; Ruby methods
//	    now produce class-qualified symbols (e.g. Animal.speak)
//	3 — Svelte chunker added; .svelte files now indexed via two-phase TS injection
const IndexVersion = "3"

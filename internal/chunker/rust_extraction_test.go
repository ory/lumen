// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package chunker_test

import "testing"

func TestRustDocCommentsAndAttributesAttachToItems(t *testing.T) {
	src := `//! Crate docs are not item docs.
/// Accepts a new incoming connection.
///
/// This function yields once a connection is established.
///
/// # Errors
///
/// Returns an error when the socket fails.
///
/// # Cancel safety
///
/// This method is cancel safe.
pub async fn accept(&self) -> io::Result<()> {
    Ok(())
}

/// A configuration value.
#[derive(Debug, Clone)]
#[serde(rename_all = "snake_case")]
pub struct Config {
    name: String,
}
`
	chunks := chunkSource(t, "listener.rs", src)
	assertDecls(t, chunks, []wantDecl{
		{symbol: "accept", kind: "function", start: 2, end: 15, contains: []string{"/// Accepts a new incoming connection.", "# Cancel safety"}},
		{symbol: "Config", kind: "type", start: 17, end: 22, contains: []string{"/// A configuration value.", "#[derive(Debug, Clone)]", "#[serde("}},
	})
}

func TestRustInnerDocCommentDoesNotAttach(t *testing.T) {
	src := `//! Module docs.
fn first() {}
`
	chunks := chunkSource(t, "lib.rs", src)
	assertDecls(t, chunks, []wantDecl{{symbol: "first", kind: "function", start: 2, end: 2}})
}

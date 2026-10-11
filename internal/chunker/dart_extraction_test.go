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

func TestDartDeclarationsSpanSignatureBodyAndDocs(t *testing.T) {
	src := `/// A handler.
/// Returns a response.
typedef Handler = FutureOr<Response> Function(Request request);

/// Entry point.
void main() {
  print(1);
}

abstract class Repository {
  /// Saves a value.
  Future<void> save(String value);
}

class Message {
  Message(this.body) {
    init();
  }

  /// Reads the body.
  ///
  /// Uses [encoding] when given.
  Future<String> readAsString([Encoding? encoding]) {
    return body;
  }

  /// The content length.
  int get contentLength => body.length;

  set encoding(Encoding value) {
    body = value;
  }

  @override
  void dispose() {
    super.dispose();
  }
}
`
	chunks := chunkSource(t, "message.dart", src)
	assertDecls(t, chunks, []wantDecl{
		{symbol: "Handler", kind: "type", start: 1, end: 3, contains: []string{"/// A handler."}},
		{symbol: "main", kind: "function", start: 5, end: 8, contains: []string{"/// Entry point.", "print(1);"}},
		{symbol: "Repository.save", kind: "method", start: 11, end: 12, contains: []string{"/// Saves a value."}},
		{symbol: "Message.Message", kind: "function", start: 16, end: 18, contains: []string{"init();"}},
		{symbol: "Message.readAsString", kind: "method", start: 20, end: 25, contains: []string{"/// Reads the body.", "return body;"}},
		{symbol: "Message.contentLength", kind: "method", start: 27, end: 28, contains: []string{"/// The content length.", "body.length"}},
		{symbol: "Message.encoding", kind: "method", start: 30, end: 32, contains: []string{"body = value;"}},
		{symbol: "Message.dispose", kind: "method", start: 34, end: 37, contains: []string{"@override", "super.dispose();"}},
	})
}

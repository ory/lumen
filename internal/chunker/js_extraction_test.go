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

const jsAssignedFunctions = `/**
 * Mount middleware.
 */
app.use = function use(fn) {
  return this;
};

Reply.prototype.send = function (payload) {
  return payload;
};

exports.compile = (src) => src;

exports = module.exports = createApplication;

function createApplication() {
  const app = {};
  app.handle = function handle() {};
  return app;
}
`

func TestJSAssignedFunctionsAreDeclarations(t *testing.T) {
	for _, path := range []string{"application.js", "application.ts"} {
		t.Run(path, func(t *testing.T) {
			chunks := chunkSource(t, path, jsAssignedFunctions)
			assertDecls(t, chunks, []wantDecl{
				{symbol: "app.use", kind: "function", start: 1, end: 6, contains: []string{"Mount middleware.", "return this;"}},
				{symbol: "Reply.prototype.send", kind: "function", start: 8, end: 10, contains: []string{"return payload;"}},
				{symbol: "exports.compile", kind: "function", start: 12, end: 12},
				{symbol: "module.exports", kind: "var", start: 14, end: 14},
				{symbol: "createApplication", kind: "function", start: 16, end: 20},
			})
			assertNoSymbol(t, chunks, "app.handle", "createApplication.app.handle")
		})
	}
}

func TestJSModuleExportsStatements(t *testing.T) {
	src := `module.exports = function createServer(opts) {
  return opts;
};
`
	chunks := chunkSource(t, "server.js", src)
	assertDecls(t, chunks, []wantDecl{{symbol: "module.exports", kind: "function", start: 1, end: 3}})

	src = `module.exports = {
  build,
};
`
	chunks = chunkSource(t, "index.cjs", src)
	assertDecls(t, chunks, []wantDecl{{symbol: "module.exports", kind: "var", start: 1, end: 3}})
}

func TestJSAndTSModuleExtensionsAreChunked(t *testing.T) {
	for _, path := range []string{"a.mts", "a.cts", "a.cjs", "a.mjs"} {
		t.Run(path, func(t *testing.T) {
			chunks := chunkSource(t, path, "export function run() {\n  return 1;\n}\n")
			assertDecls(t, chunks, []wantDecl{{symbol: "run", kind: "function", start: 1, end: 3}})
		})
	}
}

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

func TestPythonAssignmentsOnlyAtModuleAndClassLevel(t *testing.T) {
	src := `DEFAULT_TIMEOUT = 30

if TYPE_CHECKING:
    Alias = int


class Flask:
    default_config = {}

    def url_for(self, endpoint):
        _anchor = quote(endpoint)
        rv = build(_anchor)

        def nested():
            inner = 1
            return inner

        return rv
`
	chunks := chunkSource(t, "app.py", src)
	assertDecls(t, chunks, []wantDecl{
		{symbol: "DEFAULT_TIMEOUT", kind: "var", start: 1, end: 1},
		{symbol: "Alias", kind: "var", start: 4, end: 4},
		{symbol: "default_config", kind: "var", start: 8, end: 8},
		{symbol: "url_for", kind: "function", start: 10, end: 18},
	})
	assertNoSymbol(t, chunks, "url_for._anchor", "url_for.rv", "nested.inner", "Flask.url_for._anchor")
}

func TestPythonDecoratedDefinitionEmittedOnce(t *testing.T) {
	src := `class Manager:
    @classmethod
    def _get_queryset_methods(cls, queryset_class):
        return {}


@dataclass
class Point:
    x: int
`
	chunks := chunkSource(t, "manager.py", src)
	assertDecls(t, chunks, []wantDecl{
		{symbol: "_get_queryset_methods", kind: "function", start: 2, end: 4, contains: []string{"@classmethod"}},
		{symbol: "Point", kind: "type", start: 7, end: 9, contains: []string{"@dataclass"}},
	})
}

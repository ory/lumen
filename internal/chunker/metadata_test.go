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

package chunker

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestCodeMetadata(t *testing.T) {
	languages := DefaultLanguages(500)
	tests := []struct{ ext, source, symbol, signature, comment, ref, imp string }{
		{".go", "package p\nimport \"fmt\"\n// Pay prevents duplicate payments.\nfunc Pay(id string) { reserve(id); fmt.Println(id) }", "Pay", "func Pay(id string)", "prevents duplicate", "reserve", "fmt"},
		{".ts", "import { reserve } from './store';\n// Prevent duplicate payments.\nfunction pay(id: string) { reserve(id); store.other(id); }", "pay", "function pay(id: string)", "duplicate payments", "reserve", "./store"},
		{".py", "from store import reserve\ndef pay(id):\n    \"\"\"Prevent duplicate payments.\"\"\"\n    reserve(id)\n    store.other(id)\n", "pay", "def pay(id):", "duplicate payments", "reserve", "from store import reserve"},
		{".rs", "use crate::store::reserve;\n/// Prevent duplicate payments.\nfn pay(id: &str) { reserve(id); store.other(id); }", "pay", "fn pay(id: &str)", "duplicate payments", "reserve", "use crate::store::reserve;"},
	}
	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			chunks, err := languages[tt.ext].Chunk("sample"+tt.ext, []byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			var got *Chunk
			for i := range chunks {
				if chunks[i].Symbol == tt.symbol {
					got = &chunks[i]
					break
				}
			}
			if got == nil {
				t.Fatalf("missing %s: %#v", tt.symbol, chunks)
			}
			if got.Signature != tt.signature {
				t.Errorf("signature = %q", got.Signature)
			}
			if !strings.Contains(got.Comments, tt.comment) {
				t.Errorf("comments = %q", got.Comments)
			}
			expectedRefs := []string{tt.ref}
			if tt.ext == ".go" {
				expectedRefs = []string{"fmt.Println", tt.ref}
			}
			if tt.ext == ".py" {
				// Unsupported Python imports are recorded as hints but not resolved references.
				if len(got.References) != 0 {
					t.Errorf("Python imported call must not resolve by name: %#v", got.References)
				}
			} else if !reflect.DeepEqual(got.References, expectedRefs) {
				t.Errorf("references = %#v", got.References)
			}
			if !strings.Contains(strings.Join(got.Imports, "\n"), tt.imp) {
				t.Errorf("imports = %#v", got.Imports)
			}
		})
	}
}

func TestMetadataSkipsDynamicAndShadowedCalls(t *testing.T) {
	languages := DefaultLanguages(500)
	for ext, source := range map[string]string{
		".go": "package p\nfunc pay(reserve func()) { reserve(); store.reserve(); (get())() }",
		".ts": "function pay(reserve: () => void) { reserve(); store.reserve(); funcs[0](); }",
		".py": "def pay(reserve):\n    reserve()\n    store.reserve()\n    funcs[0]()\n",
	} {
		t.Run(ext, func(t *testing.T) {
			chunks, err := languages[ext].Chunk("sample"+ext, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range chunks {
				for _, ref := range c.References {
					if ref != "get" {
						t.Errorf("unsafe reference %q", ref)
					}
				}
			}
		})
	}
}

func TestTreeMetadataRejectsMalformedDeclaration(t *testing.T) {
	language := DefaultLanguages(500)[".ts"]
	chunks, err := language.Chunk("broken.ts", []byte("function pay() { reserve(; }"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		if len(c.References) != 0 {
			t.Errorf("malformed chunk has references: %#v", c.References)
		}
	}
}

func TestArrowSignatureAndDestructuredBinding(t *testing.T) {
	language := DefaultLanguages(500)[".ts"]
	chunks, err := language.Chunk("arrow.ts", []byte("const pay = (id: string) => reserve(id);\nfunction other() { const {reserve} = store; reserve(); }"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		if c.Symbol == "pay" && strings.Contains(c.Signature, "reserve(id)") {
			t.Errorf("body in signature: %q", c.Signature)
		}
		if c.Symbol == "other" && len(c.References) != 0 {
			t.Errorf("destructured binding resolved: %#v", c.References)
		}
	}
}

func TestImportBindingMetadata(t *testing.T) {
	languages := DefaultLanguages(500)
	for _, tt := range []struct {
		ext, source   string
		imports, refs []string
	}{
		{".ts", "import {reserve as claim} from './store';\nfunction pay() { claim(); }", []string{"./store", "claim=./store#reserve"}, []string{"claim"}},
		{".go", "package p\nimport store \"example.org/project/internal/store\"\nfunc pay() { store.Reserve() }", []string{"example.org/project/internal/store", "store=example.org/project/internal/store"}, []string{"store.Reserve"}},
	} {
		t.Run(tt.ext, func(t *testing.T) {
			chunks, err := languages[tt.ext].Chunk("sample"+tt.ext, []byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(chunks) != 1 {
				t.Fatalf("chunks = %#v", chunks)
			}
			if !reflect.DeepEqual(chunks[0].Imports, tt.imports) {
				t.Errorf("imports = %#v", chunks[0].Imports)
			}
			if !reflect.DeepEqual(chunks[0].References, tt.refs) {
				t.Errorf("references = %#v", chunks[0].References)
			}
		})
	}
}

func TestGoShadowedImportIsNotReference(t *testing.T) {
	chunks, err := NewGoAST().Chunk("sample.go", []byte("package p\nimport \"store\"\nfunc pay(store Store) { store.Reserve() }"))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks[0].References) != 0 {
		t.Errorf("references = %#v", chunks[0].References)
	}
}

func TestGoMethodExportedUsesMethodName(t *testing.T) {
	chunks, err := NewGoAST().Chunk("sample.go", []byte("package p\ntype Server struct{}\ntype server struct{}\nfunc (Server) handle() {}\nfunc (server) Run() {}\nfunc (*Server) Serve() {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Server": true, "server": false, "Server.handle": false, "server.Run": true, "Server.Serve": true}
	for _, c := range chunks {
		if exported, ok := want[c.Symbol]; ok && c.Exported != exported {
			t.Errorf("%s: Exported = %v, want %v", c.Symbol, c.Exported, exported)
		}
	}
}

func TestGoImportAssumedPackageNames(t *testing.T) {
	source := "package p\nimport (\n\t\"github.com/foo/bar/v2\"\n\t\"gopkg.in/yaml.v3\"\n\t\"github.com/mattn/go-sqlite3\"\n\t\"example.com/v3\"\n)\nfunc run() { bar.New(); yaml.Unmarshal(); sqlite3.Open(); example.Do() }\n"
	chunks, err := NewGoAST().Chunk("sample.go", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var got *Chunk
	for i := range chunks {
		if chunks[i].Symbol == "run" {
			got = &chunks[i]
		}
	}
	if got == nil {
		t.Fatalf("missing run: %#v", chunks)
	}
	wantRefs := []string{"bar.New", "example.Do", "sqlite3.Open", "yaml.Unmarshal"}
	if !reflect.DeepEqual(got.References, wantRefs) {
		t.Errorf("references = %#v, want %#v", got.References, wantRefs)
	}
	for _, imp := range []string{"bar=github.com/foo/bar/v2", "yaml=gopkg.in/yaml.v3", "sqlite3=github.com/mattn/go-sqlite3", "example=example.com/v3"} {
		if !slices.Contains(got.Imports, imp) {
			t.Errorf("imports %#v missing %q", got.Imports, imp)
		}
	}
	for _, bogus := range []string{"v2=github.com/foo/bar/v2", "yaml.v3=gopkg.in/yaml.v3", "go-sqlite3=github.com/mattn/go-sqlite3"} {
		if slices.Contains(got.Imports, bogus) {
			t.Errorf("imports %#v contain bogus alias %q", got.Imports, bogus)
		}
	}
}

func TestExportedDeclarationLeadingComment(t *testing.T) {
	languages := DefaultLanguages(500)
	for _, ext := range []string{".ts", ".js"} {
		chunks, err := languages[ext].Chunk("sample"+ext, []byte("const x = 1;\n/** Prevent duplicate payments. */\nexport function pay() {}\n"))
		if err != nil {
			t.Fatal(err)
		}
		var got *Chunk
		for i := range chunks {
			if chunks[i].Symbol == "pay" {
				got = &chunks[i]
			}
		}
		if got == nil {
			t.Fatalf("%s: missing pay: %#v", ext, chunks)
		}
		// The leading doc comment belongs to the exported declaration's chunk.
		if got.StartLine != 2 || !strings.HasPrefix(got.Content, "/** Prevent") {
			t.Errorf("%s: start=%d content=%q", ext, got.StartLine, got.Content)
		}
		if got.Comments != "/** Prevent duplicate payments. */" {
			t.Errorf("%s: comments = %q", ext, got.Comments)
		}
		if !got.Exported {
			t.Errorf("%s: pay not exported", ext)
		}
	}
}

// A call in a declaration's initializer is a call, not a binding: only the
// declared name is bound.
func TestInitializerCallsAreReferences(t *testing.T) {
	languages := DefaultLanguages(500)
	for _, tt := range []struct{ ext, source, symbol, ref, notRef string }{
		{".ts", "import { chargeDeposit } from './payments';\nexport async function confirmBooking(id: string) {\n  const depositId = await chargeDeposit(id);\n  return depositId;\n}\n", "confirmBooking", "chargeDeposit", "depositId"},
		{".js", "import { convertCurrency } from './fx.js';\nexport function quote(n) { const converted = convertCurrency(n); return converted; }\n", "quote", "convertCurrency", "converted"},
		{".rs", "fn pay(id: &str) { let receipt = reserve(id); receipt.done(); }\n", "pay", "reserve", "receipt"},
	} {
		t.Run(tt.ext, func(t *testing.T) {
			chunks, err := languages[tt.ext].Chunk("sample"+tt.ext, []byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range chunks {
				if c.Symbol != tt.symbol {
					continue
				}
				if !slices.Contains(c.References, tt.ref) || slices.Contains(c.References, tt.notRef) {
					t.Fatalf("references = %v, want %s and not %s", c.References, tt.ref, tt.notRef)
				}
				return
			}
			t.Fatalf("missing %s: %#v", tt.symbol, chunks)
		})
	}
}

// Direct top-level export forms are statically provable; aliases and
// re-exports are not attributed to a local declaration.
func TestExportClauseAndDefaultExports(t *testing.T) {
	source := `function releaseHold(id: string) { return id; }
function holdCount() { return 1; }
function renamed() { return 2; }
function internalOnly() { return 3; }
function plainDefault() { return 4; }
export { releaseHold, holdCount as holdCount, renamed as other };
export { internalOnly } from './elsewhere';
export default plainDefault;
`
	for _, tt := range []struct {
		ext, source string
		want        map[string][2]bool // symbol -> {Exported, DefaultExport}
	}{
		{".ts", source, map[string][2]bool{"releaseHold": {true, false}, "holdCount": {true, false}, "renamed": {false, false}, "internalOnly": {false, false}, "plainDefault": {false, true}}},
		{".js", "export default function issueInvoice(id) { return id; }\nexport function sendInvoice() {}\n", map[string][2]bool{"issueInvoice": {false, true}, "sendInvoice": {true, false}}},
	} {
		t.Run(tt.ext, func(t *testing.T) {
			chunks, err := DefaultLanguages(500)[tt.ext].Chunk("m"+tt.ext, []byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			got := map[string][2]bool{}
			for _, c := range chunks {
				got[c.Symbol] = [2]bool{c.Exported, c.DefaultExport}
			}
			for symbol, want := range tt.want {
				if g, ok := got[symbol]; !ok || g != want {
					t.Errorf("%s {Exported, DefaultExport} = %v (present=%v), want %v", symbol, g, ok, want)
				}
			}
		})
	}
}

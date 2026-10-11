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
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func uniqueNames(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func walkSyntax(node *sitter.Node, visit func(*sitter.Node) bool) {
	if node == nil || !visit(node) {
		return
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkSyntax(node.NamedChild(i), visit)
	}
}

// Metadata is deliberately syntax-only. Direct calls are candidates, not proof of
// a dependency; member access, dynamic calls and malformed trees are excluded.
func populateTreeMetadata(chunk *Chunk, decl *sitter.Node, content []byte, startByte uint32) {
	if parent := decl.Parent(); parent != nil && parent.Type() == "export_statement" {
		direct := parent.ChildByFieldName("declaration")
		if direct != nil && direct.StartByte() == decl.StartByte() {
			chunk.Exported = true
			for i := 0; i < int(parent.ChildCount()); i++ {
				if parent.Child(i).Type() == "default" {
					chunk.Exported = false
				}
			}
		}
	}
	signatureEnd := decl.EndByte()
	var comments, refs []string
	// Leading comments end where the declaration, or its export statement, begins;
	// the export keyword and decorators are not comment text.
	commentEnd := decl.StartByte()
	if parent := decl.Parent(); parent != nil && parent.Type() == "export_statement" && startByte <= parent.StartByte() {
		commentEnd = parent.StartByte()
	}
	if startByte < commentEnd {
		comments = append(comments, string(content[startByte:commentEnd]))
	}
	bound := map[string]bool{}
	walkSyntax(decl, func(n *sitter.Node) bool {
		if body := n.ChildByFieldName("body"); body != nil && body.StartByte() < signatureEnd {
			signatureEnd = body.StartByte()
		}
		if isCommentNode(n.Type()) {
			comments = append(comments, n.Content(content))
			return false
		}
		if n.StartByte() != decl.StartByte() {
			switch n.Type() {
			case "function_declaration", "function_definition", "class_declaration", "class_definition":
				if name := n.ChildByFieldName("name"); name != nil {
					bound[name.Content(content)] = true
				}
			}
		}
		switch n.Type() {
		case "block", "statement_block", "compound_statement", "class_body", "declaration_list", "interface_body", "enum_body":
			if n.StartByte() < signatureEnd {
				signatureEnd = n.StartByte()
			}
		case "parameters", "formal_parameters", "lambda_parameters", "parameter_list", "catch_clause", "except_clause", "import_statement", "import_from_statement":
			// Over-approximate bindings: excluding a type name is safer than following
			// a parameter whose name happens to match a repository function.
			walkSyntax(n, func(param *sitter.Node) bool {
				if param.Type() == "identifier" {
					bound[param.Content(content)] = true
				}
				return true
			})
		case "variable_declarator", "let_declaration", "assignment", "for_in_statement", "for_statement", "for_each_statement", "for_in_clause", "for_clause":
			for _, field := range []string{"name", "pattern", "left", "value", "key"} {
				if name := n.ChildByFieldName(field); name != nil {
					walkSyntax(name, func(binding *sitter.Node) bool {
						if binding.Type() == "identifier" || binding.Type() == "shorthand_property_identifier_pattern" {
							bound[binding.Content(content)] = true
						}
						return true
					})
				}
			}
		}
		// Python docstrings are expression statements containing a string as their
		// first statement in a body. Ordinary string literals are not comments.
		if n.Type() == "block" && n.NamedChildCount() > 0 {
			first := n.NamedChild(0)
			if first.Type() == "expression_statement" && first.NamedChildCount() == 1 && first.NamedChild(0).Type() == "string" {
				comments = append(comments, first.Content(content))
			}
		}
		if !n.HasError() {
			var callee *sitter.Node
			switch n.Type() {
			case "call_expression", "call", "invocation_expression", "function_call_expression":
				callee = n.ChildByFieldName("function")
				if n.Type() == "call" && callee == nil && n.ChildByFieldName("receiver") == nil {
					callee = n.ChildByFieldName("method")
				}
			case "method_invocation":
				if n.ChildByFieldName("object") == nil {
					callee = n.ChildByFieldName("name")
				}
			}
			if callee != nil && (callee.Type() == "identifier" || callee.Type() == "name") {
				refs = append(refs, callee.Content(content))
			}
		}
		return true
	})
	chunk.Signature = strings.TrimSpace(string(content[decl.StartByte():signatureEnd]))
	chunk.Comments = strings.TrimSpace(strings.Join(comments, "\n"))
	if !decl.HasError() {
		for _, ref := range uniqueNames(refs) {
			if !bound[ref] {
				chunk.References = append(chunk.References, ref)
			}
		}
	}
}

// Keep import declarations as hints when a grammar does not expose a source
// field. Consumers must resolve these explicitly, never by basename guessing.
func treeImports(root *sitter.Node, content []byte) []string {
	var imports []string
	walkSyntax(root, func(n *sitter.Node) bool {
		switch n.Type() {
		case "import_statement", "import_from_statement", "import_declaration", "use_declaration", "using_directive", "preproc_include":
			if n.HasError() {
				return false
			}
			if source := n.ChildByFieldName("source"); source != nil {
				sourcePath := strings.Trim(source.Content(content), "\"'")
				imports = append(imports, sourcePath)
				if n.Type() == "import_statement" {
					walkSyntax(n, func(binding *sitter.Node) bool {
						if binding.Type() == "import_specifier" {
							name := binding.ChildByFieldName("name")
							if name != nil && name.Type() == "identifier" {
								local := name.Content(content)
								if alias := binding.ChildByFieldName("alias"); alias != nil {
									local = alias.Content(content)
								}
								imports = append(imports, local+"="+sourcePath+"#"+name.Content(content))
							}
							return false
						}
						// Default imports are direct children of import_clause.
						if binding.Type() == "import_clause" {
							for i := 0; i < int(binding.NamedChildCount()); i++ {
								child := binding.NamedChild(i)
								if child.Type() == "identifier" {
									imports = append(imports, child.Content(content)+"="+sourcePath+"#default")
								}
							}
						}
						return true
					})
				}
			} else {
				imports = append(imports, n.Content(content))
			}
			return false
		}
		return true
	})
	return uniqueNames(imports)
}

// Unsupported rebinding anywhere in a file is conservatively excluded from
// lexical expansion. This intentionally favors missing an edge over reporting
// a same-named function which an import or assignment replaced.
func treeShadowedNames(root *sitter.Node, content []byte) map[string]bool {
	names := map[string]bool{}
	bind := func(node *sitter.Node) {
		walkSyntax(node, func(n *sitter.Node) bool {
			if n.Type() == "identifier" || n.Type() == "shorthand_property_identifier_pattern" {
				names[n.Content(content)] = true
			}
			return true
		})
	}
	walkSyntax(root, func(n *sitter.Node) bool {
		switch n.Type() {
		case "assignment", "assignment_expression", "augmented_assignment", "variable_declarator", "let_declaration":
			for _, field := range []string{"name", "pattern", "left"} {
				if node := n.ChildByFieldName(field); node != nil {
					bind(node)
				}
			}
		case "import_statement", "import_from_statement":
			// JS/TS source-bearing named imports have explicit binding metadata.
			if n.ChildByFieldName("source") == nil {
				bind(n)
			}
		}
		return true
	})
	return names
}

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
	sitter_dart "github.com/alexaandru/go-sitter-forest/dart"
	sitter "github.com/smacker/go-tree-sitter"
	sitter_c "github.com/smacker/go-tree-sitter/c"
	sitter_cpp "github.com/smacker/go-tree-sitter/cpp"
	sitter_cs "github.com/smacker/go-tree-sitter/csharp"
	sitter_java "github.com/smacker/go-tree-sitter/java"
	sitter_js "github.com/smacker/go-tree-sitter/javascript"
	sitter_php "github.com/smacker/go-tree-sitter/php"
	sitter_py "github.com/smacker/go-tree-sitter/python"
	sitter_rb "github.com/smacker/go-tree-sitter/ruby"
	sitter_rs "github.com/smacker/go-tree-sitter/rust"
	sitter_tsx "github.com/smacker/go-tree-sitter/typescript/tsx"
	sitter_ts "github.com/smacker/go-tree-sitter/typescript/typescript"
)

// supportedExtensions is the canonical list of file extensions handled by DefaultLanguages.
var supportedExtensions = []string{
	".go",
	".ts", ".tsx",
	".js", ".jsx", ".mjs",
	".py",
	".rs",
	".rb",
	".java",
	".c", ".h",
	".cpp", ".cc", ".cxx", ".hpp",
	".php",
	".cs",
	".dart",
	".svelte",
	".md", ".mdx",
	".yaml", ".yml", ".json",
}

// SupportedExtensions returns the file extensions indexed by DefaultLanguages.
func SupportedExtensions() []string { return supportedExtensions }

// DefaultLanguages returns a map of file extension → Chunker for all supported languages.
// maxChunkTokens is the token budget per chunk used by the StructuredChunker.
// Panics if any hardcoded query pattern is invalid (programming error).
func DefaultLanguages(maxChunkTokens int) map[string]Chunker {
	py := mustTreeSitterChunker(LanguageDef{
		Language: sitter_py.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(function_definition name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(class_definition name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(decorated_definition definition: (function_definition name: (identifier) @name)) @decl`, Kind: "function"},
			{Pattern: `(decorated_definition definition: (class_definition name: (identifier) @name)) @decl`, Kind: "type"},
			{Pattern: `(assignment left: (identifier) @name) @decl`, Kind: "var"},
		},
	})

	ts := mustTreeSitterChunker(LanguageDef{
		Language: sitter_ts.GetLanguage(),
		Queries: []QueryDef{
			// Exported const/let with non-function values (regex, string, object, etc.).
			// Must come before the lexical_declaration function patterns so that exported
			// arrow functions are overwritten by the more specific "function" kind below.
			{Pattern: `(export_statement (lexical_declaration (variable_declarator name: (identifier) @name))) @decl`, Kind: "const"},
			{Pattern: `(function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(generator_function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(lexical_declaration (variable_declarator name: (identifier) @name value: [(arrow_function) (function_expression) (generator_function)])) @decl`, Kind: "function"},
			{Pattern: `(variable_declaration (variable_declarator name: (identifier) @name value: [(arrow_function) (function_expression) (generator_function)])) @decl`, Kind: "function"},
			{Pattern: `(class_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(abstract_class_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(interface_declaration name: (type_identifier) @name) @decl`, Kind: "interface"},
			{Pattern: `(type_alias_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(method_definition name: (property_identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(method_signature name: (property_identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(internal_module name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(ambient_declaration (function_signature name: (identifier) @name)) @decl`, Kind: "function"},
		},
	})

	tsx := mustTreeSitterChunker(LanguageDef{
		Language: sitter_tsx.GetLanguage(),
		Queries: []QueryDef{
			// Exported const/let with non-function values (regex, string, object, etc.).
			// Must come before the lexical_declaration function patterns so that exported
			// arrow functions are overwritten by the more specific "function" kind below.
			{Pattern: `(export_statement (lexical_declaration (variable_declarator name: (identifier) @name))) @decl`, Kind: "const"},
			{Pattern: `(function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(generator_function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(lexical_declaration (variable_declarator name: (identifier) @name value: [(arrow_function) (function_expression) (generator_function)])) @decl`, Kind: "function"},
			{Pattern: `(variable_declaration (variable_declarator name: (identifier) @name value: [(arrow_function) (function_expression) (generator_function)])) @decl`, Kind: "function"},
			{Pattern: `(class_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(abstract_class_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(interface_declaration name: (type_identifier) @name) @decl`, Kind: "interface"},
			{Pattern: `(type_alias_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(method_definition name: (property_identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(method_signature name: (property_identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(internal_module name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(ambient_declaration (function_signature name: (identifier) @name)) @decl`, Kind: "function"},
		},
	})

	js := mustTreeSitterChunker(LanguageDef{
		Language: sitter_js.GetLanguage(),
		Queries: []QueryDef{
			// Exported const/let with non-function values (regex, string, object, etc.).
			// Must come before the lexical_declaration function patterns so that exported
			// arrow functions are overwritten by the more specific "function" kind below.
			{Pattern: `(export_statement (lexical_declaration (variable_declarator name: (identifier) @name))) @decl`, Kind: "const"},
			{Pattern: `(function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(generator_function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(lexical_declaration (variable_declarator name: (identifier) @name value: [(arrow_function) (function_expression) (generator_function)])) @decl`, Kind: "function"},
			{Pattern: `(variable_declaration (variable_declarator name: (identifier) @name value: [(arrow_function) (function_expression) (generator_function)])) @decl`, Kind: "function"},
			{Pattern: `(class_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(method_definition name: (property_identifier) @name) @decl`, Kind: "method"},
		},
	})

	rs := mustTreeSitterChunker(LanguageDef{
		Language: sitter_rs.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(function_item name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(struct_item name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_item name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(trait_item name: (type_identifier) @name) @decl`, Kind: "interface"},
			{Pattern: `(const_item name: (identifier) @name) @decl`, Kind: "const"},
			{Pattern: `(impl_item type: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(impl_item type: (generic_type type: (type_identifier) @name)) @decl`, Kind: "type"},
			{Pattern: `(impl_item type: (scoped_type_identifier name: (type_identifier) @name)) @decl`, Kind: "type"},
			{Pattern: `(type_item name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(static_item name: (identifier) @name) @decl`, Kind: "var"},
			{Pattern: `(mod_item name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(macro_definition name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(union_item name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(function_signature_item name: (identifier) @name) @decl`, Kind: "function"},
		},
	})

	rb := mustTreeSitterChunker(LanguageDef{
		Language: sitter_rb.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(method name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(singleton_method name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(class name: (constant) @name) @decl`, Kind: "type"},
			{Pattern: `(module name: (constant) @name) @decl`, Kind: "type"},
			{Pattern: `(alias name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(assignment left: (constant) @name) @decl`, Kind: "var"},
		},
	})

	java := mustTreeSitterChunker(LanguageDef{
		Language: sitter_java.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(method_declaration name: (identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(class_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(interface_declaration name: (identifier) @name) @decl`, Kind: "interface"},
			{Pattern: `(constructor_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(enum_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(annotation_type_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(record_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(field_declaration declarator: (variable_declarator name: (identifier) @name)) @decl`, Kind: "var"},
		},
	})

	c := mustTreeSitterChunker(LanguageDef{
		Language: sitter_c.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(function_definition declarator: (function_declarator declarator: (identifier) @name)) @decl`, Kind: "function"},
			{Pattern: `(function_definition declarator: (pointer_declarator declarator: (function_declarator declarator: (identifier) @name))) @decl`, Kind: "function"},
			{Pattern: `(struct_specifier name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_specifier name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(union_specifier name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(type_definition declarator: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(preproc_function_def name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(preproc_def name: (identifier) @name) @decl`, Kind: "const"},
		},
	})

	cpp := mustTreeSitterChunker(LanguageDef{
		Language: sitter_cpp.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(function_definition declarator: (function_declarator declarator: (identifier) @name)) @decl`, Kind: "function"},
			{Pattern: `(function_definition declarator: (pointer_declarator declarator: (function_declarator declarator: (identifier) @name))) @decl`, Kind: "function"},
			{Pattern: `(class_specifier name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(struct_specifier name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_specifier name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(function_definition declarator: (function_declarator declarator: (qualified_identifier name: (identifier) @name))) @decl`, Kind: "method"},
			{Pattern: `(namespace_definition name: (namespace_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(alias_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(concept_definition name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(union_specifier name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(type_definition declarator: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(preproc_function_def name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(preproc_def name: (identifier) @name) @decl`, Kind: "const"},
			// template_declaration wrappers
			{Pattern: `(template_declaration (function_definition declarator: (function_declarator declarator: (identifier) @name))) @decl`, Kind: "function"},
			{Pattern: `(template_declaration (function_definition declarator: (function_declarator declarator: (qualified_identifier name: (identifier) @name)))) @decl`, Kind: "method"},
			{Pattern: `(template_declaration (function_definition declarator: (pointer_declarator declarator: (function_declarator declarator: (identifier) @name)))) @decl`, Kind: "function"},
			{Pattern: `(template_declaration (class_specifier name: (type_identifier) @name)) @decl`, Kind: "type"},
			{Pattern: `(template_declaration (struct_specifier name: (type_identifier) @name)) @decl`, Kind: "type"},
			{Pattern: `(template_declaration (alias_declaration name: (type_identifier) @name)) @decl`, Kind: "type"},
		},
	})

	php := mustTreeSitterChunker(LanguageDef{
		Language: sitter_php.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(function_definition name: (name) @name) @decl`, Kind: "function"},
			{Pattern: `(interface_declaration name: (name) @name) @decl`, Kind: "interface"},
			{Pattern: `(trait_declaration name: (name) @name) @decl`, Kind: "type"},
			{Pattern: `(method_declaration name: (name) @name) @decl`, Kind: "method"},
			{Pattern: `(class_declaration name: (name) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_declaration name: (name) @name) @decl`, Kind: "type"},
			{Pattern: `(namespace_definition name: (namespace_name) @name) @decl`, Kind: "type"},
			{Pattern: `(const_declaration (const_element (name) @name)) @decl`, Kind: "const"},
			{Pattern: `(property_declaration (property_element (variable_name (name) @name))) @decl`, Kind: "var"},
		},
	})

	cs := mustTreeSitterChunker(LanguageDef{
		Language: sitter_cs.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(method_declaration name: (identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(class_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(interface_declaration name: (identifier) @name) @decl`, Kind: "interface"},
			{Pattern: `(struct_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(property_declaration name: (identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(constructor_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(destructor_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(delegate_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(record_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(event_field_declaration (variable_declaration (variable_declarator (identifier) @name))) @decl`, Kind: "var"},
			{Pattern: `(namespace_declaration name: (_) @name) @decl`, Kind: "type"},
			{Pattern: `(field_declaration (variable_declaration (variable_declarator (identifier) @name))) @decl`, Kind: "var"},
			{Pattern: `(event_declaration name: (identifier) @name) @decl`, Kind: "var"},
		},
	})

	// svelteTS reuses the TypeScript grammar for script block injection.
	// It is intentionally a separate instance so its Language pointer is
	// the same *sitter.Language used elsewhere for .ts files.
	svelteTS := mustTreeSitterChunker(LanguageDef{
		Language: sitter_ts.GetLanguage(),
		Queries: []QueryDef{
			{Pattern: `(export_statement (lexical_declaration (variable_declarator name: (identifier) @name))) @decl`, Kind: "const"},
			{Pattern: `(function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(generator_function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(lexical_declaration (variable_declarator name: (identifier) @name value: [(arrow_function) (function_expression) (generator_function)])) @decl`, Kind: "function"},
			{Pattern: `(variable_declaration (variable_declarator name: (identifier) @name value: [(arrow_function) (function_expression) (generator_function)])) @decl`, Kind: "function"},
			{Pattern: `(class_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(abstract_class_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(interface_declaration name: (type_identifier) @name) @decl`, Kind: "interface"},
			{Pattern: `(type_alias_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(method_definition name: (property_identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(method_signature name: (property_identifier) @name) @decl`, Kind: "method"},
			{Pattern: `(internal_module name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(ambient_declaration (function_signature name: (identifier) @name)) @decl`, Kind: "function"},
		},
	})

	svelte := NewSvelteChunker(svelteTS)

	dart := mustTreeSitterChunker(LanguageDef{
		Language: sitter.NewLanguage(sitter_dart.GetLanguage()),
		Queries: []QueryDef{
			{Pattern: `(class_definition name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(mixin_declaration (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_declaration name: (identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(extension_declaration name: (identifier) @name) @decl`, Kind: "type"},
			// Top-level and abstract method function signatures.
			{Pattern: `(function_signature name: (identifier) @name) @decl`, Kind: "function"},
			// Class/mixin body declarations override function_signature kind to "method".
			{Pattern: `(declaration (function_signature name: (identifier) @name)) @decl`, Kind: "method"},
			{Pattern: `(method_signature (function_signature name: (identifier) @name)) @decl`, Kind: "method"},
			{Pattern: `(method_signature (getter_signature (identifier) @name)) @decl`, Kind: "method"},
			{Pattern: `(method_signature (setter_signature (identifier) @name)) @decl`, Kind: "method"},
			{Pattern: `(type_alias . (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(constructor_signature (identifier) @name) @decl`, Kind: "function"},
		},
	})

	goChunker := NewGoAST()

	md := NewMarkdownChunker()

	structured := NewStructuredChunker(maxChunkTokens)

	return map[string]Chunker{
		".go":   goChunker,
		".ts":   ts,
		".tsx":  tsx,
		".js":   js,
		".jsx":  js,
		".mjs":  js,
		".py":   py,
		".rs":   rs,
		".rb":   rb,
		".java": java,
		".c":    c,
		".h":    cpp,
		".cpp":  cpp,
		".cc":   cpp,
		".cxx":  cpp,
		".hpp":  cpp,
		".php":  php,
		".cs":   cs,
		".dart":   dart,
		".svelte": svelte,
		".md":     md,
		".mdx":  md,
		".yaml": structured,
		".yml":  structured,
		".json": structured,
	}
}

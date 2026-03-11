# SWE-Bench Detail Report

Generated: 2026-03-11 11:37 UTC

---

## rust-hard [rust]

**Issue:** False duplicate key error for dotted keys when parent table is implicitly created by a nested header

> The following valid TOML document is incorrectly rejected by the parser with a false 'duplicate key' error:
> 
> ```toml
> [fruit.apple.texture]
> smooth = true
> 
> [fruit]
> apple.color = "red"
> apple.taste.sweet = true
> ```
> 
> Parsing this document produces the error:
> ```
> TOML parse error at line 6, column 7
>   |
> 6 | apple.color = "red"
>   |       ^^^^^
> duplicate key
> ```
> 
> The document is valid according to the TOML specification. When `[fruit.apple.texture]` is declared first, it implicitly creates the `fruit` and `fruit.apple` tables. When `[fruit]` is later explicitly declared, dotted keys like `apple.color` should be allowed to extend the implicitly-created `fruit.apple` table, since it was never explicitly defined -- only implicitly created as a parent of `fruit.apple.texture`.
> 
> Both the Python standard library `tomllib` and the BurntSushi TOML validator accept this document as valid.
> 
> The TOML specification discussion at toml-lang/toml#769 clarifies that order should not matter in TOML (except for arrays of tables), and that implicitly created tables can be extended with dotted keys.
> 
> Note that the parser currently accepts these related cases correctly:
> 
> 1. When the nested header creates an unrelated key under the same parent:
> ```toml
> [fruit.apple.texture]
> smooth = true
> 
> [fruit]
> banana = "yellow"
> ```
> 
> 2. When the dotted keys appear before the nested header:
> ```toml
> [fruit]
> apple.color = "red"
> apple.taste.sweet = true
> 
> [fruit.apple.texture]
> smooth = true
> ```
> 
> The bug only manifests when dotted keys in a later `[fruit]` section attempt to extend a table that was implicitly created by an earlier nested header like `[fruit.apple.texture]`.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 309.7s | 25 | 718705 | 17717 | $0.6113 |
| **with-lumen** | 204.0s | 17 | 240658 | 12291 | $0.3748 |

### baseline

**Rating: Poor**

```diff
diff --git a/crates/toml_edit/src/parser/document.rs b/crates/toml_edit/src/parser/document.rs
index 14f17fb..3439fcb 100644
--- a/crates/toml_edit/src/parser/document.rs
+++ b/crates/toml_edit/src/parser/document.rs
@@ -307,7 +307,9 @@ impl State {
             return;
         };
         // "Likewise, using dotted keys to redefine tables already defined in [table] form is not allowed"
-        let mixed_table_types = parent_table.is_dotted() == path.is_empty();
+        // Implicit tables (created as side effects of nested headers like [a.b.c]) can be extended
+        let mixed_table_types =
+            !parent_table.is_implicit() && (parent_table.is_dotted() == path.is_empty());
         if mixed_table_types {
             let key_span = get_key_span(&key).expect("all keys have spans");
             errors.report_error(ParseError::new("duplicate key").with_unexpected(key_span));
diff --git a/crates/toml_edit/tests/compliance/parse.rs b/crates/toml_edit/tests/compliance/parse.rs
index 6ab6cae..9f3acdb 100644
--- a/crates/toml_edit/tests/compliance/parse.rs
+++ b/crates/toml_edit/tests/compliance/parse.rs
@@ -257,6 +257,31 @@ name = "plantain"
     );
 }
 
+#[test]
+fn dotted_key_extends_implicit_table_from_nested_header() {
+    // Regression test: dotted keys should be allowed to extend a table that was
+    // implicitly created by a nested header like [fruit.apple.texture].
+    // See: https://github.com/toml-lang/toml/issues/769
+    let table = r#"[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#
+    .parse::<crate::RustDocument>()
+    .unwrap();
+    assert_eq!(
+        table["fruit"]["apple"]["texture"]["smooth"].as_bool(),
+        Some(true)
+    );
+    assert_eq!(table["fruit"]["apple"]["color"].as_str(), Some("red"));
+    assert_eq!(
+        table["fruit"]["apple"]["taste"]["sweet"].as_bool(),
+        Some(true)
+    );
+}
+
 #[test]
 fn blank_literal_string() {
     let table = "foo = ''".parse::<crate::RustDocument>().unwrap();
```

### with-lumen

**Rating: Poor**

```diff
diff --git a/crates/toml_edit/src/parser/document.rs b/crates/toml_edit/src/parser/document.rs
index 14f17fb..bd6956a 100644
--- a/crates/toml_edit/src/parser/document.rs
+++ b/crates/toml_edit/src/parser/document.rs
@@ -307,7 +307,9 @@ impl State {
             return;
         };
         // "Likewise, using dotted keys to redefine tables already defined in [table] form is not allowed"
-        let mixed_table_types = parent_table.is_dotted() == path.is_empty();
+        // Implicit tables (created as parents by nested headers like [a.b.c]) can be extended with dotted keys
+        let mixed_table_types =
+            parent_table.is_dotted() == path.is_empty() && !parent_table.is_implicit();
         if mixed_table_types {
             let key_span = get_key_span(&key).expect("all keys have spans");
             errors.report_error(ParseError::new("duplicate key").with_unexpected(key_span));
```



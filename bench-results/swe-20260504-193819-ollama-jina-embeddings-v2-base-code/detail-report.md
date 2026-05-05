# SWE-Bench Detail Report

Generated: 2026-05-04 23:58 UTC

---

## java-hard [java]

**Issue:** `*=` trims its argument

> The CSS attribute selector `*=` (contains) incorrectly trims whitespace from its argument value, causing it to match elements that it should not match according to the CSS specification.
> 
> For example, given the HTML:
> ```html
> <body>
>   <article xxx="xpaywalled"></article>
> </body>
> ```
> 
> The selector `article[xxx*="paywalled"]` correctly matches the element (the attribute value contains "paywalled").
> 
> However, the selector `article[xxx*=" paywalled "]` (with leading and trailing spaces in the value) also matches the element, which is incorrect. The browser's native `document.querySelectorAll` respects the spaces and does not match, because the attribute value "xpaywalled" does not contain the substring " paywalled " (with spaces).
> 
> This trimming behavior affects multiple attribute selectors including exact match (`=`), starts-with (`^=`), ends-with (`$=`), and contains (`*=`). Quoted attribute values should preserve their whitespace exactly as specified, consistent with browser CSS selector behavior.
> 
> Additionally, selectors with empty attribute values like `div[data='']` or `div[data^='']` should be valid and match elements with empty or present attributes, but currently fail with a validation error because the value is rejected as empty after trimming.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 171.2s | 226 | 1347600 | 13876 | $0.4108 |
| **with-lumen** | 165.8s | 272 | 1810826 | 16192 | $0.3617 |

### baseline

**Rating: Poor**

```diff
diff --git a/src/main/java/org/jsoup/select/Evaluator.java b/src/main/java/org/jsoup/select/Evaluator.java
index 8c93fba..64d0bca 100644
--- a/src/main/java/org/jsoup/select/Evaluator.java
+++ b/src/main/java/org/jsoup/select/Evaluator.java
@@ -267,12 +267,12 @@ public abstract class Evaluator {
      */
     public static final class AttributeWithValue extends AttributeKeyPair {
         public AttributeWithValue(String key, String value) {
-            super(key, value);
+            super(key, value, false);
         }
 
         @Override
         public boolean matches(Element root, Element element) {
-            return element.hasAttr(key) && value.equalsIgnoreCase(element.attr(key).trim());
+            return element.hasAttr(key) && value.equalsIgnoreCase(lowerCase(element.attr(key)));
         }
 
         @Override protected int cost() {
@@ -291,12 +291,12 @@ public abstract class Evaluator {
      */
     public static final class AttributeWithValueNot extends AttributeKeyPair {
         public AttributeWithValueNot(String key, String value) {
-            super(key, value);
+            super(key, value, false);
         }
 
         @Override
         public boolean matches(Element root, Element element) {
-            return !value.equalsIgnoreCase(element.attr(key));
+            return !value.equalsIgnoreCase(lowerCase(element.attr(key)));
         }
 
         @Override protected int cost() {
@@ -361,7 +361,7 @@ public abstract class Evaluator {
      */
     public static final class AttributeWithValueContaining extends AttributeKeyPair {
         public AttributeWithValueContaining(String key, String value) {
-            super(key, value);
+            super(key, value, false);
         }
 
         @Override
```

### with-lumen

**Rating: Poor**

```diff
diff --git a/src/main/java/org/jsoup/select/Evaluator.java b/src/main/java/org/jsoup/select/Evaluator.java
index 8c93fba..6df184d 100644
--- a/src/main/java/org/jsoup/select/Evaluator.java
+++ b/src/main/java/org/jsoup/select/Evaluator.java
@@ -267,12 +267,12 @@ public abstract class Evaluator {
      */
     public static final class AttributeWithValue extends AttributeKeyPair {
         public AttributeWithValue(String key, String value) {
-            super(key, value);
+            super(key, value, false);
         }
 
         @Override
         public boolean matches(Element root, Element element) {
-            return element.hasAttr(key) && value.equalsIgnoreCase(element.attr(key).trim());
+            return element.hasAttr(key) && value.equalsIgnoreCase(lowerCase(element.attr(key)));
         }
 
         @Override protected int cost() {
@@ -291,12 +291,12 @@ public abstract class Evaluator {
      */
     public static final class AttributeWithValueNot extends AttributeKeyPair {
         public AttributeWithValueNot(String key, String value) {
-            super(key, value);
+            super(key, value, false);
         }
 
         @Override
         public boolean matches(Element root, Element element) {
-            return !value.equalsIgnoreCase(element.attr(key));
+            return !value.equalsIgnoreCase(lowerCase(element.attr(key)));
         }
 
         @Override protected int cost() {
@@ -361,7 +361,7 @@ public abstract class Evaluator {
      */
     public static final class AttributeWithValueContaining extends AttributeKeyPair {
         public AttributeWithValueContaining(String key, String value) {
-            super(key, value);
+            super(key, value, false);
         }
 
         @Override
@@ -421,7 +421,7 @@ public abstract class Evaluator {
 
         public AttributeKeyPair(String key, String value, boolean trimQuoted) {
             Validate.notEmpty(key);
-            Validate.notEmpty(value);
+            Validate.notNull(value);
 
             this.key = normalize(key);
             boolean quoted = value.startsWith("'") && value.endsWith("'")
diff --git a/src/test/java/org/jsoup/select/SelectorTest.java b/src/test/java/org/jsoup/select/SelectorTest.java
index 6207a6c..a01c970 100644
--- a/src/test/java/org/jsoup/select/SelectorTest.java
+++ b/src/test/java/org/jsoup/select/SelectorTest.java
@@ -1047,6 +1047,48 @@ public class SelectorTest {
         assertEquals(1, els.size());
     }
 
+    @Test public void containsWithLeadingAndTrailingSpaces() {
+        Document doc = Jsoup.parse("<body><article xxx=\"xpaywalled\"></article></body>");
+        // Should NOT match because " paywalled " (with spaces) is not contained in "xpaywalled"
+        Elements els = doc.select("article[xxx*=\" paywalled \"]");
+        assertEquals(0, els.size());
+    }
+
+    @Test public void containsWithLeadingAndTrailingSpacesMatches() {
+        Document doc = Jsoup.parse("<body><article xxx=\"x paywalled y\"></article></body>");
+        // Should match because " paywalled " is contained in "x paywalled y"
+        Elements els = doc.select("article[xxx*=\" paywalled \"]");
+        assertEquals(1, els.size());
+    }
+
+    @Test public void exactMatchWithSpaces() {
+        Document doc = Jsoup.parse("<body><article xxx=\" test \"></article></body>");
+        // Should match exact value with spaces
+        Elements els = doc.select("article[xxx=\" test \"]");
+        assertEquals(1, els.size());
+    }
+
+    @Test public void exactMatchWithSpacesShouldNotMatch() {
+        Document doc = Jsoup.parse("<body><article xxx=\"test\"></article></body>");
+        // Should NOT match because "test" is not equal to " test "
+        Elements els = doc.select("article[xxx=\" test \"]");
+        assertEquals(0, els.size());
+    }
+
+    @Test public void emptyAttributeValue() {
+        Document doc = Jsoup.parse("<body><div data=\"\"></div></body>");
+        // Should match elements with empty attribute value
+        Elements els = doc.select("div[data='']");
+        assertEquals(1, els.size());
+    }
+
+    @Test public void emptyAttributeValueWithContains() {
+        Document doc = Jsoup.parse("<body><div data=\"\"></div></body>");
+        // Should match elements where attribute contains empty string
+        Elements els = doc.select("div[data*='']");
+        assertEquals(1, els.size());
+    }
+
     // https://github.com/jhy/jsoup/issues/1257
     private final String mixedCase =
         "<html xmlns:n=\"urn:ns\"><n:mixedCase>text</n:mixedCase></html>";
```

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
| **baseline** | 449.1s | 514 | 4007527 | 37897 | $0.8494 |
| **with-lumen** | 539.4s | 586 | 5425592 | 49360 | $0.9237 |

### baseline

**Rating: Poor**

```diff
diff --git a/crates/toml_edit/src/parser/document.rs b/crates/toml_edit/src/parser/document.rs
index 14f17fb..6054bf3 100644
--- a/crates/toml_edit/src/parser/document.rs
+++ b/crates/toml_edit/src/parser/document.rs
@@ -308,7 +308,7 @@ impl State {
         };
         // "Likewise, using dotted keys to redefine tables already defined in [table] form is not allowed"
         let mixed_table_types = parent_table.is_dotted() == path.is_empty();
-        if mixed_table_types {
+        if mixed_table_types && !parent_table.is_implicit() {
             let key_span = get_key_span(&key).expect("all keys have spans");
             errors.report_error(ParseError::new("duplicate key").with_unexpected(key_span));
             return;
diff --git a/crates/toml_edit/src/parser/inline_table.rs b/crates/toml_edit/src/parser/inline_table.rs
index 5bc556c..e07068b 100644
--- a/crates/toml_edit/src/parser/inline_table.rs
+++ b/crates/toml_edit/src/parser/inline_table.rs
@@ -193,7 +193,7 @@ impl State {
 
             // "Likewise, using dotted keys to redefine tables already defined in [table] form is not allowed"
             let mixed_table_types = table.is_dotted() == path.is_empty();
-            if mixed_table_types {
+            if mixed_table_types && !table.is_implicit() {
                 let key_span = get_key_span(&key).unwrap_or_else(|| event.span());
                 errors.report_error(ParseError::new("duplicate key").with_unexpected(key_span));
             } else {
diff --git a/crates/toml_edit/tests/testsuite/edit.rs b/crates/toml_edit/tests/testsuite/edit.rs
index 1ec8121..89006be 100644
--- a/crates/toml_edit/tests/testsuite/edit.rs
+++ b/crates/toml_edit/tests/testsuite/edit.rs
@@ -1681,3 +1681,72 @@ tool = { typst-test.tests = "tests" }
 "#]]
     );
 }
+
+#[test]
+fn implicit_table_with_dotted_keys() {
+    let toml = r#"[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+
+    let doc = toml.parse::<DocumentMut>();
+    assert!(doc.is_ok(), "Failed to parse: {:?}", doc.err());
+    let doc = doc.unwrap();
+
+    assert_eq!(doc["fruit"]["apple"]["color"].as_str(), Some("red"));
+    assert_eq!(doc["fruit"]["apple"]["taste"]["sweet"].as_bool(), Some(true));
+    assert_eq!(doc["fruit"]["apple"]["texture"]["smooth"].as_bool(), Some(true));
+}
+
+#[test]
+fn dotted_keys_before_nested_header() {
+    let toml = r#"[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+
+[fruit.apple.texture]
+smooth = true
+"#;
+
+    let doc = toml.parse::<DocumentMut>();
+    assert!(doc.is_ok(), "Failed to parse: {:?}", doc.err());
+    let doc = doc.unwrap();
+
+    assert_eq!(doc["fruit"]["apple"]["color"].as_str(), Some("red"));
+    assert_eq!(doc["fruit"]["apple"]["taste"]["sweet"].as_bool(), Some(true));
+    assert_eq!(doc["fruit"]["apple"]["texture"]["smooth"].as_bool(), Some(true));
+}
+
+#[test]
+fn nested_header_with_unrelated_key() {
+    let toml = r#"[fruit.apple.texture]
+smooth = true
+
+[fruit]
+banana = "yellow"
+"#;
+
+    let doc = toml.parse::<DocumentMut>();
+    assert!(doc.is_ok(), "Failed to parse: {:?}", doc.err());
+    let doc = doc.unwrap();
+
+    assert_eq!(doc["fruit"]["banana"].as_str(), Some("yellow"));
+    assert_eq!(doc["fruit"]["apple"]["texture"]["smooth"].as_bool(), Some(true));
+}
+
+#[test]
+fn reject_dotted_keys_on_explicit_inline_table() {
+    // This should fail: apple is defined as an inline table,
+    // then we try to use dotted keys to extend it
+    let toml = r#"[fruit]
+apple = { color = "red" }
+apple.taste = "sweet"
+"#;
+
+    let doc = toml.parse::<DocumentMut>();
+    // This should fail because we're trying to extend an explicitly-defined inline table with dotted keys
+    assert!(doc.is_err());
+}
diff --git a/crates/toml_edit/verify_fix.rs b/crates/toml_edit/verify_fix.rs
new file mode 100644
index 0000000..eda899f
--- /dev/null
+++ b/crates/toml_edit/verify_fix.rs
@@ -0,0 +1,20 @@
+fn main() {
+    let toml = r#"[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+
+    match toml.parse::<toml_edit::DocumentMut>() {
+        Ok(doc) => {
+            println!("✓ Parsed successfully!");
+            println!("\nDocument structure:");
+            println!("fruit.apple.color = {:?}", doc["fruit"]["apple"]["color"].as_str());
+            println!("fruit.apple.taste.sweet = {:?}", doc["fruit"]["apple"]["taste"]["sweet"].as_bool());
+            println!("fruit.apple.texture.smooth = {:?}", doc["fruit"]["apple"]["texture"]["smooth"].as_bool());
+        }
+        Err(e) => println!("✗ Parse error: {}", e),
+    }
+}
diff --git a/test_bug.rs b/test_bug.rs
new file mode 100644
index 0000000..f583148
--- /dev/null
+++ b/test_bug.rs
@@ -0,0 +1,15 @@
+fn main() {
+    let toml_string = r#"
+[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+
+    match toml_edit::parse(toml_string) {
+        Ok(doc) => println!("Parsed successfully: {:#?}", doc),
+        Err(e) => println!("Parse error: {}", e),
+    }
+}
diff --git a/test_dotted_implicit.rs b/test_dotted_implicit.rs
new file mode 100644
index 0000000..3557868
--- /dev/null
+++ b/test_dotted_implicit.rs
@@ -0,0 +1,20 @@
+#[test]
+fn test_dotted_keys_on_implicit_table() {
+    let toml_str = r#"
+[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+
+    let result = toml_str.parse::<toml_edit::DocumentMut>();
+    assert!(result.is_ok(), "Failed to parse: {:?}", result.err());
+    let doc = result.unwrap();
+    assert_eq!(doc["fruit"]["apple"]["color"].as_str(), Some("red"));
+    assert_eq!(doc["fruit"]["apple"]["taste"]["sweet"].as_bool(), Some(true));
+    assert_eq!(doc["fruit"]["apple"]["texture"]["smooth"].as_bool(), Some(true));
+}
+
+fn main() {}
diff --git a/test_issue.rs b/test_issue.rs
new file mode 100644
index 0000000..48fdc2f
--- /dev/null
+++ b/test_issue.rs
@@ -0,0 +1,21 @@
+fn main() {
+    let toml_str = r#"
+[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+
+    println!("Testing TOML parsing...");
+    match toml_str.parse::<toml_edit::DocumentMut>() {
+        Ok(doc) => {
+            println!("✓ Parsed successfully!");
+            println!("{}", doc);
+        }
+        Err(e) => {
+            println!("✗ Parse error: {}", e);
+        }
+    }
+}
diff --git a/test_issue.toml b/test_issue.toml
new file mode 100644
index 0000000..10beb5d
--- /dev/null
+++ b/test_issue.toml
@@ -0,0 +1,6 @@
+[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
```

### with-lumen

**Rating: Good**

```diff
diff --git a/crates/toml/src/de/parser/document.rs b/crates/toml/src/de/parser/document.rs
index 7f174b1..7277f11 100644
--- a/crates/toml/src/de/parser/document.rs
+++ b/crates/toml/src/de/parser/document.rs
@@ -224,8 +224,9 @@ impl<'i> State<'i> {
             return;
         };
         // "Likewise, using dotted keys to redefine tables already defined in [table] form is not allowed"
-        let mixed_table_types = parent_table.is_dotted() == path.is_empty();
-        if mixed_table_types {
+        // Only report an error if the parent table was explicitly defined (non-implicit) with a [table] header
+        // and we're trying to extend it with a dotted key path
+        if !parent_table.is_implicit() && !path.is_empty() {
             let key_span = get_key_span(&key);
             errors.report_error(ParseError::new("duplicate key").with_unexpected(key_span));
             return;
diff --git a/crates/toml_edit/src/parser/document.rs b/crates/toml_edit/src/parser/document.rs
index 14f17fb..54b411d 100644
--- a/crates/toml_edit/src/parser/document.rs
+++ b/crates/toml_edit/src/parser/document.rs
@@ -307,8 +307,9 @@ impl State {
             return;
         };
         // "Likewise, using dotted keys to redefine tables already defined in [table] form is not allowed"
-        let mixed_table_types = parent_table.is_dotted() == path.is_empty();
-        if mixed_table_types {
+        // Only report an error if the parent table was explicitly defined with a [table] header (has a position)
+        let is_explicitly_defined = parent_table.position().is_some();
+        if is_explicitly_defined && !path.is_empty() {
             let key_span = get_key_span(&key).expect("all keys have spans");
             errors.report_error(ParseError::new("duplicate key").with_unexpected(key_span));
             return;
@@ -478,7 +479,7 @@ fn descend_path<'t>(
                         // Since tables cannot be defined more than once, redefining such tables using a
                         // [table] header is not allowed. Likewise, using dotted keys to redefine tables
                         // already defined in [table] form is not allowed.
-                        if dotted && !sweet_child_of_mine.is_implicit() {
+                        if dotted && !sweet_child_of_mine.is_implicit() && sweet_child_of_mine.position().is_some() {
                             let key_span = get_key_span(key).expect("all keys have spans");
                             errors.report_error(
                                 ParseError::new("duplicate key").with_unexpected(key_span),
diff --git a/crates/toml_edit/tests/test_issue.rs b/crates/toml_edit/tests/test_issue.rs
new file mode 100644
index 0000000..1ad3799
--- /dev/null
+++ b/crates/toml_edit/tests/test_issue.rs
@@ -0,0 +1,15 @@
+#[test]
+fn test_dotted_key_extending_implicit_table() {
+    let toml = r#"[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+    
+    match toml.parse::<toml_edit::DocumentMut>() {
+        Ok(_) => println!("Test passed: parsed successfully"),
+        Err(e) => panic!("Parse error (should succeed): {}", e),
+    }
+}
diff --git a/crates/toml_edit/tests/testsuite/edit.rs b/crates/toml_edit/tests/testsuite/edit.rs
index 1ec8121..2a39fcf 100644
--- a/crates/toml_edit/tests/testsuite/edit.rs
+++ b/crates/toml_edit/tests/testsuite/edit.rs
@@ -1681,3 +1681,21 @@ tool = { typst-test.tests = "tests" }
 "#]]
     );
 }
+
+#[test]
+fn dotted_key_extending_implicit_table() {
+    // Test case from: False duplicate key error for dotted keys when parent table is implicitly created
+    // This should parse successfully - dotted keys should be allowed to extend implicitly created tables
+    let toml = r#"[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+
+    match toml.parse::<DocumentMut>() {
+        Ok(_) => {}, // Success - test passes
+        Err(e) => panic!("Should parse successfully, but got error: {}", e),
+    }
+}
```



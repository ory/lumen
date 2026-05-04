# SWE-Bench Detail Report

Generated: 2026-05-04 19:27 UTC

---

## go-hard [go]

**Issue:** Decoder overrides defaults with null values

> The decoder overrides the default values of a nested struct with zero-initialised values when the YAML node is null or contains only comments.
> 
> Given the following Go types:
> 
> ```go
> type Default struct {
>     Key string `yaml:"key"`
> }
> 
> type Test struct {
>     First    string  `yaml:"first"`
>     Defaults Default `yaml:"second"`
> }
> ```
> 
> And the following YAML input where the nested struct key is present but has no actual values (only comments):
> 
> ```yaml
> first: "Test"
> second:
>   # Just want to document the fields of the nested struct
> #  key: "Value from YAML"
> ```
> 
> When decoding into a pre-initialized struct with default values:
> 
> ```go
> test := Test{Defaults: Default{Key: "My default Value"}}
> d := yaml.NewDecoder(strings.NewReader(yml))
> err := d.Decode(&test)
> ```
> 
> The decoder zeroes out the nested struct, losing the pre-set default value. After decoding, `test.Defaults.Key` is an empty string instead of preserving the original default value "My default Value".
> 
> The standard library yaml.v3 preserves the default values in this scenario. The go-yaml decoder should also preserve struct defaults when the corresponding YAML node is null or contains no actual values.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 275.6s | 23 | 593288 | 12567 | $0.6775 |
| **with-lumen** | 210.4s | 24 | 561363 | 12236 | $0.5367 |

### baseline

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..34f5535 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,12 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		// For non-pointer struct types, preserve the caller-supplied default
+		// rather than zeroing it out. Pointer-typed fields are already handled
+		// before this call (set to nil by the caller).
+		if defaultVal.IsValid() && typ.Kind() == reflect.Struct {
+			return defaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
diff --git a/decode_test.go b/decode_test.go
index f0b0c08..0fa42f5 100644
--- a/decode_test.go
+++ b/decode_test.go
@@ -1945,6 +1945,42 @@ w: w_value
 	}
 }
 
+func TestDecoder_DefaultValuesWithNullNestedStruct(t *testing.T) {
+	type Inner struct {
+		Key string `yaml:"key"`
+	}
+	type Outer struct {
+		First  string `yaml:"first"`
+		Second Inner  `yaml:"second"`
+	}
+
+	tests := []struct {
+		name string
+		src  string
+	}{
+		{
+			name: "null node",
+			src:  "first: Test\nsecond:\n",
+		},
+		{
+			name: "comments only",
+			src:  "first: Test\nsecond:\n  # just a comment\n",
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			v := Outer{Second: Inner{Key: "My default Value"}}
+			if err := yaml.NewDecoder(strings.NewReader(tt.src)).Decode(&v); err != nil {
+				t.Fatalf("unexpected decode error: %s", err)
+			}
+			if v.Second.Key != "My default Value" {
+				t.Fatalf("v.Second.Key should be `My default Value`, got `%s`", v.Second.Key)
+			}
+		})
+	}
+}
+
 func Example_YAMLTags() {
 	yml := `---
 foo: 1
```

### with-lumen

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..831770c 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,17 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		if typ.Kind() == reflect.Struct {
+			// For struct types, a null node should not zero out the default value
+			// since structs are not nullable in Go. Preserve the default instead.
+			dv := defaultVal
+			for dv.Kind() == reflect.Ptr {
+				dv = dv.Elem()
+			}
+			if dv.IsValid() && dv.Type() == typ {
+				return dv, nil
+			}
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
diff --git a/decode_test.go b/decode_test.go
index f0b0c08..5929ffd 100644
--- a/decode_test.go
+++ b/decode_test.go
@@ -1945,6 +1945,47 @@ w: w_value
 	}
 }
 
+func TestDecoder_DefaultValuesWithNullNode(t *testing.T) {
+	type Default struct {
+		Key string `yaml:"key"`
+	}
+	type Test struct {
+		First    string  `yaml:"first"`
+		Defaults Default `yaml:"second"`
+	}
+
+	t.Run("null node from comment-only block", func(t *testing.T) {
+		yml := `first: "Test"
+second:
+  # Just want to document the fields of the nested struct
+#  key: "Value from YAML"
+`
+		test := Test{Defaults: Default{Key: "My default Value"}}
+		if err := yaml.NewDecoder(strings.NewReader(yml)).Decode(&test); err != nil {
+			t.Fatalf("parsing should succeed: %s", err)
+		}
+		if test.First != "Test" {
+			t.Fatalf("test.First should be `Test`, got `%s`", test.First)
+		}
+		if test.Defaults.Key != "My default Value" {
+			t.Fatalf("test.Defaults.Key should be `My default Value`, got `%s`", test.Defaults.Key)
+		}
+	})
+
+	t.Run("explicit null node", func(t *testing.T) {
+		yml := `first: "Test"
+second:
+`
+		test := Test{Defaults: Default{Key: "My default Value"}}
+		if err := yaml.NewDecoder(strings.NewReader(yml)).Decode(&test); err != nil {
+			t.Fatalf("parsing should succeed: %s", err)
+		}
+		if test.Defaults.Key != "My default Value" {
+			t.Fatalf("test.Defaults.Key should be `My default Value`, got `%s`", test.Defaults.Key)
+		}
+	})
+}
+
 func Example_YAMLTags() {
 	yml := `---
 foo: 1
```



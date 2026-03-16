# SWE-Bench Detail Report

Generated: 2026-03-15 21:47 UTC

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
| **baseline** | 221.9s | 546 | 3813270 | 21634 | $0.5966 |
| **with-lumen** | 176.6s | 346 | 2721063 | 15251 | $0.4227 |

### baseline

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..f68cbc4 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,17 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		// If we have a valid default value, preserve it instead of returning zero
+		targetType := typ
+		for targetType.Kind() == reflect.Ptr {
+			targetType = targetType.Elem()
+		}
+		for defaultVal.Kind() == reflect.Ptr {
+			defaultVal = defaultVal.Elem()
+		}
+		if defaultVal.IsValid() && defaultVal.Type().AssignableTo(targetType) {
+			return defaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
diff --git a/decode_test.go b/decode_test.go
index f0b0c08..8737394 100644
--- a/decode_test.go
+++ b/decode_test.go
@@ -3027,3 +3027,66 @@ func TestMapKeyCustomUnmarshaler(t *testing.T) {
 		t.Fatalf("expected to have value \"value\", but got %q", val)
 	}
 }
+
+func TestDecoderPreservesDefaultsWithNullNestedStruct(t *testing.T) {
+	type Default struct {
+		Key string `yaml:"key"`
+	}
+
+	type Test struct {
+		First    string  `yaml:"first"`
+		Defaults Default `yaml:"second"`
+	}
+
+	yml := `first: "Test"
+second:
+  # Just want to document the fields of the nested struct
+`
+
+	test := Test{Defaults: Default{Key: "My default Value"}}
+	d := yaml.NewDecoder(strings.NewReader(yml))
+	err := d.Decode(&test)
+
+	if err != nil {
+		t.Fatalf("decode should succeed: %v", err)
+	}
+
+	if test.Defaults.Key != "My default Value" {
+		t.Fatalf("expected 'My default Value', got '%s'", test.Defaults.Key)
+	}
+
+	if test.First != "Test" {
+		t.Fatalf("expected 'Test', got '%s'", test.First)
+	}
+}
+
+func TestDecoderPreservesDefaultsWithExplicitNull(t *testing.T) {
+	type Default struct {
+		Key string `yaml:"key"`
+	}
+
+	type Test struct {
+		First    string  `yaml:"first"`
+		Defaults Default `yaml:"second"`
+	}
+
+	yml := `first: "Test"
+second: null
+`
+
+	test := Test{Defaults: Default{Key: "My default Value"}}
+	d := yaml.NewDecoder(strings.NewReader(yml))
+	err := d.Decode(&test)
+
+	if err != nil {
+		t.Fatalf("decode should succeed: %v", err)
+	}
+
+	if test.Defaults.Key != "My default Value" {
+		t.Fatalf("expected 'My default Value', got '%s'", test.Defaults.Key)
+	}
+
+	if test.First != "Test" {
+		t.Fatalf("expected 'Test', got '%s'", test.First)
+	}
+}
```

### with-lumen

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..21cc794 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,14 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		// Preserve the default value when the node is null
+		derefDefaultVal := defaultVal
+		for derefDefaultVal.Kind() == reflect.Ptr {
+			derefDefaultVal = derefDefaultVal.Elem()
+		}
+		if derefDefaultVal.IsValid() && derefDefaultVal.Type().AssignableTo(typ) {
+			return derefDefaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
diff --git a/decode_test.go b/decode_test.go
index f0b0c08..b44706f 100644
--- a/decode_test.go
+++ b/decode_test.go
@@ -3027,3 +3027,37 @@ func TestMapKeyCustomUnmarshaler(t *testing.T) {
 		t.Fatalf("expected to have value \"value\", but got %q", val)
 	}
 }
+
+func TestDecoder_DefaultValuesWithNullNode(t *testing.T) {
+	type Default struct {
+		Key string `yaml:"key"`
+	}
+
+	type Test struct {
+		First    string  `yaml:"first"`
+		Defaults Default `yaml:"second"`
+	}
+
+	test := Test{Defaults: Default{Key: "My default Value"}}
+
+	yml := `first: "Test"
+second:
+  # Just want to document the fields of the nested struct
+#  key: "Value from YAML"
+`
+
+	d := yaml.NewDecoder(strings.NewReader(yml))
+	err := d.Decode(&test)
+
+	if err != nil {
+		t.Fatalf("Decode failed: %v", err)
+	}
+
+	if test.First != "Test" {
+		t.Errorf("Expected test.First to be 'Test', got '%s'", test.First)
+	}
+
+	if test.Defaults.Key != "My default Value" {
+		t.Errorf("Expected test.Defaults.Key to be 'My default Value', got '%s'", test.Defaults.Key)
+	}
+}
```



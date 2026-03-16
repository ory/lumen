# SWE-Bench Detail Report

Generated: 2026-03-15 22:01 UTC

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
| **baseline** | 253.0s | 18 | 455681 | 7212 | $0.6500 |
| **with-lumen** | 154.5s | 16 | 279914 | 5235 | $0.4151 |

### baseline

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..733c0fd 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,9 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		if typ.Kind() == reflect.Struct && defaultVal.IsValid() {
+			return defaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
diff --git a/decode_test.go b/decode_test.go
index f0b0c08..2c50bf0 100644
--- a/decode_test.go
+++ b/decode_test.go
@@ -1945,6 +1945,33 @@ w: w_value
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
+	yml := `
+first: "Test"
+second:
+  # Just want to document the fields of the nested struct
+#  key: "Value from YAML"
+`
+	test := Test{Defaults: Default{Key: "My default Value"}}
+	if err := yaml.NewDecoder(strings.NewReader(yml)).Decode(&test); err != nil {
+		t.Fatalf("unexpected error: %s", err)
+	}
+	if test.First != "Test" {
+		t.Fatalf("test.First should be `Test`, got `%s`", test.First)
+	}
+	if test.Defaults.Key != "My default Value" {
+		t.Fatalf("test.Defaults.Key should be `My default Value`, got `%s`", test.Defaults.Key)
+	}
+}
+
 func Example_YAMLTags() {
 	yml := `---
 foo: 1
```

### with-lumen

**Rating: Poor**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..733c0fd 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,9 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		if typ.Kind() == reflect.Struct && defaultVal.IsValid() {
+			return defaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
```



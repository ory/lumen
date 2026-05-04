# SWE-Bench Detail Report

Generated: 2026-05-04 22:28 UTC

---

## python-hard [python]

**Issue:** New 'default' behaviour in Click 8.3.x is broken for negative boolean flags

> The behaviour of `default=...` for flag values changed in Click 8.3.0, as documented in the release notes. The new behaviour is broken for negative flags, unless I am holding it wrong.
> 
> Here's a minimal repro:
> 
> ```python
> import click
> 
> @click.command('foo')
> @click.option('--without-xyz', 'enable_xyz',
>               help="Disable xyz", flag_value=False, default=True, show_default=True)
> def foo(enable_xyz):
>     print(f'enable_xyz = {enable_xyz}')
> 
> foo()
> ```
> 
> Expected (Click 8.2.x):
> - `./foo.py` → `enable_xyz = True`
> - `./foo.py --without-xyz` → `enable_xyz = False`
> 
> Actual (Click 8.3.0+):
> - `./foo.py` → `enable_xyz = False`
> - `./foo.py --without-xyz` → `enable_xyz = False`
> 
> So the default value of `True` is being silently replaced by the `flag_value` of `False`, making the flag completely useless — it always returns `False` regardless of whether it was passed or not.
> 
> This is a regression from Click 8.2.x where `default=True` was respected as a literal Python value for boolean flags.
> 
> Questions:
> 1. Is the `default=True` special case truly necessary?
> 2. Should it produce an explicit error rather than a silent behavioral change?
> 3. What is the canonical approach for negative options that default to "off"?
> 4. Would improved documentation help?
> 
> Environment: Python 3.10, Click 8.3.x

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 268.6s | 22 | 505106 | 13349 | $0.5774 |
| **with-lumen** | 416.2s | 25 | 468043 | 13495 | $0.5612 |

### baseline

**Rating: Good**

```diff
diff --git a/src/click/core.py b/src/click/core.py
index 6dc44f3..3b73211 100644
--- a/src/click/core.py
+++ b/src/click/core.py
@@ -2897,7 +2897,9 @@ class Option(Parameter):
         # (instead of eagerly in __init__) prevents callable flag_values
         # (like classes) from being instantiated by the callable check below.
         # https://github.com/pallets/click/issues/3121
-        if value is True and self.is_flag:
+        # Skip the mapping when flag_value is False: the user explicitly set
+        # default=True to mean the literal Python True, not "flag is active".
+        if value is True and self.is_flag and self.flag_value is not False:
             value = self.flag_value
         elif call and callable(value):
             value = value()
diff --git a/tests/test_options.py b/tests/test_options.py
index 85f6224..829a3fb 100644
--- a/tests/test_options.py
+++ b/tests/test_options.py
@@ -1459,13 +1459,13 @@ def test_type_from_flag_value():
         # Not passing --foo returns the default value as-is, in its Python type, then
         # converted by the option type.
         ({"type": bool, "default": True, "flag_value": True}, [], True),
-        ({"type": bool, "default": True, "flag_value": False}, [], False),
+        ({"type": bool, "default": True, "flag_value": False}, [], True),
         ({"type": bool, "default": False, "flag_value": True}, [], False),
         ({"type": bool, "default": False, "flag_value": False}, [], False),
         ({"type": bool, "default": None, "flag_value": True}, [], None),
         ({"type": bool, "default": None, "flag_value": False}, [], None),
         ({"type": str, "default": True, "flag_value": True}, [], "True"),
-        ({"type": str, "default": True, "flag_value": False}, [], "False"),
+        ({"type": str, "default": True, "flag_value": False}, [], "True"),
         ({"type": str, "default": False, "flag_value": True}, [], "False"),
         ({"type": str, "default": False, "flag_value": False}, [], "False"),
         ({"type": str, "default": "foo", "flag_value": True}, [], "foo"),
```

### with-lumen

**Rating: Good**

```diff
diff --git a/src/click/core.py b/src/click/core.py
index 6dc44f3..037b276 100644
--- a/src/click/core.py
+++ b/src/click/core.py
@@ -2897,7 +2897,7 @@ class Option(Parameter):
         # (instead of eagerly in __init__) prevents callable flag_values
         # (like classes) from being instantiated by the callable check below.
         # https://github.com/pallets/click/issues/3121
-        if value is True and self.is_flag:
+        if value is True and self.is_flag and not self.is_bool_flag:
             value = self.flag_value
         elif call and callable(value):
             value = value()
diff --git a/tests/test_options.py b/tests/test_options.py
index 85f6224..7dda50e 100644
--- a/tests/test_options.py
+++ b/tests/test_options.py
@@ -1459,7 +1459,7 @@ def test_type_from_flag_value():
         # Not passing --foo returns the default value as-is, in its Python type, then
         # converted by the option type.
         ({"type": bool, "default": True, "flag_value": True}, [], True),
-        ({"type": bool, "default": True, "flag_value": False}, [], False),
+        ({"type": bool, "default": True, "flag_value": False}, [], True),
         ({"type": bool, "default": False, "flag_value": True}, [], False),
         ({"type": bool, "default": False, "flag_value": False}, [], False),
         ({"type": bool, "default": None, "flag_value": True}, [], None),
diff --git a/tests/test_termui.py b/tests/test_termui.py
index be0c2df..74ddf7b 100644
--- a/tests/test_termui.py
+++ b/tests/test_termui.py
@@ -825,9 +825,9 @@ FLAG_VALUE_PROMPT_CASES = [
     ({"prompt": True, "default": True, "flag_value": True}, [], "[Y/n]", "", True),
     ({"prompt": True, "default": True, "flag_value": True}, [], "[Y/n]", "y", True),
     ({"prompt": True, "default": True, "flag_value": True}, [], "[Y/n]", "n", False),
-    ({"prompt": True, "default": True, "flag_value": False}, [], "[y/N]", "", False),
-    ({"prompt": True, "default": True, "flag_value": False}, [], "[y/N]", "y", True),
-    ({"prompt": True, "default": True, "flag_value": False}, [], "[y/N]", "n", False),
+    ({"prompt": True, "default": True, "flag_value": False}, [], "[Y/n]", "", True),
+    ({"prompt": True, "default": True, "flag_value": False}, [], "[Y/n]", "y", True),
+    ({"prompt": True, "default": True, "flag_value": False}, [], "[Y/n]", "n", False),
     # default=False
     ({"prompt": True, "default": False, "flag_value": True}, [], "[y/N]", "", False),
     ({"prompt": True, "default": False, "flag_value": True}, [], "[y/N]", "y", True),
```



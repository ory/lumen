# SWE-Bench Detail Report

Generated: 2026-03-10 20:25 UTC

---

## python-hard [python]

**Issue:** Flag option with secondary opts: show_default=True does not show value from default_map in help output

> When setting the `default_map` from a config file, the values from the `default_map` are not correctly shown in the `--help` output when `show_default=True` is set on a boolean flag option with secondary opts.
> 
> Given an option defined like this:
> 
> ```python
> @click.option(
>     "--long/--short",
>     "-l/-s",
>     is_flag=True,
>     show_default=True,
>     help="show additional information like size and creation date",
> )
> ```
> 
> And a custom command class that sets a `default_map` via `context_settings`:
> 
> ```python
> class ConfigAwareCommand(click.Command):
>     def __init__(self, *args, **kwargs):
>         kwargs["context_settings"] = {
>             "default_map": {"long": True}
>         }
>         super().__init__(*args, **kwargs)
> ```
> 
> The `default_map` value is `{'long': True}`. When executing the command, the default value from the `default_map` is used correctly (`long` defaults to `True`). However, in the `--help` output, the default value is shown incorrectly:
> 
> ```
>   -l, --long / -s, --short        show additional information like size and
>                                   creation date  [default: short]
> ```
> 
> The help text shows `[default: short]` instead of `[default: long]`, because the help rendering for boolean flags with secondary opts uses the option's own default attribute rather than the resolved default value that accounts for the `default_map`.
> 
> Environment:
> - Python version: 3.10.12
> - Click version: 8.1.7

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 43.0s | 9 | 132115 | 1710 | $0.1188 |
| **with-lumen** | 30.6s | 9 | 89688 | 1092 | $0.0956 |

### baseline

**Rating: Perfect**

```diff
diff --git a/src/click/core.py b/src/click/core.py
index 367beb2..ab7d452 100644
--- a/src/click/core.py
+++ b/src/click/core.py
@@ -2800,7 +2800,7 @@ class Option(Parameter):
                 # For boolean flags that have distinct True/False opts,
                 # use the opt without prefix instead of the value.
                 default_string = split_opt(
-                    (self.opts if self.default else self.secondary_opts)[0]
+                    (self.opts if default_value else self.secondary_opts)[0]
                 )[1]
             elif self.is_bool_flag and not self.secondary_opts and not default_value:
                 default_string = ""
```

### with-lumen

**Rating: Perfect**

```diff
diff --git a/src/click/core.py b/src/click/core.py
index 367beb2..ab7d452 100644
--- a/src/click/core.py
+++ b/src/click/core.py
@@ -2800,7 +2800,7 @@ class Option(Parameter):
                 # For boolean flags that have distinct True/False opts,
                 # use the opt without prefix instead of the value.
                 default_string = split_opt(
-                    (self.opts if self.default else self.secondary_opts)[0]
+                    (self.opts if default_value else self.secondary_opts)[0]
                 )[1]
             elif self.is_bool_flag and not self.secondary_opts and not default_value:
                 default_string = ""
```



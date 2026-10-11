## Rating: Perfect

The candidate patch makes the exact same one-line fix to `src/click/core.py` as the gold patch — replacing `self.default` with `default_value` so the help text uses the resolved default (which accounts for `default_map`) rather than the option's own unresolved default attribute. The candidate omits the `CHANGES.rst` entry and the new test in `tests/test_defaults.py`, but the functional fix is identical and correct.

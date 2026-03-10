## Rating: Perfect

The candidate patch makes exactly the same code change as the gold patch in `src/click/core.py` — replacing `self.default` with `default_value` so the boolean flag help rendering uses the resolved default (which accounts for `default_map`) rather than the option's own default attribute. The only difference is the candidate omits the `CHANGES.rst` update and the new test in `tests/test_defaults.py`, but the core logic fix is identical and correct.

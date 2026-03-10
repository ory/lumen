## Rating: Perfect

The candidate patch makes the exact same one-line change as the gold patch in `src/click/core.py`, replacing `self.default` with `default_value` so the resolved default (including `default_map`) is used instead of the option's raw default attribute. The only difference is the candidate omits the `CHANGES.rst` update and the new test case, but the core logic fix is identical and correct.

## Rating: Good

The candidate applies the identical logical fix — returning early from `format_from_header` when `accept_header == '*/*'` — which correctly resolves the regression. The only meaningful differences are that the gold patch extracts the string into a named constant `ALL_MEDIA_TYPES` and updates the accompanying spec, while the candidate inlines the literal and adds a CHANGELOG entry instead of updating the test. The fix is functionally equivalent and correctly addresses the issue.

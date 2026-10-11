# Authentication

Widget sessions use short-lived access tokens. When a request fails with
`401`, the client refreshes the token once and retries.

## Refreshing tokens manually

Scripts that call the API directly can reuse the same helper:

```ts
import { refreshAccessToken } from "@acme/widget/auth";

const session = await refreshAccessToken(store, { force: true });
console.log(session.expiresAt);
```

Pass `force: true` to refresh even when the current token has not expired.

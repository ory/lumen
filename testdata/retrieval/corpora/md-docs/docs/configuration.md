# Configuration reference

Settings live in `widget.config.json` at the project root. Every key is
optional; the defaults are shown below.

## Network

### Requests

#### Request timeout

`timeoutMs` bounds a single HTTP request to the control plane. The default is
`10000`. Long-running uploads ignore this value and use `uploadTimeoutMs`.

#### Retry policy

Failed requests are retried up to `maxRetries` times (default `3`). Only
idempotent requests and responses with status 502, 503 or 504 are retried.

##### Backoff

The delay before attempt *n* is `baseDelayMs * backoffMultiplier ^ n`, capped at
`maxDelayMs`. The defaults are `250`, `2` and `8000`.

###### Jitter

Each delay is randomised by up to `jitterRatio` (default `0.2`) of its value so
that many clients recovering from the same outage do not retry in lockstep.

## Logging

Set `logLevel` to `debug`, `info`, `warn` or `error`. Logs are written to
stderr; set `logFile` to also append them to a file.

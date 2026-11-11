2026-11-11T01:50:32-03:00

# OpenCode V2 canary readiness timeout

## Causal chain

The canary gives `waitForPlugin` one 30-second deadline to observe both the
package-backed plugin and its MCP server becoming ready. With the documented
package specifier, OpenCode may spend part of that window resolving the package
before starting the local MCP process. OpenCode V2.2.7 separately allows 30
seconds for the MCP startup handshake. If package resolution and the handshake
together exceed the canary's 30-second budget, `waitForPlugin` throws and the
canary's cleanup stops the host even though the MCP handshake may still be
within OpenCode's own deadline. This is a potential cold-start test failure,
not evidence of a plugin runtime failure.

## Correction

The canary now allows 60 seconds for plugin and MCP readiness, enough for
package resolution followed by OpenCode's full 30-second handshake window. The
OpenCode handshake timeout and the separate host-start/API request timeouts
remain unchanged.

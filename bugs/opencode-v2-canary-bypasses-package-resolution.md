2026-11-11T01:25:53-03:00

# OpenCode V2 canary bypasses package resolution

## Causal chain

The installation test packed and installed `@ory/lumen-opencode`, then passed
that directory to `scripts/test_opencode_v2_host.mjs`. The canary symlinked the
directory into its temporary OpenCode plugin folder. OpenCode therefore loaded
the local plugin directly; it did not read a project `plugins` entry, resolve
the package specifier, or prove that package installation selected the packed
artifact. The test only checked that the plugin became active and its MCP
server connected, so it could pass even if the documented package-install
path was broken.

## Boundary and correction

This was a false-positive risk in the canary, not evidence that normal package
installation failed. The canary now configures `plugins: ["@ory/lumen-opencode"]`
in an isolated project, resolves it through a loopback registry serving the
packed tarball, checks OpenCode's reported package source and registry request,
and compares the loaded package files with the npm-installed artifact before
accepting MCP connectivity.

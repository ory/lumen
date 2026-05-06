#!/usr/bin/env node
// Spawns the lumen native binary matching the host OS/arch and forwards
// stdin/stdout/stderr + exit code. Replaces the scripts/run.{sh,bat,cmd}
// polyglot launcher. This Phase 1 file is intentionally minimal: it only
// resolves a binary already present at <pluginRoot>/bin/ (or via the
// LUMEN_BIN_PATH env override). Lazy fetch from GitHub Releases, SHA-256
// checksum verification, file lock, and per-host cache resolution land in
// Phase 2.

import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const ext = process.platform === "win32" ? ".exe" : "";

function findBinary() {
  // Explicit override — used by integration tests and (future) Phase 2 cache
  // resolution that wants to bypass the search.
  if (process.env.LUMEN_BIN_PATH) {
    return process.env.LUMEN_BIN_PATH;
  }
  const candidates = [
    path.join(here, "bin", `lumen${ext}`),
    path.join(here, "bin", `lumen-${process.platform}-${process.arch}${ext}`),
  ];
  for (const c of candidates) {
    try {
      fs.accessSync(c, fs.constants.X_OK);
      return c;
    } catch {
      // not present, try next
    }
  }
  return null;
}

const bin = findBinary();
if (!bin) {
  process.stderr.write(
    `lumen launcher: no binary found. Set LUMEN_BIN_PATH or build with \`make build-local\`.\n`,
  );
  process.exit(127);
}

const child = spawn(bin, process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code ?? 0);
  }
});

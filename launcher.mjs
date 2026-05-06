#!/usr/bin/env node
// Spawns the lumen native binary matching the host OS/arch, lazy-fetching it
// from GitHub Releases on first use into a per-host cache. Replaces the
// scripts/run.{sh,bat,cmd} polyglot.
//
// Resolution order for the binary:
//   1. $LUMEN_BIN_PATH                            — explicit override
//   2. <pluginRoot>/bin/lumen[.exe]               — `make build-local` output
//   3. <cacheDir>/bin/<version>/lumen-<version>-<os>-<arch>[.exe]
//                                                 — lazy-fetched + SHA-256-verified
//
// Cache dir resolution chain: $CLAUDE_PLUGIN_DATA → $CURSOR_PLUGIN_DATA →
// $XDG_DATA_HOME (Unix) / $LOCALAPPDATA (Windows) → ~/.local/share / tmpdir.
//
// Subcommand `prefetch` runs the cache-population flow then exits 0; used by
// SessionStart hooks (Claude Code) and the OpenCode plugin entry to warm
// the cache before the MCP server is invoked, working around tight
// startup-handshake timeouts on OpenCode (5 s) and Codex (10 s default).

import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import * as fs from "node:fs";
import { readFile } from "node:fs/promises";
import { request as httpsRequest } from "node:https";
import { homedir, tmpdir } from "node:os";
import path from "node:path";
import { setTimeout as sleep } from "node:timers/promises";
import { fileURLToPath } from "node:url";

const REPO = "ory/lumen";
// 15 min covers a 30 MB download even at trickle speeds (~30 KB/s) before
// another prefetch process treats the lock as stale and races the writer.
const STALE_LOCK_MS = 15 * 60 * 1000;
// Per-socket inactivity timeout. Kills the connection if no data has flowed
// for HTTP_IDLE_MS — guards against hung connections, ISP throttling, and
// CDN mid-stream stalls. Active downloads keep flowing data so this never
// fires for healthy fetches.
const HTTP_IDLE_MS = 60 * 1000;
const VERBOSE = process.env.LUMEN_LAUNCHER_VERBOSE === "1";

const here = path.dirname(fileURLToPath(import.meta.url));
const ext = process.platform === "win32" ? ".exe" : "";

function log(...args) {
  if (VERBOSE) process.stderr.write(`[lumen-launcher] ${args.join(" ")}\n`);
}
function fatal(msg, code = 1) {
  process.stderr.write(`lumen launcher: ${msg}\n`);
  process.exit(code);
}

async function readVersion() {
  const manifestPath = path.join(here, ".release-please-manifest.json");
  const raw = await readFile(manifestPath, "utf8");
  const manifest = JSON.parse(raw);
  const version = manifest["."];
  if (!version || !/^\d/.test(version)) {
    throw new Error(
      `unexpected ${manifestPath} content: ${JSON.stringify(manifest)}`,
    );
  }
  return version;
}

function resolveCacheDir() {
  for (const env of ["CLAUDE_PLUGIN_DATA", "CURSOR_PLUGIN_DATA"]) {
    if (process.env[env]) return path.join(process.env[env], "lumen");
  }
  if (process.platform === "win32") {
    if (process.env.LOCALAPPDATA) {
      return path.join(process.env.LOCALAPPDATA, "lumen");
    }
  } else {
    if (process.env.XDG_DATA_HOME) {
      return path.join(process.env.XDG_DATA_HOME, "lumen");
    }
    return path.join(homedir(), ".local", "share", "lumen");
  }
  return path.join(tmpdir(), "lumen-cache");
}

// Node's process.platform / process.arch (darwin/win32, x64/arm64) differs
// from Go's GOOS / GOARCH (darwin/windows, amd64/arm64) used by goreleaser
// when naming release artifacts. Map between them.
function goos() {
  return { darwin: "darwin", linux: "linux", win32: "windows" }[process.platform] ?? process.platform;
}
function goarch() {
  return { x64: "amd64", arm64: "arm64", ia32: "386" }[process.arch] ?? process.arch;
}

function assetName(version) {
  return `lumen-${version}-${goos()}-${goarch()}${ext}`;
}

function cachedBinaryPath(cacheDir, version) {
  return path.join(cacheDir, "bin", version, assetName(version));
}

// Follows redirects up to 5 hops; resolves with the IncomingMessage on 200,
// rejects on error or non-2xx. Per-socket inactivity timeout fires on
// hung connect, ISP throttling, or CDN mid-stream stalls (no data for
// HTTP_IDLE_MS).
function httpsGet(url, hops = 0) {
  return new Promise((resolve, reject) => {
    if (hops > 5) return reject(new Error("too many redirects"));
    const req = httpsRequest(
      url,
      {
        method: "GET",
        headers: { "User-Agent": "lumen-launcher" },
        timeout: HTTP_IDLE_MS,
      },
      (res) => {
        const code = res.statusCode || 0;
        if (code >= 300 && code < 400 && res.headers.location) {
          res.destroy();
          resolve(
            httpsGet(new URL(res.headers.location, url).toString(), hops + 1),
          );
          return;
        }
        if (code !== 200) {
          res.destroy();
          reject(new Error(`HTTP ${code} for ${url}`));
          return;
        }
        // Idle-timeout the response stream too; req.timeout only covers
        // pre-headers, while a stalled body stream blocks downloadAndVerify.
        res.setTimeout(HTTP_IDLE_MS, () => {
          res.destroy(new Error(`response idle timeout for ${url}`));
        });
        resolve(res);
      },
    );
    req.on("timeout", () => {
      req.destroy(new Error(`request timeout for ${url}`));
    });
    req.on("error", reject);
    req.end();
  });
}

async function fetchText(url) {
  const res = await httpsGet(url);
  let buf = "";
  for await (const chunk of res) buf += chunk;
  return buf;
}

function parseChecksum(text, fileName) {
  // goreleaser checksums.txt: "<sha256>  <filename>" per line
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (!line) continue;
    const m = line.match(/^([0-9a-f]{64})\s+(.+)$/);
    if (m && m[2] === fileName) return m[1];
  }
  return null;
}

async function downloadAndVerify(url, expectedSha256, destPath) {
  const tmpPath = `${destPath}.partial`;
  fs.mkdirSync(path.dirname(destPath), { recursive: true });
  const res = await httpsGet(url);
  const hash = createHash("sha256");
  try {
    await new Promise((resolve, reject) => {
      const out = fs.createWriteStream(tmpPath);
      const fail = (err) => {
        res.destroy();
        out.destroy();
        reject(err);
      };
      res.on("data", (chunk) => hash.update(chunk));
      res.on("error", fail);
      out.on("error", fail);
      out.on("finish", resolve);
      res.pipe(out);
    });
  } catch (e) {
    // Mid-stream network reset / idle-timeout / write error — drop the
    // partial file so the next run starts clean instead of relying on
    // createWriteStream's truncate-on-open.
    try {
      fs.unlinkSync(tmpPath);
    } catch {}
    throw e;
  }
  const actual = hash.digest("hex");
  if (actual !== expectedSha256) {
    try {
      fs.unlinkSync(tmpPath);
    } catch {}
    throw new Error(
      `SHA-256 mismatch for ${path.basename(destPath)}: expected ${expectedSha256}, got ${actual}`,
    );
  }
  fs.chmodSync(tmpPath, 0o755);
  fs.renameSync(tmpPath, destPath);
}

async function withLock(lockPath, fn) {
  const start = Date.now();
  fs.mkdirSync(path.dirname(lockPath), { recursive: true });
  while (true) {
    try {
      // 'wx' is atomic-create-fail-if-exists on both POSIX and Windows.
      const fd = fs.openSync(lockPath, "wx");
      try {
        fs.writeSync(fd, String(process.pid));
      } finally {
        fs.closeSync(fd);
      }
      try {
        return await fn();
      } finally {
        try {
          fs.unlinkSync(lockPath);
        } catch {}
      }
    } catch (e) {
      if (e.code !== "EEXIST") throw e;
    }
    // Held — check staleness, reclaim, or wait briefly and retry.
    try {
      const st = fs.statSync(lockPath);
      if (Date.now() - st.mtimeMs > STALE_LOCK_MS) {
        log(`reclaiming stale lock at ${lockPath}`);
        try {
          fs.unlinkSync(lockPath);
        } catch {}
        continue;
      }
    } catch (e) {
      // ENOENT: file disappeared between EEXIST and stat (POSIX race).
      // EPERM: Windows transient state while another process is unlinking
      // the lock — Windows file-sharing semantics differ from POSIX and
      // briefly surface stat as not-permitted. Both mean "lock is gone";
      // retry the openSync to claim it.
      if (e.code === "ENOENT" || e.code === "EPERM") continue;
      throw e;
    }
    if (Date.now() - start > STALE_LOCK_MS) {
      throw new Error(`timeout waiting for lock at ${lockPath}`);
    }
    await sleep(200);
  }
}

async function ensureBinary() {
  if (process.env.LUMEN_BIN_PATH) return process.env.LUMEN_BIN_PATH;

  // Developer fast path: `make build-local` writes here.
  const dev = path.join(here, "bin", `lumen${ext}`);
  try {
    fs.accessSync(dev, fs.constants.X_OK);
    return dev;
  } catch {}

  const version = await readVersion();
  const cacheDir = resolveCacheDir();
  const binPath = cachedBinaryPath(cacheDir, version);

  try {
    fs.accessSync(binPath, fs.constants.X_OK);
    return binPath;
  } catch {}

  const lockPath = path.join(cacheDir, ".lock");
  await withLock(lockPath, async () => {
    // Re-check under lock — another process may have completed it while we waited.
    try {
      fs.accessSync(binPath, fs.constants.X_OK);
      return;
    } catch {}
    log(`fetching lumen ${version} for ${process.platform}-${process.arch}`);
    const releaseURL = `https://github.com/${REPO}/releases/download/v${version}`;
    const aName = assetName(version);
    const checksums = await fetchText(`${releaseURL}/checksums.txt`);
    const expected = parseChecksum(checksums, aName);
    if (!expected) {
      throw new Error(
        `no SHA-256 for ${aName} in ${releaseURL}/checksums.txt`,
      );
    }
    await downloadAndVerify(`${releaseURL}/${aName}`, expected, binPath);
    log(`installed lumen at ${binPath}`);
  });
  return binPath;
}

async function main() {
  const args = process.argv.slice(2);
  let bin;
  try {
    bin = await ensureBinary();
  } catch (e) {
    fatal(`failed to locate or fetch lumen binary: ${e.message}`, 2);
  }
  if (args[0] === "prefetch") {
    process.exit(0);
  }
  const child = spawn(bin, args, { stdio: "inherit" });

  // Forward SIGINT/SIGTERM/SIGHUP/SIGQUIT to the child so signals delivered
  // only to the launcher PID (e.g. supervisor SIGTERM, IDE-driven shutdown,
  // `kill <pid>`) don't orphan the lumen process — which would leak its
  // sqlite advisory lock and keep its background indexer alive.
  // Windows process model has no equivalent — only SIGINT/SIGBREAK are
  // really meaningful and Node treats child.kill on win32 as TerminateProcess.
  const forwardable =
    process.platform === "win32" ? [] : ["SIGINT", "SIGTERM", "SIGHUP", "SIGQUIT"];
  const onSignal = (sig) => {
    try {
      child.kill(sig);
    } catch {}
  };
  for (const sig of forwardable) process.on(sig, onSignal);

  // Both 'error' (failed to spawn) and 'exit' (child terminated) can fire;
  // the previous handlers double-called process.exit / fatal. Settle once.
  let settled = false;
  const settle = (fn) => {
    if (settled) return;
    settled = true;
    for (const sig of forwardable) process.off(sig, onSignal);
    fn();
  };

  child.on("error", (err) => {
    settle(() => fatal(`failed to spawn binary: ${err.message}`, 126));
  });
  child.on("exit", (code, signal) => {
    settle(() => {
      if (signal && process.platform !== "win32") {
        process.kill(process.pid, signal);
      } else {
        process.exit(code ?? (signal ? 1 : 0));
      }
    });
  });
}

main();

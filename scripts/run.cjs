#!/usr/bin/env node
// Cross-platform MCP launcher for Lumen.
//
// Plugin manifests for Claude Code and Cursor accept a single `command`
// string with no per-OS dispatch. A shell-script polyglot (scripts/run.cmd)
// cannot satisfy both POSIX shebang detection (byte 0 == "#!") and cmd.exe
// stdout cleanliness (line 1 must be "@"-prefixed or a label) at the same
// byte 0, so a single-file polyglot is fundamentally incompatible with
// macOS posix_spawn. This launcher routes through Node — guaranteed to be
// in PATH for Claude Code and Cursor — and dispatches to the platform's
// shell script.

const { spawn } = require('node:child_process');
const path = require('node:path');

const root =
  process.env.CLAUDE_PLUGIN_ROOT ||
  process.env.CURSOR_PLUGIN_ROOT ||
  path.resolve(__dirname, '..');

const isWin = process.platform === 'win32';
const args = process.argv.slice(2);

const command = isWin ? 'cmd.exe' : path.join(root, 'scripts', 'run.sh');
const commandArgs = isWin
  ? ['/c', path.join(root, 'scripts', 'run.bat'), ...args]
  : args;

const child = spawn(command, commandArgs, { stdio: 'inherit' });

// Forward termination signals from MCP host to child so bin/lumen does not
// linger as an orphan process when Claude Code / Cursor shuts the launcher
// down.
['SIGTERM', 'SIGINT', 'SIGHUP'].forEach((sig) => {
  process.on(sig, () => {
    if (!child.killed) child.kill(sig);
  });
});

child.on('error', (err) => {
  console.error(`lumen launcher failed to spawn ${command}:`, err.message);
  process.exit(1);
});

child.on('exit', (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  else process.exit(code ?? 1);
});

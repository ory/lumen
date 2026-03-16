#!/usr/bin/env node
'use strict';

const { execFileSync } = require('child_process');
const path = require('path');
const os = require('os');

function getBinary() {
  const platform = os.platform(); // 'darwin', 'linux', 'win32'
  const arch = os.arch();         // 'arm64', 'x64'
  const ext = platform === 'win32' ? '.exe' : '';
  const suffix = `${platform}-${arch}`;

  try {
    const pkgJson = require.resolve(`@ory/lumen-${suffix}/package.json`);
    return path.join(path.dirname(pkgJson), 'bin', `lumen${ext}`);
  } catch {
    // platform package not installed
  }

  process.stderr.write(
    `@ory/lumen: no binary found for ${platform}/${arch}.\n` +
    `Platform package @ory/lumen-${platform}-${arch} is not installed.\n` +
    `Please report this at https://github.com/ory/lumen/issues\n`
  );
  process.exit(1);
}

const bin = getBinary();
try {
  execFileSync(bin, process.argv.slice(2), { stdio: 'inherit' });
} catch (e) {
  process.exit(e.status ?? 1);
}

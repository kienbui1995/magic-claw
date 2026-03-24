#!/usr/bin/env node
import { createWriteStream, chmodSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import https from 'https';

const __dirname = dirname(fileURLToPath(import.meta.url));
const pkg = JSON.parse(
  (await import('fs')).readFileSync(join(__dirname, '..', 'package.json'), 'utf8')
);
const VERSION = pkg.version;
const REPO = 'kienbui1995/magic';
const BINARY_PATH = join(__dirname, '..', 'bin', 'magic-bin');

function getPlatform() {
  const p = process.platform;
  const a = process.arch;
  if (p === 'darwin' && a === 'x64')  return 'darwin-amd64';
  if (p === 'darwin' && a === 'arm64') return 'darwin-arm64';
  if (p === 'linux'  && a === 'x64')  return 'linux-amd64';
  if (p === 'linux'  && a === 'arm64') return 'linux-arm64';
  throw new Error(`Unsupported platform: ${p}-${a}`);
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = createWriteStream(dest);
    const get = (u) => https.get(u, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) return get(res.headers.location);
      if (res.statusCode !== 200) return reject(new Error(`HTTP ${res.statusCode}`));
      res.pipe(file);
      file.on('finish', () => file.close(resolve));
    }).on('error', reject);
    get(url);
  });
}

const platform = getPlatform();
const url = `https://github.com/${REPO}/releases/download/v${VERSION}/magic-${platform}`;

console.log(`Downloading MagiC binary for ${platform}...`);
await download(url, BINARY_PATH);
chmodSync(BINARY_PATH, 0o755);
console.log('MagiC installed successfully.');

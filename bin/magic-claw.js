#!/usr/bin/env node
import { spawnSync } from 'child_process';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync } from 'fs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const binary = join(__dirname, 'magic-bin');

if (!existsSync(binary)) {
  console.error('MagiC binary not found. Try reinstalling: npm install -g magic-claw');
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: 'inherit' });
process.exit(result.status ?? 1);

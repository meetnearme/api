#!/usr/bin/env node

/**
 * Wrapper script to run wrangler dev with environment variables from .env
 * Reads CLOUDFLARE_API_TOKEN, CLOUDFLARE_ACCOUNT_ID, and CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID
 * from .env file and passes them to wrangler
 */

/* eslint-disable no-console */
import { writeFileSync, unlinkSync } from 'fs';
import { spawn } from 'child_process';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import process from 'node:process';
import dotenv from 'dotenv';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '..');
const env = dotenv.config({ path: join(projectRoot, '.env') });

// Create .dev.vars file for wrangler (it reads this automatically)
const devVarsPath = join(projectRoot, '.dev.vars');
const devVarsContent = Object.entries(env.parsed)
  .filter(([key]) => key.startsWith('CLOUDFLARE_'))
  .map(([key, value]) => `${key}=${value}`)
  .join('\n');

try {
  writeFileSync(devVarsPath, devVarsContent, 'utf-8');
  console.log('Created .dev.vars file from .env values');
} catch (error) {
  console.error(`Error writing .dev.vars file: ${error.message}`);
  process.exit(1);
}

// Build wrangler command
const wranglerArgs = [
  'dev',
  'worker/index.dev.js',
  '--env',
  'local',
  '--port',
  '8787',
];

console.log('Starting Cloudflare Worker dev server...');
console.log(`Reading from .env file: ${join(projectRoot, '.env')}`);
console.log(`Proxying to: http://127.0.0.1:8000`);
console.log('');

// Spawn wrangler process
const wrangler = spawn('npx', ['wrangler', ...wranglerArgs], {
  stdio: 'inherit',
  cwd: projectRoot,
  shell: true,
});

wrangler.on('error', (error) => {
  console.error(`Failed to start wrangler: ${error.message}`);
  process.exit(1);
});

// Handle process termination
const cleanup = () => {
  try {
    // Clean up .dev.vars file on exit
    unlinkSync(devVarsPath);
  } catch (error) {
    // Ignore errors if file doesn't exist
  }
};

process.on('SIGINT', () => {
  cleanup();
  wrangler.kill('SIGINT');
});
process.on('SIGTERM', () => {
  cleanup();
  wrangler.kill('SIGTERM');
});

wrangler.on('exit', (code) => {
  cleanup();
  process.exit(code || 0);
});

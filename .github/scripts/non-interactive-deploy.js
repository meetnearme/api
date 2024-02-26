#!/usr/bin/env node
import { spawn } from 'child_process';

let child;
try {
  child = spawn('npx', [
    'sst',
    'deploy',
    '--stage',
    process.env.GIT_BRANCH_NAME,
  ]);
} catch (err) {
  console.error('Error spawning sst deploy', err);
}

child.stdout.on('data', (data) => {
  if (data.toString().includes('Please enter a name')) {
    child.stdin.write(`${process.env.GIT_BRANCH_NAME}\n`);
  }
});

child.stderr.on('data', (data) => {
  // if sst deploy writes to stderr
  console.log('~stderr from non-interactive-deploy.js~', data.toString());
  if (data.toString().includes('<insert output here>')) {
    child.stdin.write('yes\n'); // or whatever it expects
  }
});

// Listen for sst deploy to close
child.on('close', (code) => {
  console.log(`sst deploy exited with code ${code}`);
  child.stdin.end();
});

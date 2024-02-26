#!/usr/bin/env node
const { spawn } = require('child_process');

const child = spawn('sst', ['deploy', '--stage', '<dynamic var from GitHub>']);

child.stdout.on('data', (data) => {
  if (data.toString().includes('Please enter a name')) {
    child.stdin.write(`${process.env.GITHUB_BRANCH_NAME}\n`); // or whatever it expects
  }
});

// child.stderr.on('data', (data) => {
//   // if sst deploy writes to stderr
//   if (data.toString().includes('<insert output here>')) {
//     child.stdin.write('yes\n'); // or whatever it expects
//   }
// });

// Listen for sst deploy to close
child.on('close', (code) => {
  console.log(`sst deploy exited with code ${code}`);
  child.exitCode(code);
});

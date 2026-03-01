#!/usr/bin/env node

const { spawn } = require('child_process');
const { join } = require('path');

// Get the path to the dcode binary
const binaryPath = join(__dirname, '..', 'dcode');

// Spawn the dcode binary with all arguments
const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  shell: false
});

child.on('exit', (code) => {
  process.exit(code);
});

child.on('error', (err) => {
  console.error('Failed to start dcode:', err.message);
  process.exit(1);
});

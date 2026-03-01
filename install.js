#!/usr/bin/env node

const { existsSync, chmodSync, copyFileSync, mkdirSync } = require('fs');
const { join } = require('path');
const { platform, arch } = require('os');

function getLocalBinaryPath() {
  // Check if there's a pre-built binary in the current directory
  const localBinary = join(__dirname, 'dcode');
  if (existsSync(localBinary)) {
    return localBinary;
  }
  return null;
}

function install() {
  const localBinary = getLocalBinaryPath();
  
  if (!localBinary) {
    console.error('Error: No pre-built dcode binary found in the package.');
    console.error('This package expects a compiled dcode binary to be included.');
    console.error('');
    console.error('To use dcode, you can:');
    console.error('1. Build from source: https://github.com/dcode-dev/dcode');
    console.error('2. Wait for platform-specific binaries to be published');
    process.exit(1);
  }

  try {
    // Ensure the binary is executable
    chmodSync(localBinary, 0o755);
    console.log('Successfully installed dcode binary');
  } catch (error) {
    console.error('Error setting executable permissions:', error.message);
    process.exit(1);
  }
}

install();

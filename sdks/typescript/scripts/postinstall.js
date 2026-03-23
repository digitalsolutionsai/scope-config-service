#!/usr/bin/env node

/**
 * Postinstall script for @digitalsolutionsai/scopeconfig
 *
 * Automatically copies proto files to the consuming project's root directory
 * after `npm install`. This ensures the proto file is available at runtime,
 * even in environments like Next.js standalone mode where files from
 * node_modules are not automatically included.
 *
 * The proto file is copied to: <project_root>/proto/config/v1/config.proto
 *
 * Set SCOPECONFIG_SKIP_POSTINSTALL=1 to skip this step.
 */

const fs = require("fs");
const path = require("path");

function main() {
  // Allow users to skip the postinstall
  if (process.env.SCOPECONFIG_SKIP_POSTINSTALL === "1") {
    return;
  }

  // INIT_CWD is set by npm/yarn to the directory where `npm install` was run
  const projectRoot = process.env.INIT_CWD;
  if (!projectRoot) {
    // Not running via npm install (e.g., during package development)
    return;
  }

  // Don't copy if we're installing within our own package (development)
  const packageDir = path.resolve(__dirname, "..");
  if (path.resolve(projectRoot) === packageDir) {
    return;
  }

  const srcProto = path.join(packageDir, "proto", "config", "v1", "config.proto");
  const destDir = path.join(projectRoot, "proto", "config", "v1");
  const destProto = path.join(destDir, "config.proto");

  // Check if source proto exists
  if (!fs.existsSync(srcProto)) {
    return;
  }

  try {
    fs.mkdirSync(destDir, { recursive: true });
    fs.copyFileSync(srcProto, destProto);
    console.log(
      "[@digitalsolutionsai/scopeconfig] Proto file copied to proto/config/v1/config.proto"
    );
  } catch (err) {
    // Don't fail the install if copy fails
    console.warn(
      "[@digitalsolutionsai/scopeconfig] Warning: Could not copy proto file:",
      err.message
    );
  }
}

main();

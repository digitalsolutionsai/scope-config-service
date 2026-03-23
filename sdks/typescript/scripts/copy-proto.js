#!/usr/bin/env node

/**
 * CLI script to copy proto files from the @digitalsolutionsai/scopeconfig package
 * to a target directory.
 *
 * Usage:
 *   npx scopeconfig-copy-proto [destination]
 *
 * Examples:
 *   npx scopeconfig-copy-proto
 *     → Copies to ./proto/config/v1/config.proto
 *
 *   npx scopeconfig-copy-proto .next/standalone
 *     → Copies to .next/standalone/proto/config/v1/config.proto
 *
 *   npx scopeconfig-copy-proto /app/proto/config/v1
 *     → Copies to /app/proto/config/v1/config.proto
 */

const fs = require("fs");
const path = require("path");

function main() {
  const args = process.argv.slice(2);

  if (args.includes("--help") || args.includes("-h")) {
    console.log("Usage: scopeconfig-copy-proto [destination]");
    console.log("");
    console.log("Copy proto files from @digitalsolutionsai/scopeconfig to a target directory.");
    console.log("");
    console.log("Arguments:");
    console.log("  destination  Target directory (default: ./proto/config/v1)");
    console.log("");
    console.log("Examples:");
    console.log("  scopeconfig-copy-proto");
    console.log("  scopeconfig-copy-proto .next/standalone");
    console.log("  scopeconfig-copy-proto /app/proto/config/v1");
    process.exit(0);
  }

  const packageDir = path.resolve(__dirname, "..");
  const srcProto = path.join(packageDir, "proto", "config", "v1", "config.proto");

  if (!fs.existsSync(srcProto)) {
    console.error("Error: Proto file not found at", srcProto);
    process.exit(1);
  }

  let destDir;
  if (args[0]) {
    const dest = path.resolve(args[0]);
    // If the destination looks like it already includes the proto path structure,
    // use it directly. Otherwise, append the proto path.
    const protoSuffix = path.join("proto", "config", "v1");
    if (path.normalize(dest).endsWith(path.normalize(protoSuffix))) {
      destDir = dest;
    } else {
      destDir = path.join(dest, "proto", "config", "v1");
    }
  } else {
    destDir = path.join(process.cwd(), "proto", "config", "v1");
  }

  const destProto = path.join(destDir, "config.proto");

  try {
    fs.mkdirSync(destDir, { recursive: true });
    fs.copyFileSync(srcProto, destProto);
    console.log("Proto file copied to", path.relative(process.cwd(), destProto));
  } catch (err) {
    console.error("Error copying proto file:", err.message);
    process.exit(1);
  }
}

main();

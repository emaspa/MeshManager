// set-version.mjs — sync a release version into the files that carry it.
// Usage: node scripts/set-version.mjs 1.2.3
//
// The git tag is the single source of truth during a CI release; this script
// stamps that version into tauri.conf.json and package.json so the bundles are
// named and reported correctly. The Go sidecar version is injected separately
// via -ldflags at build time.
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const version = process.argv[2];
if (!version || !/^\d+\.\d+\.\d+/.test(version)) {
  console.error(
    `Usage: node scripts/set-version.mjs <version>  (got: ${version ?? "<none>"})`,
  );
  process.exit(1);
}

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const targets = [
  join(root, "app", "src-tauri", "tauri.conf.json"),
  join(root, "app", "package.json"),
];

for (const file of targets) {
  const text = readFileSync(file, "utf8");
  // Replace only the first top-level "version" field so the rest of the file's
  // formatting (e.g. inline arrays) is left untouched.
  const pattern = /("version":\s*")[^"]*(")/;
  if (!pattern.test(text)) {
    console.error(`no "version" field found in ${file}`);
    process.exit(1);
  }
  writeFileSync(file, text.replace(pattern, `$1${version}$2`));
  console.log(`set version ${version} in ${file}`);
}

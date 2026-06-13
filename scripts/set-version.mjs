// set-version.mjs — sync a release version into the files that carry it.
// Usage: node scripts/set-version.mjs 1.2.3
//
// The git tag is the single source of truth during a CI release; this script
// stamps that version into the frontend manifests (tauri.conf.json,
// package.json) and the Rust crate (Cargo.toml + Cargo.lock) so the bundles,
// the installer metadata, and the compile log all agree. The Go sidecar
// version is injected separately via -ldflags at build time.
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

// Each target names the file and a regex that captures the value to replace as
// group 1 ... group 2. Patterns are anchored narrowly so dependency versions
// and other unrelated fields are never touched.
const targets = [
  // JSON manifests: first top-level "version" field.
  {
    file: join(root, "app", "src-tauri", "tauri.conf.json"),
    pattern: /("version":\s*")[^"]*(")/,
  },
  {
    file: join(root, "app", "package.json"),
    pattern: /("version":\s*")[^"]*(")/,
  },
  // Cargo.toml: the [package] version is the only line that *starts* with
  // `version = ` (dependency versions live inside inline tables).
  {
    file: join(root, "app", "src-tauri", "Cargo.toml"),
    pattern: /^(version = ")[^"]*(")/m,
  },
  // Cargo.lock: the version line directly under the meshmanager package entry.
  {
    file: join(root, "app", "src-tauri", "Cargo.lock"),
    pattern: /(name = "meshmanager"\nversion = ")[^"]*(")/,
  },
];

for (const { file, pattern } of targets) {
  const text = readFileSync(file, "utf8");
  if (!pattern.test(text)) {
    console.error(`no version field matched in ${file}`);
    process.exit(1);
  }
  writeFileSync(file, text.replace(pattern, `$1${version}$2`));
  console.log(`set version ${version} in ${file}`);
}

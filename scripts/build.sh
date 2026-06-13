#!/usr/bin/env bash
# build.sh — build the amtd sidecar for the host triple, then bundle the Tauri
# desktop app (deb / rpm / AppImage under app/src-tauri/target/release/bundle).
#
# Linux counterpart of scripts/build.ps1. Requires Go, Node/npm, the Rust
# toolchain, and the Tauri system libraries (webkit2gtk-4.1, gtk+-3.0,
# libsoup-3.0). The Tauri CLI is taken from app/node_modules (npm run tauri).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Resolve the Rust host triple so the sidecar is named as Tauri expects
# (externalBin "binaries/amtd" → binaries/amtd-<triple>).
triple="$(rustc -vV | sed -n 's/^host: //p')"
out="$root/app/src-tauri/binaries/amtd-$triple"

echo "Building amtd sidecar → $out"
mkdir -p "$root/app/src-tauri/binaries"
( cd "$root/amtd" && go build -ldflags "-s -w -X main.Version=0.1.0" -o "$out" . )

echo "Installing frontend dependencies…"
( cd "$root/app" && npm install )

echo "Bundling Tauri app…"
( cd "$root/app" && npm run tauri build )

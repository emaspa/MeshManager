#!/usr/bin/env bash
# dev.sh — run the amtd sidecar and the Vite frontend together for browser dev.
#
# Linux counterpart of scripts/dev.ps1. Generates a shared random token, starts
# amtd on a fixed port, points the frontend at it, and serves the UI at
# http://localhost:1420. Ctrl-C stops both.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

token="$(LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 32)"
port=7777

echo "Building amtd…"
( cd "$root/amtd" && go build -o amtd . )

echo "Starting amtd on 127.0.0.1:$port"
"$root/amtd/amtd" -addr "127.0.0.1:$port" -token "$token" &
amtd_pid=$!
trap 'kill "$amtd_pid" 2>/dev/null || true' EXIT

export VITE_AMTD_URL="http://127.0.0.1:$port"
export VITE_AMTD_TOKEN="$token"

echo "Starting frontend on http://localhost:1420"
( cd "$root/app" && npm run dev )

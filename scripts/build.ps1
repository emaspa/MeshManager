# build.ps1 — build the amtd sidecar for the host triple, then bundle the
# Tauri desktop app (installer + standalone exe under app/src-tauri/target).

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
            [System.Environment]::GetEnvironmentVariable("Path", "User")

# Resolve the Rust host triple so the sidecar is named as Tauri expects.
$triple = (rustc -vV | Select-String "host:").ToString().Split(" ")[1].Trim()
$out = "$root\app\src-tauri\binaries\amtd-$triple.exe"

Write-Host "Building amtd sidecar → $out" -ForegroundColor Cyan
New-Item -ItemType Directory -Force "$root\app\src-tauri\binaries" | Out-Null
Push-Location "$root\amtd"
go build -ldflags "-s -w -X main.Version=0.1.1" -o $out .
Pop-Location

Write-Host "Bundling Tauri app…" -ForegroundColor Cyan
Push-Location "$root\app"
cargo tauri build
Pop-Location

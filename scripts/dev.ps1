# dev.ps1 — run the amtd sidecar and the Vite frontend together for browser dev.
#
# Generates a shared random token, starts amtd on a fixed port, points the
# frontend at it, and opens http://localhost:1420. Ctrl-C stops both.

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
            [System.Environment]::GetEnvironmentVariable("Path", "User")

$token = -join ((48..57) + (97..122) | Get-Random -Count 32 | ForEach-Object { [char]$_ })
$port = 7777

Write-Host "Building amtd…" -ForegroundColor Cyan
Push-Location "$root\amtd"
go build -o amtd.exe .
Pop-Location

Write-Host "Starting amtd on 127.0.0.1:$port" -ForegroundColor Cyan
$amtd = Start-Process -FilePath "$root\amtd\amtd.exe" `
  -ArgumentList "-addr", "127.0.0.1:$port", "-token", $token `
  -PassThru -NoNewWindow

$env:VITE_AMTD_URL = "http://127.0.0.1:$port"
$env:VITE_AMTD_TOKEN = $token

try {
  Write-Host "Starting frontend on http://localhost:1420" -ForegroundColor Cyan
  Push-Location "$root\app"
  bun run dev
} finally {
  Pop-Location
  if ($amtd -and -not $amtd.HasExited) { Stop-Process -Id $amtd.Id -Force }
}

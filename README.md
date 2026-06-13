# MeshManager

[![Release](https://img.shields.io/github/v/release/emaspa/meshmanager)](https://github.com/emaspa/meshmanager/releases/latest)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)

A modern Intel(R) AMT / vPro management console. It is a ground-up rebuild of
the classic MeshCommander, built with a Go protocol engine, a typed React UI,
and a Tauri desktop shell.

MeshManager talks to Intel AMT / vPro machines out of band (independently of the
host OS) to control power and boot, read hardware and logs, manage accounts,
certificates and networking, and drive the redirection features
(Serial-over-LAN, KVM remote desktop, and boot-from-ISO).

License: Apache 2.0. Platform: the desktop build runs on Windows, macOS, and
Linux (Linux bundles as deb / rpm / AppImage; macOS as .app / .dmg); the Go
sidecar and React UI are cross-platform.

## Features

Connectivity
- Connect by host with Digest auth over HTTP (16992) or TLS (16993), with an
  "allow self-signed" option for the usual AMT certificate.
- Discover devices with a subnet / CIDR scan that probes the AMT port and
  flags hosts by their HTTP Server header.
- Save connections as bookmarks (organized into groups, with an optional
  remembered password and last-connected time). Bookmarks persist across
  disconnects and restarts.

Power and boot
- Power actions: on, off, graceful off, reset, power cycle, sleep, hibernate, NMI.
- One-time boot to PXE, CD/DVD, hard disk, or BIOS setup, followed by a reset.

System and inventory
- System Status: Intel ME version and control mode, System ID, provisioning
  state, user-consent policy, device clock, and the active features
  (Redirection Port, Serial-over-LAN, IDE-Redirect, KVM).
- Hardware: system manufacturer/model/serial, BIOS vendor/version, processors
  (brand, manufacturer, clock, stepping, status), memory (type, form factor,
  size, manufacturer, part and serial), and storage (model, serial, and
  capacity).
- Decoded firmware event log and audit log.

Network
- Wired interfaces (IP, DHCP, MAC, DNS, link state) with edit (switch to DHCP or
  set a dedicated static IP), plus a general summary (domain, respond-to-ping,
  dynamic DNS).
- Wireless: list / add (WPA2-PSK) / remove Wi-Fi profiles.
- Remote Access (CIRA): manage MPS servers and policy rules, and view
  environment detection.

Accounts and security
- User accounts: list, add a digest user (with realms and access level),
  enable / disable, and remove.
- Certificates: view stored certificates, add a trusted root, and delete.

Redirection
- Serial-over-LAN terminal (xterm.js over the AMT redirection channel).
- KVM remote desktop: selectable color depth (16-bit, 8-bit, grayscale) and
  compression, mouse + keyboard forwarding, a special-keys menu
  (Ctrl+Alt+Del, Alt+Tab, Alt+F4, Win, function keys, ...), paste-as-keystrokes,
  view-only mode, fit / actual scaling, screenshot, fullscreen, video
  recording to WebM, and a data-activity LED.
- IDE-R: boot a machine from a local ISO, served as a virtual CD-ROM by the
  sidecar (ATAPI emulation).

Tooling
- WS-MAN browser: read-only inspection of any supported AMT / CIM / IPS class.
- Scheduled wake (Alarm Clock): add / list / remove wake-ups.
- Logging designed for bug reports (see [Logs and bug reports](#logs-and-bug-reports)).

## Architecture

```
+--------------------------  MeshManager (Tauri desktop app)  --------------------------+
|                                                                                       |
|   React + TypeScript UI  <--- HTTP / WebSocket --->  amtd (Go sidecar, 127.0.0.1)     |
|   (Vite, Tailwind, react-query)                       |                               |
|                                                       +-- WS-MAN over HTTP Digest/TLS |
|   Rust core: spawns amtd, generates a bearer          |   (AMT ports 16992 / 16993)   |
|   token, hands the endpoint to the UI                 +-- Binary redirection          |
|                                                           (16994 / 16995):            |
|                                                           SOL . KVM . IDE-R           |
+---------------------------------------------------------------------------------------+
```

- `amtd/` is the Go daemon. All AMT protocol work lives here so the UI never
  needs raw TCP. It wraps Intel's
  [go-wsman-messages](https://github.com/device-management-toolkit/go-wsman-messages)
  for the WS-MAN layer, implements the binary redirection protocols (SOL, KVM,
  IDE-R) itself, and exposes a small local HTTP + WebSocket API on `127.0.0.1`
  protected by a bearer token.
- `app/` holds the React / TypeScript frontend (Vite + Tailwind) and the Tauri
  shell in `app/src-tauri/`.
- `scripts/` has helper scripts for development and release builds.

The desktop shell is deliberately thin: it spawns `amtd` on a random loopback
port with a freshly generated bearer token, learns the port from the sidecar's
first stdout line, and hands the endpoint plus token to the UI through a
`sidecar_info` command. The UI auto-detects Tauri and uses it; in a plain
browser it falls back to a configurable endpoint.

## Requirements

- Go 1.26 or newer (sidecar).
- Node.js with npm (frontend tooling). The Tauri CLI is installed as a project
  dev dependency, so `npm install` in `app/` provides it (no global install
  needed). Bun also works if you prefer it; the scripts use npm.
- For the Windows desktop build: the MSVC Rust toolchain
  (`rustup default stable-x86_64-pc-windows-msvc`), Visual Studio Build Tools
  with the C++ workload, and the WebView2 runtime (preinstalled on Windows 11).
- For the Linux desktop build: the stable Rust toolchain plus the Tauri system
  libraries (`webkit2gtk-4.1`, `gtk+-3.0`, `libsoup-3.0`) and the usual C
  build tools. On Debian/Ubuntu:
  `sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev libsoup-3.0-dev build-essential pkg-config`.
- For the macOS desktop build (Apple Silicon or Intel): the Xcode Command Line
  Tools (`xcode-select --install`) and the stable Rust toolchain. Go, Node, and
  Rust can all be installed via Homebrew (`brew install go node rust`). The
  WebView (WKWebView) ships with the OS. Tauri targets the host architecture.

## Build and run

### Standalone desktop app

Windows:

```powershell
pwsh scripts/build.ps1
```

Linux and macOS:

```bash
bash scripts/build.sh
```

`build.sh` resolves the host Rust triple, so the same script bundles for Linux
and macOS. Each builds the sidecar, compiles the frontend, and bundles the app
under `app/src-tauri/target/release/`.

Windows produces installers plus a portable executable:

- `bundle/nsis/MeshManager_<version>_x64-setup.exe` (installer)
- `bundle/msi/MeshManager_<version>_x64_en-US.msi` (MSI)
- `meshmanager.exe` plus the `amtd-*.exe` sidecar next to it (portable, runs
  without installing; keep the two files together). GitHub releases ship this
  pair as `MeshManager_<version>_x64-portable.zip`.

Linux produces:

- `bundle/deb/MeshManager_<version>_amd64.deb`
- `bundle/rpm/MeshManager-<version>-1.x86_64.rpm`
- `bundle/appimage/MeshManager_<version>_amd64.AppImage` (portable)

macOS produces (named for the host architecture):

- `bundle/macos/MeshManager.app` (app bundle)
- `bundle/dmg/MeshManager_<version>_<arch>.dmg` (e.g. `_aarch64` on Apple Silicon)

Builds are currently unsigned, so Windows SmartScreen warns on first run
("More info" then "Run anyway"). On macOS, Gatekeeper blocks the unsigned app on
first launch: right-click it and choose Open, or clear the quarantine flag with
`xattr -dr com.apple.quarantine MeshManager.app`.

### Develop the desktop app

```bash
cd app
npm install        # one time
npm run tauri dev  # starts Vite and the desktop window together
```

`tauri dev` spawns the bundled `amtd` sidecar from `app/src-tauri/binaries/`,
so build it at least once first (`bash scripts/build.sh`, or on Windows
`pwsh scripts/build.ps1`) to produce the `amtd-<triple>` binary it expects.

Do not run the debug binary at `app/src-tauri/target/debug/meshmanager` (or
`meshmanager.exe` on Windows) on its own: a debug Tauri build loads the dev
server at `http://localhost:1420`, so without Vite running the window shows
"can't reach this page". Use `npm run tauri dev`, or build a release bundle for
a standalone app that embeds the frontend.

### Develop in a browser (no Tauri)

Windows:

```powershell
cd app; bun install; cd ..
pwsh scripts/dev.ps1
```

Linux and macOS:

```bash
cd app && npm install && cd ..
bash scripts/dev.sh
```

Starts the sidecar and the Vite dev server with a shared token, then serves the
UI at <http://localhost:1420>.

## Using it

In the sidebar, click `+` to add a device or the radar icon to scan a subnet.
Enter the host, AMT admin username and password, choose TLS if the device uses
it (port defaults to 16992, or 16993 with TLS), and tick "Allow self-signed"
for the typical self-signed AMT certificate. Optionally set a Group and
"Remember password". Connecting saves the device as a bookmark.

## HTTP and WebSocket API

The sidecar serves a local API under `/api`. Every route except `/api/health`
requires the bearer token (the desktop shell injects it; WebSocket routes accept
it via the `access_token` query parameter).

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Liveness and version |
| POST | `/log` | Ingest a client-side log record |
| POST | `/connect` | Open a session `{host, port, username, password, tls, insecure}` |
| POST | `/discover` | Subnet scan `{cidr, port, tls}` |
| GET | `/devices` | List active sessions |
| POST | `/devices/{id}/disconnect` | Drop a session |
| GET | `/devices/{id}/info` | Identity, ME version/mode, active features, consent, clock |
| GET, POST | `/devices/{id}/power` | Read power state, or request a change `{action}` |
| POST | `/devices/{id}/boot` | One-time boot `{device, power}` (pxe / cd / hdd / bios) |
| GET | `/devices/{id}/hardware` | System, BIOS, CPU, memory, storage |
| GET, POST | `/devices/{id}/network` | Read wired interfaces, or set DHCP / static IP |
| GET, POST | `/devices/{id}/wifi` | List or add a Wi-Fi profile |
| DELETE | `/devices/{id}/wifi/{instanceId}` | Remove a Wi-Fi profile |
| GET | `/devices/{id}/remoteaccess` | MPS servers, policies, environment detection |
| POST, DELETE | `/devices/{id}/remoteaccess/mps[/{name}]` | Add / remove an MPS server |
| POST, DELETE | `/devices/{id}/remoteaccess/policies[/{name}]` | Add / remove a CIRA policy |
| GET, POST | `/devices/{id}/accounts` | List users, or add a digest user |
| POST, DELETE | `/devices/{id}/accounts/{handle}` | Enable/disable, or remove a user |
| GET, POST | `/devices/{id}/certificates` | List certificates, or add a trusted root |
| DELETE | `/devices/{id}/certificates/{instanceId}` | Delete a certificate |
| GET, POST | `/devices/{id}/alarms` | List or add a scheduled wake |
| DELETE | `/devices/{id}/alarms/{instanceId}` | Remove a scheduled wake |
| GET | `/devices/{id}/browse/classes` | List browsable WS-MAN classes |
| GET | `/devices/{id}/browse?class=...` | Enumerate a WS-MAN class |
| GET | `/devices/{id}/eventlog`, `/auditlog` | Decoded event and audit logs |
| POST | `/devices/{id}/ider/start`, `/stop` | Mount / eject a remote ISO `{isoPath, boot}` |
| GET | `/devices/{id}/ider/status` | IDE-R transfer stats |
| WS | `/devices/{id}/sol`, `/kvm` | Serial-over-LAN and KVM redirection |

Power actions: `on`, `off`, `off-graceful`, `reset`, `reset-graceful`, `cycle`,
`sleep`, `hibernate`, `nmi`.

Run the sidecar by itself:

```bash
cd amtd
go build -o amtd .      # amtd.exe on Windows
./amtd -addr 127.0.0.1:7777 -token devtoken -log-dir ./logs -debug
```

## Logs and bug reports

Everything logs to one place so a tester can attach a single folder to a report.
Logs are appended across sessions and rotated, not cleared on each run.

- The sidecar writes rotating `amtd.log` files (5 MB each, gzipped, up to 20
  kept) holding the startup environment, one line per HTTP request (failures at
  warn / error), connect results, and AMT operation errors. Rotated files are
  retained for a configurable number of days (default 30).
- The Tauri shell appends the sidecar's stdout and stderr to `shell.log` as a
  safety net for crashes that happen before the sidecar can write its own log.
- The frontend forwards `window.onerror`, unhandled promise rejections, and API
  failures to the sidecar, so UI errors appear in `amtd.log` tagged `src=ui`
  with a stack trace.

Retention is adjustable: the Add Device dialog has a "Keep logs for (days)"
field. Because the sidecar's file logger starts with the app, a change takes
effect on the next launch.

In the packaged app the log folder is the per-user app log directory: on
Windows `%LOCALAPPDATA%\com.emaspa.meshmanager\logs\`, on macOS
`~/Library/Logs/com.emaspa.meshmanager/`, on Linux
`~/.local/share/com.emaspa.meshmanager/logs/` (or `$XDG_DATA_HOME`). Click the
Logs button in the sidebar footer to open it. When running the sidecar standalone, pass
`-log-dir <path>` (omit it to log to stderr only) and optionally
`-log-max-age <days>` / `-log-max-backups <n>`.

## Project layout

```
amtd/                Go sidecar (AMT protocol engine + local API)
  internal/amt/        sessions, power, boot, inventory, network, accounts,
                       certificates, alarms, remote access, discovery, browse
  internal/redirect/   SOL, KVM, and IDE-R binary redirection
  internal/api/        HTTP + WebSocket server, request logging
app/                 frontend + desktop shell
  src/                 React + TypeScript UI (tabs per feature)
  src-tauri/           Rust shell (spawns the sidecar, logging, commands)
scripts/             dev.ps1, build.ps1, icon generator
```

## Notes and limitations

- Hardware validation is partial. The WS-MAN operations use Intel's library and
  several features (KVM, SOL, hardware, accounts, networking) have been
  exercised against real vPro hardware; others are faithful ports that would
  benefit from more device testing. The logs are the first place to look if something
  behaves unexpectedly.
- A few panels are not implemented because the current go-wsman-messages
  release exposes no write API for them (System Defense, power policies, event
  subscriptions). Their state can still be read through the WS-MAN browser.
- Installers are unsigned. Code signing requires a code-signing certificate.

## Support

If you find MeshManager useful, consider [buying me a coffee](https://buymeacoffee.com/emaspa).

## License

MeshManager is licensed under the Apache License 2.0; see [LICENSE](LICENSE).

Parts of the redirection layer (Serial-over-LAN, KVM, IDE-R) are derived from
[MeshCommander](https://github.com/Ylianst/MeshCommander) by Ylian
Saint-Hilaire (Apache 2.0), ported and modified; see [NOTICE](NOTICE). Bundled
open-source dependencies are listed in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

This project is not affiliated with or endorsed by Intel or the MeshCommander
author. Intel, AMT, and vPro are trademarks of Intel Corporation.

# MeshManager

A modern Intel(R) AMT / vPro management console. It is a ground-up rebuild of
the classic MeshCommander, built with a Go protocol engine, a typed React UI,
and a Tauri desktop shell.

MeshManager talks to Intel AMT / vPro machines out of band (independently of the
host OS) to control power and boot, read hardware and logs, manage accounts,
certificates and networking, and drive the redirection features
(Serial-over-LAN, KVM remote desktop, and boot-from-ISO).

License: Apache 2.0. Platform: the desktop build targets Windows; the Go
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
- Bun (frontend tooling).
- For the desktop build: the MSVC Rust toolchain
  (`rustup default stable-x86_64-pc-windows-msvc`), Visual Studio Build Tools
  with the C++ workload, the WebView2 runtime (preinstalled on Windows 11), and
  the Tauri CLI (`cargo install tauri-cli --version "^2"`).

## Build and run

### Standalone desktop app

```powershell
pwsh scripts/build.ps1
```

Builds the sidecar, compiles the frontend, and bundles installers plus a
portable executable under `app/src-tauri/target/release/`:

- `bundle/nsis/MeshManager_<version>_x64-setup.exe` (installer)
- `bundle/msi/MeshManager_<version>_x64_en-US.msi` (MSI)
- `meshmanager.exe` (portable, runs without installing)

Builds are currently unsigned, so Windows SmartScreen warns on first run
("More info" then "Run anyway").

### Develop the desktop app

```powershell
cd app
bun install        # one time
cargo tauri dev    # starts Vite and the desktop window together
```

Do not double-click the debug binary at
`app/src-tauri/target/debug/meshmanager.exe` on its own: a debug Tauri build
loads the dev server at `http://localhost:1420`, so without Vite running the
window shows "can't reach this page". Use `cargo tauri dev`, or build a release
bundle for a standalone app that embeds the frontend.

### Develop in a browser (no Tauri)

```powershell
cd app; bun install; cd ..
pwsh scripts/dev.ps1
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

```powershell
cd amtd
go build -o amtd.exe .
./amtd.exe -addr 127.0.0.1:7777 -token devtoken -log-dir ./logs -debug
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

In the packaged app the log folder is at
`%LOCALAPPDATA%\com.emaspa.meshmanager\logs\`. Click the Logs button in the
sidebar footer to open it. When running the sidecar standalone, pass
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

## License

MeshManager is licensed under the Apache License 2.0; see [LICENSE](LICENSE).

Parts of the redirection layer (Serial-over-LAN, KVM, IDE-R) are derived from
[MeshCommander](https://github.com/Ylianst/MeshCommander) by Ylian
Saint-Hilaire (Apache 2.0), ported and modified; see [NOTICE](NOTICE). Bundled
open-source dependencies are listed in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

This project is not affiliated with or endorsed by Intel or the MeshCommander
author. Intel, AMT, and vPro are trademarks of Intel Corporation.

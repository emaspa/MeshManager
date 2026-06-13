# MeshManager

A modern Intel(R) AMT / vPro management console. It is a ground-up rebuild of
the classic MeshCommander, built with a Go protocol engine, a typed React UI,
and a Tauri desktop shell.

MeshManager talks to Intel AMT / vPro machines out of band (independently of the
host OS) to control power, boot, hardware, the firmware logs, and the
redirection features (Serial-over-LAN, KVM remote desktop, and boot-from-ISO).

License: Apache 2.0. Platform: Windows (the desktop build targets Windows;
the Go sidecar and React UI are cross-platform).

## Features

- Connect by host or discover devices with a subnet / CIDR scan, and save them
  as bookmarks (optionally with a remembered password).
- Power control: on, off, graceful off, reset, cycle, sleep, hibernate, NMI.
- One-time boot to PXE, CD/DVD, hard disk, or BIOS setup, with a reset.
- Hardware inventory: CPU, memory, storage, and chassis.
- Network info: AMT ethernet interfaces (IP, DHCP, MAC, DNS, link state).
- User accounts: list, add a digest user (with realms and access level),
  enable / disable, and remove.
- Firmware event log and audit log, decoded.
- Serial-over-LAN terminal (xterm.js over the AMT redirection channel).
- KVM remote desktop with selectable color depth (16-bit, 8-bit, grayscale) and
  compression, with mouse and keyboard forwarding and Ctrl+Alt+Del.
- IDE-R: boot a machine from a local ISO, served as a virtual CD-ROM by the
  sidecar (ATAPI emulation).
- WS-MAN browser: read-only inspection of any supported AMT / CIM / IPS class
  (certificates, Wi-Fi, remote-access policy, opt-in, TLS, and more).
- Logging built for bug reports (see [Logs and bug reports](#logs-and-bug-reports)).

## Architecture

```
+--------------------------  MeshManager (Tauri desktop app)  --------------------------+
|                                                                                       |
|   React + TypeScript UI  <--- HTTP / WebSocket --->  amtd (Go sidecar, 127.0.0.1)     |
|   (Vite, Tailwind, react-query)                       |                               |
|                                                       +-- WS-MAN over HTTP Digest/TLS |
|   Rust core: spawns amtd, generates a bearer          |   (AMT ports 16992 / 16993)   |
|   token, hands the endpoint to the UI                 +-- Binary redirection          |
|                                                           (16994 / 16995):             |
|                                                           SOL . KVM . IDE-R           |
+---------------------------------------------------------------------------------------+
```

- `amtd/` is the Go daemon. All AMT protocol work lives here so the UI never
  needs raw TCP. It wraps Intel's
  [go-wsman-messages](https://github.com/device-management-toolkit/go-wsman-messages)
  for the WS-MAN message layer, implements the binary redirection protocols
  itself, and exposes a small local HTTP + WebSocket API on `127.0.0.1`
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

## Quick start

### Build a standalone desktop app

```powershell
pwsh scripts/build.ps1
```

This builds the sidecar, compiles the frontend, and bundles an installer plus a
portable executable under `app/src-tauri/target/release/`:

- `bundle/nsis/MeshManager_<version>_x64-setup.exe` (installer)
- `bundle/msi/MeshManager_<version>_x64_en-US.msi` (MSI)
- `meshmanager.exe` (portable, runs without installing)

The builds are currently unsigned, so Windows SmartScreen will warn on first
run; choose "More info" then "Run anyway".

### Develop the desktop app

```powershell
cd app
bun install        # one time
cargo tauri dev    # starts Vite and the desktop window together
```

Do not double-click the debug binary at
`app/src-tauri/target/debug/meshmanager.exe` on its own. A debug Tauri build
loads the dev server at `http://localhost:1420`, so without Vite running the
window shows "can't reach this page". Use `cargo tauri dev` for development, or
build a release bundle for a standalone app that embeds the frontend.

### Develop in a browser (no Tauri)

```powershell
cd app; bun install; cd ..
pwsh scripts/dev.ps1
```

This starts the sidecar and the Vite dev server with a shared token, then serves
the UI at <http://localhost:1420>.

## Connecting to a device

In the sidebar, click `+` to add a device or the radar icon to scan a subnet.
Enter the host, AMT admin username and password, and choose TLS if the device
uses it. The port defaults to 16992 (or 16993 with TLS). Tick "Allow
self-signed" for the typical self-signed AMT certificate, and "Remember
password" for one-click reconnects. Connecting saves the device as a bookmark
that persists across disconnects and restarts.

## HTTP and WebSocket API

The sidecar serves a local API under `/api`. Every route except `/api/health`
requires the bearer token (the desktop shell injects it automatically; WebSocket
routes accept it via the `access_token` query parameter).

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Liveness and version |
| POST | `/connect` | Open a session `{host, port, username, password, tls, insecure}` |
| POST | `/discover` | Subnet scan `{cidr, port, tls}` |
| GET | `/devices` | List active sessions |
| POST | `/devices/{id}/disconnect` | Drop a session |
| GET | `/devices/{id}/info` | Identity, general settings, firmware versions |
| GET, POST | `/devices/{id}/power` | Read power state, or request a change `{action}` |
| POST | `/devices/{id}/boot` | One-time boot `{device, power}` (pxe / cd / hdd / bios) |
| GET | `/devices/{id}/hardware` | CPU, memory, disk, chassis inventory |
| GET | `/devices/{id}/network` | AMT ethernet interfaces |
| GET, POST | `/devices/{id}/accounts` | List users, or add a digest user |
| POST, DELETE | `/devices/{id}/accounts/{handle}` | Enable/disable, or remove a user |
| GET | `/devices/{id}/eventlog`, `/auditlog` | Decoded firmware event and audit logs |
| GET | `/devices/{id}/browse/classes` | List browsable WS-MAN classes |
| GET | `/devices/{id}/browse?class=...` | Enumerate a WS-MAN class |
| POST | `/devices/{id}/ider/start`, `/stop` | Mount / eject a remote ISO `{isoPath, boot}` |
| GET | `/devices/{id}/ider/status` | IDE-R transfer stats |
| WS | `/devices/{id}/sol`, `/kvm` | Serial-over-LAN and KVM redirection |

Power actions: `on`, `off`, `off-graceful`, `reset`, `reset-graceful`, `cycle`,
`sleep`, `hibernate`, `nmi`.

You can run the sidecar by itself:

```powershell
cd amtd
go build -o amtd.exe .
./amtd.exe -addr 127.0.0.1:7777 -token devtoken -log-dir ./logs -debug
```

## Logs and bug reports

Everything logs to one place so a tester can attach a single folder to a report.

- The sidecar writes rotating `amtd.log` files (5 MB each, 5 kept, gzipped):
  startup environment, one line per HTTP request (failures at warn or error),
  connect results, and AMT operation errors.
- The Tauri shell tees the sidecar's stdout and stderr to `shell.log` as a
  safety net, which captures crashes that happen before the sidecar can write
  its own log.
- The frontend forwards `window.onerror`, unhandled promise rejections, and API
  failures to the sidecar, so UI errors appear in `amtd.log` tagged `src=ui`
  with a stack trace.

In the packaged app the log folder is at
`%LOCALAPPDATA%\com.emaspa.meshmanager\logs\`. Click the Logs button in the
sidebar footer to open it. When running the sidecar standalone, pass
`-log-dir <path>` (omit it to log to stderr only).

## Project layout

```
amtd/                Go sidecar (AMT protocol engine + local API)
  internal/amt/        sessions, power, boot, inventory, logs, accounts, discovery
  internal/redirect/   SOL, KVM, and IDE-R binary redirection
  internal/api/        HTTP + WebSocket server
app/                 frontend + desktop shell
  src/                 React + TypeScript UI
  src-tauri/           Rust shell (spawns the sidecar)
scripts/             dev.ps1, build.ps1, icon generator
```

## Status and hardware validation

Power, boot, inventory, network, accounts, and the logs use Intel's well-tested
WS-MAN library. Serial-over-LAN and KVM have been confirmed against real vPro
hardware. IDE-R and the digest user-add path are faithful ports of the
MeshCommander logic with unit tests on the wire encodings, but have had less
real-hardware validation; the logs are the first place to look if something
behaves unexpectedly.

## Roadmap

- [x] WS-MAN transport (Digest / TLS) and session management
- [x] Power control, one-time boot, hardware inventory, event and audit logs
- [x] React UI: bookmarks, dashboard, power and boot menus, inventory and logs
- [x] Serial-over-LAN terminal
- [x] KVM remote desktop with color-depth and compression controls
- [x] IDE-R (boot from a remote ISO)
- [x] Network info, account management, device discovery
- [x] WS-MAN browser
- [x] Tauri desktop shell, installer, and logging
- [x] Alarm clock (scheduled wake)
- [x] Certificate view + management (add trusted root, delete)
- [x] Wireless (WiFi) profile management
- [x] Remote Access (CIRA): MPS servers and policies
- [ ] Wired network editing (static IP / DHCP). Supported by the library but
      carries device lock-out risk, so it is gated pending hardware validation.
- [ ] System Defense, power policies, event subscriptions. The current
      go-wsman-messages release exposes no write API for these (no methods on
      `hdr8021filter` / `systempowerscheme`, and no event-manager package), so
      they need raw WS-MAN or an upstream addition.
- [ ] Code signing for the installer (requires a code-signing certificate)

## License

MeshManager is licensed under the Apache License 2.0; see [LICENSE](LICENSE).

Parts of the redirection layer (Serial-over-LAN, KVM, IDE-R) are derived from
[MeshCommander](https://github.com/Ylianst/MeshCommander) by Ylian
Saint-Hilaire (Apache 2.0), ported and modified; see [NOTICE](NOTICE). Bundled
open-source dependencies are listed in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

This project is not affiliated with or endorsed by Intel or the MeshCommander
author. Intel, AMT, and vPro are trademarks of Intel Corporation.

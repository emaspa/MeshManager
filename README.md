# MeshManager

A modern Intel® AMT / vPro management console — a ground-up rebuild of the
classic *MeshCommander* (mesh-mini), with a Go protocol engine, a typed React
UI, and a Tauri desktop shell.

> Status: **early but functional.** Core out-of-band management works today
> (connect, power control, hardware inventory, event & audit logs). Redirection
> (Serial-over-LAN, KVM remote desktop, IDE-R) and the Tauri shell are in
> progress — see [Roadmap](#roadmap).

## Architecture

```
┌──────────────────────── Tauri desktop app ────────────────────────┐
│                                                                    │
│   React + TypeScript frontend  ──HTTP/WS──▶  amtd (Go sidecar)     │
│   (Vite, Tailwind, react-query)             │                      │
│                                             ├─ WS-MAN over Digest/  │
│   Rust core: spawns amtd, hands the         │  TLS (ports 16992/3) │
│   frontend its port + auth token            └─ redirection (16994/5)│
│                                                SOL · KVM · IDE-R    │
└────────────────────────────────────────────────────────────────────┘
```

- **`amtd/`** — Go daemon. All AMT protocol work lives here so the UI never
  needs raw TCP. Wraps Intel's
  [`go-wsman-messages`](https://github.com/device-management-toolkit/go-wsman-messages)
  for the WS-MAN message layer; exposes a small local HTTP + WebSocket API on
  `127.0.0.1`, protected by a bearer token.
- **`app/`** — React/TypeScript frontend (Vite + Tailwind) and the Tauri shell
  (`app/src-tauri/`, added once the MSVC toolchain is set up).
- **`scripts/dev.ps1`** — runs the sidecar + frontend together for browser dev.

## Running in development

Requires **Go 1.26+** and **Bun**. From the repo root:

```powershell
# one-time
cd app; bun install; cd ..

# run sidecar + frontend (generates a shared token, opens :1420)
pwsh scripts/dev.ps1
```

Then open <http://localhost:1420>, click **+** in the sidebar, and connect to an
AMT device (host, AMT admin credentials; port defaults to 16992, or 16993 with
TLS).

### Running as the Tauri desktop app

Requires the **MSVC** Rust toolchain (`rustup default stable-x86_64-pc-windows-msvc`)
and VS Build Tools, plus the Tauri CLI (`cargo install tauri-cli --version "^2"`).

```powershell
# dev: hot-reloading desktop window (spawns the sidecar automatically)
cd app
cargo tauri build --debug   # or: cargo tauri dev

# release: build the sidecar + bundle an installer
pwsh scripts/build.ps1
```

The Rust shell (`app/src-tauri/`) spawns `amtd` on a random loopback port with a
generated bearer token and hands the endpoint to the frontend via the
`sidecar_info` command — the UI auto-detects Tauri and uses it.

> **Don't double-click `app/src-tauri/target/debug/meshmanager.exe`.** A *debug*
> Tauri build loads the dev server at `http://localhost:1420`, so without Vite
> running the window shows "can't reach this page." Use `cargo tauri dev` (which
> starts Vite for you) during development, or run `cargo tauri build` /
> `scripts/build.ps1` to produce a standalone app that embeds the frontend. The
> bundled installer/exe lands under `app/src-tauri/target/release/bundle/`.

### Running the sidecar alone

```powershell
cd amtd
go build -o amtd.exe .
./amtd.exe -addr 127.0.0.1:7777 -token devtoken -debug
```

API surface (all under `/api`, bearer-token auth except `/health`):

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | liveness + version |
| POST | `/connect` | open a session `{host, port, username, password, tls, insecure}` |
| GET | `/devices` | list sessions |
| POST | `/devices/{id}/disconnect` | drop a session |
| GET | `/devices/{id}/info` | identity, general settings, firmware versions |
| GET / POST | `/devices/{id}/power` | read power state / request change `{action}` |
| POST | `/devices/{id}/boot` | one-time boot `{device, power}` (pxe/cd/hdd/bios) |
| GET | `/devices/{id}/hardware` | CPU, memory, disk, chassis inventory |
| GET | `/devices/{id}/network` | AMT ethernet interfaces (IP/DHCP/MAC/DNS) |
| GET / POST | `/devices/{id}/accounts` | list users / add digest user |
| POST / DELETE | `/devices/{id}/accounts/{handle}` | enable-disable / remove user |
| GET | `/devices/{id}/eventlog` · `/auditlog` | decoded firmware event / audit logs |
| POST | `/devices/{id}/ider/start` · `/stop` | mount/eject a remote ISO `{isoPath, boot}` |
| GET | `/devices/{id}/ider/status` | IDE-R transfer stats |
| POST | `/discover` | subnet scan `{cidr, port, tls}` |
| WS | `/devices/{id}/sol` · `/kvm` | Serial-over-LAN · KVM redirection |

Power actions: `on`, `off`, `off-graceful`, `reset`, `reset-graceful`,
`cycle`, `sleep`, `hibernate`, `nmi`.

## Logs & bug reports

Everything logs to one place so a tester can attach a single folder to a report:

- The sidecar writes rotating `amtd.log` files (5 MB × 5, gzipped) — startup
  environment, one line per HTTP request (failures at warn/error), connect
  results, and AMT operation errors.
- The Tauri shell tees the sidecar's stdout/stderr to `shell.log` as a safety
  net (captures crashes before the sidecar can write its own log).
- The frontend funnels `window.onerror`, unhandled promise rejections, and API
  failures to the sidecar via `POST /api/log`, so UI errors appear in `amtd.log`
  tagged `src=ui` with a stack.

In the packaged app the log folder is under
`%LOCALAPPDATA%\com.emaspa.meshmanager\logs\`. Click **Logs** in the sidebar
footer to open it. Running the sidecar standalone, pass `-log-dir <path>` (omit
it to log to stderr only).

## Roadmap

- [x] WS-MAN transport (Digest/TLS) + session management
- [x] Power control, hardware inventory, event/audit logs
- [x] React UI: device list, dashboard, power menu, inventory & log panels
- [x] Serial-over-LAN terminal (xterm.js over the redirection channel)
- [x] Tauri desktop shell (sidecar spawn + token handoff)
- [x] KVM remote desktop (AMT RFB framebuffer decode + mouse/keyboard forwarding)
- [x] Boot control (one-time boot to PXE / CD / HDD / BIOS setup)
- [x] IDE-R (boot from remote ISO — ATAPI CD-ROM emulation in the sidecar)
- [x] Network info panel (AMT_EthernetPortSettings)
- [x] Device discovery (subnet / CIDR scan with AMT server-header check)
- [x] Account management (list / add digest user / enable-disable / remove)
- [ ] Certificate management, wireless config, system-defense, alarm clock
- [ ] Tauri packaging/signing polish

## License

MeshManager is licensed under the **Apache License 2.0** — see [`LICENSE`](LICENSE).

Parts of the redirection layer (Serial-over-LAN, KVM, IDE-R) are derived from
[MeshCommander](https://github.com/Ylianst/MeshCommander) (© Ylian
Saint-Hilaire, Apache-2.0), ported and modified — see [`NOTICE`](NOTICE).
Bundled open-source dependencies are listed in
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).

# Third-Party Notices

MeshManager is distributed with the following open-source components. Each
remains under its own license; copies of those licenses are available from the
respective projects (and via the package managers used to build MeshManager).
Apache-2.0 components are additionally covered by the `LICENSE` file in this
repository.

> This list covers direct dependencies. The Rust/Tauri build pulls a large
> transitive tree (the `windows`, `webview2-com`, `wry`, `tao`, etc. crates),
> all under permissive MIT and/or Apache-2.0 licenses. For a formal release,
> a complete machine-generated manifest can be produced with `cargo about`
> (Rust) and a license checker for the JS dependencies.

## Go sidecar (`amtd`)

| Component | License |
| --- | --- |
| github.com/device-management-toolkit/go-wsman-messages/v2 (Intel) | Apache-2.0 |
| github.com/go-chi/chi/v5 | MIT |
| github.com/gorilla/websocket | BSD-2-Clause |
| gopkg.in/natefinch/lumberjack.v2 | MIT |

## Desktop shell (Rust / Tauri)

| Component | License |
| --- | --- |
| tauri, tauri-build, tauri-plugin-shell, tauri-plugin-dialog | MIT OR Apache-2.0 |
| serde, serde_json | MIT OR Apache-2.0 |
| rand | MIT OR Apache-2.0 |
| (WebView2 / Windows bindings, wry, tao, …) | MIT OR Apache-2.0 |

## Frontend (`app`)

| Component | License |
| --- | --- |
| react, react-dom | MIT |
| @tanstack/react-query | MIT |
| zustand | MIT |
| clsx | MIT |
| fflate | MIT |
| lucide-react | ISC |
| @xterm/xterm, @xterm/addon-fit | MIT |
| @tauri-apps/api, @tauri-apps/cli, @tauri-apps/plugin-dialog | MIT OR Apache-2.0 |
| tailwindcss | MIT |
| vite | MIT |
| typescript | Apache-2.0 |

## Derived code

Parts of MeshManager are derived from **MeshCommander**
(https://github.com/Ylianst/MeshCommander), Copyright Ylian Saint-Hilaire,
Apache-2.0. See `NOTICE` for details and the affected files.

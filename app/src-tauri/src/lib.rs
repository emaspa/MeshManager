//! Tauri desktop shell for MeshManager.
//!
//! Responsibilities are deliberately thin: spawn the `amtd` Go sidecar with a
//! freshly generated bearer token on a loopback port, learn the chosen port
//! from the sidecar's first stdout line, and expose that endpoint + token to
//! the frontend via the `sidecar_info` command. All AMT logic lives in amtd.
//!
//! Logging: the shell resolves a per-user log directory and passes it to the
//! sidecar (`-log-dir`), so amtd writes rotating `amtd.log` files there. The
//! shell also tees the sidecar's stdout/stderr to `shell.log` in the same
//! directory and exposes `open_logs` so testers can grab everything for a bug
//! report from one place.

use std::fs::{File, OpenOptions};
use std::io::Write;
use std::path::PathBuf;
use std::sync::Mutex;

use rand::Rng;
use serde::Serialize;
use tauri::{Manager, State};
use tauri_plugin_shell::{process::CommandEvent, ShellExt};

#[derive(Default, Clone, Serialize)]
pub struct SidecarInfo {
    #[serde(rename = "baseUrl")]
    base_url: String,
    token: String,
}

#[derive(Default)]
struct AppState {
    info: Mutex<Option<SidecarInfo>>,
    log_dir: Mutex<Option<PathBuf>>,
}

/// Returns the sidecar endpoint + auth token for the frontend's API client.
#[tauri::command]
fn sidecar_info(state: State<AppState>) -> Result<SidecarInfo, String> {
    state
        .info
        .lock()
        .unwrap()
        .clone()
        .ok_or_else(|| "sidecar not ready".to_string())
}

/// Returns the path of the log directory (for display in the UI).
#[tauri::command]
fn log_dir(state: State<AppState>) -> String {
    state
        .log_dir
        .lock()
        .unwrap()
        .clone()
        .map(|p| p.to_string_lossy().into_owned())
        .unwrap_or_default()
}

/// Opens the log directory in the OS file manager.
#[tauri::command]
fn open_logs(state: State<AppState>) -> Result<(), String> {
    let dir = state
        .log_dir
        .lock()
        .unwrap()
        .clone()
        .ok_or_else(|| "log directory not available".to_string())?;

    #[cfg(target_os = "windows")]
    let result = std::process::Command::new("explorer").arg(&dir).spawn();
    #[cfg(target_os = "macos")]
    let result = std::process::Command::new("open").arg(&dir).spawn();
    #[cfg(all(unix, not(target_os = "macos")))]
    let result = std::process::Command::new("xdg-open").arg(&dir).spawn();

    result.map(|_| ()).map_err(|e| e.to_string())
}

fn random_token() -> String {
    const HEX: &[u8] = b"0123456789abcdef";
    let mut rng = rand::thread_rng();
    (0..32).map(|_| HEX[rng.gen_range(0..16)] as char).collect()
}

/// Opens a URL in the user's default browser.
#[tauri::command]
fn open_external(url: String) -> Result<(), String> {
    // Only allow http(s) so the command can't be coaxed into launching arbitrary
    // programs.
    if !(url.starts_with("https://") || url.starts_with("http://")) {
        return Err("only http(s) URLs are allowed".into());
    }
    #[cfg(target_os = "windows")]
    let result = std::process::Command::new("cmd").args(["/C", "start", "", &url]).spawn();
    #[cfg(target_os = "macos")]
    let result = std::process::Command::new("open").arg(&url).spawn();
    #[cfg(all(unix, not(target_os = "macos")))]
    let result = std::process::Command::new("xdg-open").arg(&url).spawn();

    result.map(|_| ()).map_err(|e| e.to_string())
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .manage(AppState::default())
        .invoke_handler(tauri::generate_handler![sidecar_info, log_dir, open_logs, open_external])
        .setup(|app| {
            // Show the app version in the window title bar.
            if let Some(win) = app.get_webview_window("main") {
                let _ = win.set_title(&format!("MeshManager v{}", app.package_info().version));
            }

            // Resolve a per-user log directory (falls back to temp).
            let log_dir = app
                .path()
                .app_log_dir()
                .unwrap_or_else(|_| std::env::temp_dir().join("MeshManager"));
            let _ = std::fs::create_dir_all(&log_dir);
            *app.state::<AppState>().log_dir.lock().unwrap() = Some(log_dir.clone());

            // shell.log captures the sidecar's raw stdout/stderr as a safety net
            // (e.g. if amtd dies before it can write its own rotating log).
            let mut shell_log: Option<File> = OpenOptions::new()
                .create(true)
                .append(true)
                .open(log_dir.join("shell.log"))
                .ok();
            if let Some(f) = shell_log.as_mut() {
                let _ = writeln!(f, "--- MeshManager shell starting, log dir: {} ---", log_dir.display());
            }

            let token = random_token();
            let (mut rx, child) = app
                .shell()
                .sidecar("amtd")
                .expect("amtd sidecar not found")
                .args([
                    "-addr",
                    "127.0.0.1:0",
                    "-token",
                    &token,
                    "-log-dir",
                    &log_dir.to_string_lossy(),
                ])
                .spawn()
                .expect("failed to spawn amtd sidecar");

            // Block until the sidecar announces its port, teeing output as we go.
            let token_for_info = token.clone();
            let info = tauri::async_runtime::block_on(async {
                while let Some(event) = rx.recv().await {
                    let line = match &event {
                        CommandEvent::Stdout(b) | CommandEvent::Stderr(b) => {
                            String::from_utf8_lossy(b).into_owned()
                        }
                        _ => String::new(),
                    };
                    if !line.is_empty() {
                        if let Some(f) = shell_log.as_mut() {
                            let _ = write!(f, "{line}");
                            if !line.ends_with('\n') {
                                let _ = writeln!(f);
                            }
                        }
                    }
                    if let CommandEvent::Stdout(_) = event {
                        if let Some(addr) = line.trim().strip_prefix("AMTD_LISTENING ") {
                            return Some(SidecarInfo {
                                base_url: format!("http://{}", addr.trim()),
                                token: token_for_info.clone(),
                            });
                        }
                    }
                }
                None
            });

            if let Some(info) = info {
                *app.state::<AppState>().info.lock().unwrap() = Some(info);
            } else if let Some(f) = shell_log.as_mut() {
                let _ = writeln!(f, "ERROR: amtd exited before announcing a listen address");
            }

            // Keep draining + teeing the sidecar's output for the rest of its
            // life, and hold the child handle so it is killed when the app exits.
            tauri::async_runtime::spawn(async move {
                let _child = child;
                while let Some(event) = rx.recv().await {
                    if let CommandEvent::Stdout(b) | CommandEvent::Stderr(b) = &event {
                        if let Some(f) = shell_log.as_mut() {
                            let line = String::from_utf8_lossy(b);
                            let _ = write!(f, "{line}");
                            if !line.ends_with('\n') {
                                let _ = writeln!(f);
                            }
                        }
                    }
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running MeshManager");
}

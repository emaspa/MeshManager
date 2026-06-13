//! Tauri desktop shell for MeshManager.
//!
//! Responsibilities are deliberately thin: spawn the `amtd` Go sidecar with a
//! freshly generated bearer token on a loopback port, learn the chosen port
//! from the sidecar's first stdout line, and expose that endpoint + token to
//! the frontend via the `sidecar_info` command. All AMT logic lives in amtd.

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

fn random_token() -> String {
    const HEX: &[u8] = b"0123456789abcdef";
    let mut rng = rand::thread_rng();
    (0..32).map(|_| HEX[rng.gen_range(0..16)] as char).collect()
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(AppState::default())
        .invoke_handler(tauri::generate_handler![sidecar_info])
        .setup(|app| {
            let token = random_token();
            let (mut rx, child) = app
                .shell()
                .sidecar("amtd")
                .expect("amtd sidecar not found")
                .args(["-addr", "127.0.0.1:0", "-token", &token])
                .spawn()
                .expect("failed to spawn amtd sidecar");

            // Block until the sidecar announces its port, then store the info.
            let token_for_info = token.clone();
            let info = tauri::async_runtime::block_on(async {
                while let Some(event) = rx.recv().await {
                    if let CommandEvent::Stdout(line) = event {
                        let text = String::from_utf8_lossy(&line);
                        if let Some(addr) = text.trim().strip_prefix("AMTD_LISTENING ") {
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
            } else {
                eprintln!("amtd exited before announcing a listen address");
            }

            // Keep draining the sidecar's output so its stdout pipe never fills,
            // and hold the child handle so it is killed when the app exits.
            tauri::async_runtime::spawn(async move {
                let _child = child;
                while rx.recv().await.is_some() {}
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running MeshManager");
}

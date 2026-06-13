// Resolves how to reach the amtd sidecar.
//
// In the packaged Tauri app, the Rust core spawns amtd, learns its port, and
// exposes the endpoint + auth token via the `sidecar_info` command. In plain
// browser dev (`bun run dev`), we fall back to a fixed endpoint and a dev
// token supplied via Vite env, so the whole UI is usable without Tauri.

export interface SidecarInfo {
  baseUrl: string;
  token: string;
}

let cached: Promise<SidecarInfo> | null = null;

async function resolve(): Promise<SidecarInfo> {
  // Detect Tauri (v2 exposes the IPC bridge on window.__TAURI_INTERNALS__).
  const isTauri = typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
  if (isTauri) {
    const { invoke } = await import("@tauri-apps/api/core");
    return await invoke<SidecarInfo>("sidecar_info");
  }
  // Browser dev fallback.
  return {
    baseUrl: import.meta.env.VITE_AMTD_URL ?? "http://127.0.0.1:7777",
    token: import.meta.env.VITE_AMTD_TOKEN ?? "",
  };
}

export function sidecar(): Promise<SidecarInfo> {
  if (!cached) cached = resolve();
  return cached;
}

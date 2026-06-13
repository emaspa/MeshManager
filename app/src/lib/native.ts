// Thin bridge to Tauri commands with graceful no-ops in browser dev mode.

export const APP_NAME = "MeshManager";
export const REPO_URL = "https://github.com/emaspa/meshmanager";

export function isTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

async function invokeTauri<T>(cmd: string, args?: Record<string, unknown>): Promise<T | undefined> {
  if (!isTauri()) return undefined;
  try {
    const { invoke } = await import("@tauri-apps/api/core");
    return await invoke<T>(cmd, args);
  } catch {
    return undefined;
  }
}

/** Opens the OS log folder (Tauri only). */
export async function openLogs(): Promise<void> {
  await invokeTauri("open_logs");
}

/** Reads the configured log retention in days (Tauri only; default 30). */
export async function getLogRetention(): Promise<number> {
  const days = await invokeTauri<number>("get_log_retention");
  return typeof days === "number" ? days : 30;
}

/** Persists the log retention preference in days. Applies on next launch. */
export async function setLogRetention(days: number): Promise<void> {
  await invokeTauri("set_log_retention", { days });
}

/** Opens a URL in the default browser (Tauri command, or a new tab in dev). */
export async function openExternal(url: string): Promise<void> {
  if (isTauri()) {
    await invokeTauri("open_external", { url });
    return;
  }
  window.open(url, "_blank", "noopener,noreferrer");
}

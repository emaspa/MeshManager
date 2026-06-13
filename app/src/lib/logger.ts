// Client-side logging: forwards UI errors to the sidecar so they land in the
// same rotating log file as everything else — making tester bug reports
// self-contained. Logging must never throw or recurse, so it uses fetch
// directly (not the api client) and swallows its own failures.
import { sidecar } from "./sidecar";

type Level = "error" | "warn" | "info" | "debug";

export async function clientLog(level: Level, message: string, context = "", stack = "") {
  try {
    const { baseUrl, token } = await sidecar();
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (token) headers.Authorization = `Bearer ${token}`;
    await fetch(`${baseUrl}/api/log`, {
      method: "POST",
      headers,
      body: JSON.stringify({ level, message, context, stack }),
    });
  } catch {
    /* logging is best-effort */
  }
}

let installed = false;

export function installErrorLogging() {
  if (installed) return;
  installed = true;

  window.addEventListener("error", (e) => {
    void clientLog(
      "error",
      e.message || "window error",
      "window.onerror",
      e.error?.stack ?? `${e.filename}:${e.lineno}:${e.colno}`,
    );
  });

  window.addEventListener("unhandledrejection", (e) => {
    const r = e.reason;
    void clientLog("error", r?.message ?? String(r), "unhandledrejection", r?.stack ?? "");
  });

  void clientLog("info", `UI started — ${navigator.userAgent}`, "startup");
}

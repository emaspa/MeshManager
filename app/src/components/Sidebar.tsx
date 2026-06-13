import { useQuery } from "@tanstack/react-query";
import { Plus, Server, Wifi, WifiOff, Radar, FileText } from "lucide-react";
import clsx from "clsx";
import { api } from "../lib/api";
import { useUi } from "../store";
import { Button, Spinner } from "../lib/ui";

function isTauri() {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

async function openLogs() {
  if (!isTauri()) return;
  try {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("open_logs");
  } catch {
    /* ignore */
  }
}

export function Sidebar() {
  const { selectedId, select, openConnect, setDiscoverOpen } = useUi();
  const health = useQuery({ queryKey: ["health"], queryFn: api.health, refetchInterval: 5000 });
  const devices = useQuery({
    queryKey: ["devices"],
    queryFn: api.listDevices,
    refetchInterval: 4000,
  });

  return (
    <aside className="flex w-64 flex-col border-r border-[--color-border] bg-[--color-panel]">
      <div className="flex items-center gap-2 border-b border-[--color-border] px-4 py-3">
        <Server className="h-5 w-5 text-[--color-accent]" />
        <div className="font-semibold">MeshManager</div>
        <div className="ml-auto">
          {health.isSuccess ? (
            <Wifi className="h-4 w-4 text-[--color-good]" />
          ) : (
            <WifiOff className="h-4 w-4 text-[--color-bad]" />
          )}
        </div>
      </div>

      <div className="flex items-center justify-between px-4 py-2">
        <span className="text-xs uppercase tracking-wide text-[--color-muted]">Devices</span>
        <div className="flex gap-1">
          <Button variant="ghost" className="px-1.5 py-1" title="Discover devices" onClick={() => setDiscoverOpen(true)}>
            <Radar className="h-4 w-4" />
          </Button>
          <Button variant="ghost" className="px-1.5 py-1" title="Add device" onClick={() => openConnect()}>
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-2">
        {devices.isLoading && (
          <div className="flex justify-center py-4">
            <Spinner />
          </div>
        )}
        {devices.data?.length === 0 && (
          <div className="px-2 py-4 text-center text-xs text-[--color-muted]">
            No connected devices
          </div>
        )}
        {devices.data?.map((d) => (
          <button
            key={d.id}
            onClick={() => select(d.id)}
            className={clsx(
              "mb-1 flex w-full flex-col items-start rounded-md px-3 py-2 text-left transition-colors",
              selectedId === d.id ? "bg-[--color-accent]/15" : "hover:bg-[--color-panel-2]",
            )}
          >
            <span className="text-sm font-medium">{d.name || d.host}</span>
            <span className="text-xs text-[--color-muted]">
              {d.host}:{d.port} {d.tls ? "· TLS" : ""}
            </span>
          </button>
        ))}
      </div>

      <div className="flex items-center justify-between border-t border-[--color-border] px-4 py-2 text-xs text-[--color-muted]">
        <span>{health.data ? `amtd v${health.data.version}` : "connecting to amtd…"}</span>
        {isTauri() && (
          <button
            onClick={openLogs}
            title="Open log folder (for bug reports)"
            className="flex items-center gap-1 hover:text-[--color-text]"
          >
            <FileText className="h-3.5 w-3.5" /> Logs
          </button>
        )}
      </div>
    </aside>
  );
}

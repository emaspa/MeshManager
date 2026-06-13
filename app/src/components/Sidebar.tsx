import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Server, Wifi, WifiOff, Radar, FileText, Pencil, Trash2 } from "lucide-react";
import clsx from "clsx";
import { api } from "../lib/api";
import { useUi } from "../store";
import { useBookmarks, effectivePort, type Bookmark } from "../lib/bookmarks";
import { Button } from "../lib/ui";

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
  const { bookmarks, remove } = useBookmarks();
  const qc = useQueryClient();

  const health = useQuery({ queryKey: ["health"], queryFn: api.health, refetchInterval: 5000 });
  const devices = useQuery({
    queryKey: ["devices"],
    queryFn: api.listDevices,
    refetchInterval: 4000,
  });

  // Match a bookmark to a live session by host + effective port.
  const sessionFor = (b: Bookmark) =>
    devices.data?.find((d) => d.host === b.host && d.port === effectivePort(b));

  const connect = useMutation({
    mutationFn: (b: Bookmark) =>
      api.connect({
        host: b.host,
        port: b.port,
        username: b.username,
        password: b.password ?? "",
        tls: b.tls,
        insecure: b.insecure,
        name: b.name,
      }),
    onSuccess: (device) => {
      qc.invalidateQueries({ queryKey: ["devices"] });
      select(device.id);
    },
  });

  function openBookmark(b: Bookmark) {
    const session = sessionFor(b);
    if (session) {
      select(session.id);
    } else if (b.password) {
      connect.mutate(b); // saved credentials → connect directly
    } else {
      openConnect({ ...b, bookmarkId: b.id }); // need a password
    }
  }

  function editBookmark(b: Bookmark) {
    openConnect({ ...b, bookmarkId: b.id, edit: true });
  }

  return (
    <aside className="flex w-64 flex-col border-r border-(--color-border) bg-(--color-panel)">
      <div className="flex items-center gap-2 border-b border-(--color-border) px-4 py-3">
        <Server className="h-5 w-5 text-(--color-accent)" />
        <div className="font-semibold">MeshManager</div>
        <div className="ml-auto">
          {health.isSuccess ? (
            <Wifi className="h-4 w-4 text-(--color-good)" />
          ) : (
            <WifiOff className="h-4 w-4 text-(--color-bad)" />
          )}
        </div>
      </div>

      <div className="flex items-center justify-between px-4 py-2">
        <span className="text-xs uppercase tracking-wide text-(--color-muted)">Devices</span>
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
        {bookmarks.length === 0 && (
          <div className="px-2 py-4 text-center text-xs text-(--color-muted)">
            No saved devices. Click + to add one.
          </div>
        )}
        {bookmarks.map((b) => {
          const session = sessionFor(b);
          const connected = !!session;
          const active = connected && session!.id === selectedId;
          const busy = connect.isPending && connect.variables?.id === b.id;
          return (
            <div
              key={b.id}
              className={clsx(
                "group mb-1 flex items-center rounded-md px-2 py-2 transition-colors",
                active ? "bg-(--color-accent)/15" : "hover:bg-(--color-panel-2)",
              )}
            >
              <button onClick={() => openBookmark(b)} className="flex min-w-0 flex-1 items-center gap-2 text-left">
                <span
                  className={clsx(
                    "h-2 w-2 shrink-0 rounded-full",
                    connected ? "bg-(--color-good)" : "bg-(--color-muted)/50",
                  )}
                  title={connected ? "Connected" : "Disconnected"}
                />
                <span className="min-w-0">
                  <span className="block truncate text-sm font-medium">{b.name || b.host}</span>
                  <span className="block truncate text-xs text-(--color-muted)">
                    {b.host}:{effectivePort(b)} {b.tls ? "· TLS" : ""} {busy ? "· connecting…" : ""}
                  </span>
                </span>
              </button>
              <div className="ml-1 flex shrink-0 gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                <button
                  onClick={() => editBookmark(b)}
                  title="Edit bookmark"
                  className="rounded p-1 text-(--color-muted) hover:bg-(--color-border) hover:text-(--color-text)"
                >
                  <Pencil className="h-3.5 w-3.5" />
                </button>
                <button
                  onClick={() => {
                    if (active) select(null);
                    remove(b.id);
                  }}
                  title="Remove bookmark"
                  className="rounded p-1 text-(--color-muted) hover:bg-(--color-border) hover:text-(--color-bad)"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          );
        })}
      </div>

      <div className="flex items-center justify-between border-t border-(--color-border) px-4 py-2 text-xs text-(--color-muted)">
        <span>{health.data ? `amtd v${health.data.version}` : "connecting to amtd…"}</span>
        {isTauri() && (
          <button
            onClick={openLogs}
            title="Open log folder (for bug reports)"
            className="flex items-center gap-1 hover:text-(--color-text)"
          >
            <FileText className="h-3.5 w-3.5" /> Logs
          </button>
        )}
      </div>
    </aside>
  );
}

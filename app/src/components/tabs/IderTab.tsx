import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Disc, Play, Square, FolderOpen } from "lucide-react";
import { api, ApiError } from "../../lib/api";
import { Badge, Button, Card, Field, Input } from "../../lib/ui";

export function IderTab({ id }: { id: string }) {
  const [iso, setIso] = useState("");

  const status = useQuery({
    queryKey: ["ider", id],
    queryFn: () => api.iderStatus(id),
    refetchInterval: (q) => (q.state.data?.active ? 1500 : false),
  });
  const active = status.data?.active ?? false;
  const stats = status.data?.stats;

  const start = useMutation({
    mutationFn: (boot: boolean) => api.iderStart(id, iso, boot),
    onSuccess: () => status.refetch(),
  });
  const stop = useMutation({
    mutationFn: () => api.iderStop(id),
    onSuccess: () => status.refetch(),
  });

  async function pickIso() {
    // Native file dialog when running under Tauri; otherwise the path field.
    if (typeof window !== "undefined" && "__TAURI_INTERNALS__" in window) {
      try {
        const { open } = await import("@tauri-apps/plugin-dialog");
        const picked = await open({
          multiple: false,
          filters: [{ name: "Disc image", extensions: ["iso", "img"] }],
        });
        if (typeof picked === "string") setIso(picked);
      } catch {
        /* dialog plugin not available; fall back to manual entry */
      }
    }
  }

  return (
    <div className="max-w-2xl space-y-4">
      <Card>
        <div className="mb-3 flex items-center gap-2">
          <Disc className="h-5 w-5 text-[--color-accent]" />
          <h3 className="font-medium">IDE-R — Boot from ISO</h3>
          {active && <Badge tone="good">mounted</Badge>}
        </div>
        <p className="mb-4 text-sm text-[--color-muted]">
          Serves a local ISO to the device as a virtual CD-ROM over IDE redirection. “Mount &amp;
          Boot” also sets the one-time boot device and resets the machine.
        </p>

        <div className="flex items-end gap-2">
          <div className="flex-1">
            <Field label="ISO path (on this machine)">
              <Input
                value={iso}
                onChange={(e) => setIso(e.target.value)}
                placeholder="C:\\images\\boot.iso"
                disabled={active}
              />
            </Field>
          </div>
          <Button onClick={pickIso} disabled={active} title="Browse">
            <FolderOpen className="h-4 w-4" />
          </Button>
        </div>

        <div className="mt-4 flex gap-2">
          {active ? (
            <Button variant="danger" onClick={() => stop.mutate()} disabled={stop.isPending}>
              <Square className="h-4 w-4" /> Eject / Stop
            </Button>
          ) : (
            <>
              <Button
                variant="primary"
                onClick={() => start.mutate(true)}
                disabled={!iso || start.isPending}
              >
                <Play className="h-4 w-4" /> Mount &amp; Boot
              </Button>
              <Button onClick={() => start.mutate(false)} disabled={!iso || start.isPending}>
                Mount only
              </Button>
            </>
          )}
        </div>

        {start.isError && (
          <div className="mt-3 rounded-md bg-[--color-bad]/15 px-3 py-2 text-sm text-[--color-bad]">
            {start.error instanceof ApiError ? start.error.message : "Failed to start IDE-R"}
          </div>
        )}
      </Card>

      {active && stats && (
        <Card>
          <h3 className="mb-3 font-medium">Transfer</h3>
          <div className="grid grid-cols-3 gap-4 text-sm">
            <Stat label="ISO size" value={fmtBytes(stats.isoSize)} />
            <Stat label="Sectors served" value={stats.sectorsRead.toLocaleString()} />
            <Stat label="Sent to device" value={fmtBytes(stats.bytesToAmt)} />
          </div>
          {stats.error && (
            <div className="mt-3 rounded-md bg-[--color-bad]/15 px-3 py-2 text-sm text-[--color-bad]">
              {stats.error}
            </div>
          )}
        </Card>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-xs text-[--color-muted]">{label}</span>
      <span className="font-mono text-base">{value}</span>
    </div>
  );
}

function fmtBytes(n: number): string {
  if (!n) return "—";
  const units = ["B", "KB", "MB", "GB"];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

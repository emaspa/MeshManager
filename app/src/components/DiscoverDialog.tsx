import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { X, Radar, ArrowRight } from "lucide-react";
import { api, ApiError, type Discovered } from "../lib/api";
import { useUi } from "../store";
import { Button, Field, Input, Spinner } from "../lib/ui";

export function DiscoverDialog() {
  const setDiscoverOpen = useUi((s) => s.setDiscoverOpen);
  const openConnect = useUi((s) => s.openConnect);

  const [cidr, setCidr] = useState("192.168.1.0/24");
  const [tls, setTls] = useState(false);

  const scan = useMutation<Discovered[]>({
    mutationFn: () => api.discover(cidr, undefined, tls),
  });

  function useDevice(d: Discovered) {
    setDiscoverOpen(false);
    openConnect({ host: d.host, port: d.port, tls: d.tls });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="flex max-h-[80vh] w-[520px] flex-col rounded-xl border border-[--color-border] bg-[--color-panel] p-5 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-lg font-semibold">
            <Radar className="h-5 w-5 text-[--color-accent]" /> Discover Devices
          </h2>
          <Button variant="ghost" className="px-1.5 py-1" onClick={() => setDiscoverOpen(false)}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <form
          className="flex items-end gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            scan.mutate();
          }}
        >
          <div className="flex-1">
            <Field label="Subnet (CIDR) or IP">
              <Input value={cidr} onChange={(e) => setCidr(e.target.value)} placeholder="192.168.1.0/24" />
            </Field>
          </div>
          <label className="flex items-center gap-2 pb-2 text-sm">
            <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} />
            TLS
          </label>
          <Button type="submit" variant="primary" disabled={scan.isPending}>
            {scan.isPending ? "Scanning…" : "Scan"}
          </Button>
        </form>

        <div className="mt-4 flex-1 overflow-y-auto">
          {scan.isPending && (
            <div className="flex items-center justify-center gap-2 py-8 text-sm text-[--color-muted]">
              <Spinner /> Scanning {cidr}…
            </div>
          )}
          {scan.isError && (
            <div className="rounded-md bg-[--color-bad]/15 px-3 py-2 text-sm text-[--color-bad]">
              {scan.error instanceof ApiError ? scan.error.message : "Scan failed"}
            </div>
          )}
          {scan.data && scan.data.length === 0 && (
            <p className="py-6 text-center text-sm text-[--color-muted]">
              No devices responded on port {tls ? 16993 : 16992}.
            </p>
          )}
          {scan.data && scan.data.length > 0 && (
            <ul className="divide-y divide-[--color-border]">
              {scan.data.map((d) => (
                <li key={d.host} className="flex items-center gap-3 py-2">
                  <div className="flex-1">
                    <div className="font-mono text-sm">
                      {d.host}:{d.port}
                    </div>
                    <div className="text-xs text-[--color-muted]">
                      {d.isAmt ? "Intel AMT" : d.server || "open port"}
                    </div>
                  </div>
                  <Button onClick={() => useDevice(d)}>
                    Connect <ArrowRight className="h-4 w-4" />
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}

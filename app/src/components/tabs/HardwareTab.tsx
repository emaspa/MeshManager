import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { Card, Spinner } from "../../lib/ui";

export function HardwareTab({ id }: { id: string }) {
  const hw = useQuery({ queryKey: ["hardware", id], queryFn: () => api.hardware(id) });

  if (hw.isLoading)
    return <div className="flex h-40 items-center justify-center"><Spinner /></div>;
  if (hw.isError)
    return <p className="text-[--color-bad]">{(hw.error as Error).message}</p>;

  const d = hw.data!;
  return (
    <div className="space-y-4">
      <Card>
        <h3 className="mb-3 font-medium">System</h3>
        <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm md:grid-cols-3">
          <KV k="Manufacturer" v={d.system.manufacturer} />
          <KV k="Model" v={d.system.model} />
          <KV k="Serial" v={d.system.serialNumber} />
          <KV k="Version" v={d.system.version} />
        </div>
      </Card>

      <Card>
        <h3 className="mb-3 font-medium">Processors ({d.processors.length})</h3>
        <Table
          head={["ID", "Max MHz", "Current MHz"]}
          rows={d.processors.map((p) => [p.id, String(p.maxClockMhz), String(p.currentClockMhz)])}
        />
      </Card>

      <Card>
        <h3 className="mb-3 font-medium">Memory ({d.memory.length})</h3>
        <Table
          head={["Bank", "Capacity (MB)", "Speed (MHz)", "Manufacturer", "Part #"]}
          rows={d.memory.map((m) => [
            m.bankLabel,
            String(m.capacityMb),
            String(m.speedMhz),
            m.manufacturer,
            m.partNumber,
          ])}
        />
      </Card>

      <Card>
        <h3 className="mb-3 font-medium">Storage ({d.disks.length})</h3>
        <Table
          head={["Device", "Name", "Max Media (KB)"]}
          rows={d.disks.map((x) => [x.deviceId, x.elementName, String(x.maxMediaKb)])}
        />
      </Card>
    </div>
  );
}

function KV({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-xs text-[--color-muted]">{k}</span>
      <span className="font-mono">{v || "—"}</span>
    </div>
  );
}

function Table({ head, rows }: { head: string[]; rows: string[][] }) {
  if (rows.length === 0)
    return <p className="text-sm text-[--color-muted]">No data reported.</p>;
  return (
    <table className="w-full text-left text-sm">
      <thead>
        <tr className="text-xs uppercase text-[--color-muted]">
          {head.map((h) => (
            <th key={h} className="pb-2 pr-4 font-medium">{h}</th>
          ))}
        </tr>
      </thead>
      <tbody className="font-mono">
        {rows.map((r, i) => (
          <tr key={i} className="border-t border-[--color-border]">
            {r.map((c, j) => (
              <td key={j} className="py-1.5 pr-4">{c || "—"}</td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

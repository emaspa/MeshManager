import { useQuery } from "@tanstack/react-query";
import { RotateCcw } from "lucide-react";
import { api } from "../../lib/api";
import { Badge, Button, Spinner } from "../../lib/ui";

function tone(sev: string): "bad" | "warn" | "muted" {
  const s = sev.toLowerCase();
  if (s.includes("crit") || s.includes("error") || s.includes("fatal")) return "bad";
  if (s.includes("warn")) return "warn";
  return "muted";
}

export function EventLogTab({ id }: { id: string }) {
  const log = useQuery({ queryKey: ["eventlog", id], queryFn: () => api.eventLog(id) });

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm text-[--color-muted]">{log.data?.length ?? 0} entries</span>
        <Button onClick={() => log.refetch()}>
          <RotateCcw className="h-4 w-4" /> Refresh
        </Button>
      </div>
      {log.isLoading && <div className="flex justify-center py-8"><Spinner /></div>}
      {log.isError && <p className="text-[--color-bad]">{(log.error as Error).message}</p>}
      {log.data && (
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="text-xs uppercase text-[--color-muted]">
              <th className="pb-2 pr-4 font-medium">Time</th>
              <th className="pb-2 pr-4 font-medium">Severity</th>
              <th className="pb-2 pr-4 font-medium">Entity</th>
              <th className="pb-2 pr-4 font-medium">Description</th>
            </tr>
          </thead>
          <tbody>
            {log.data.map((e, i) => (
              <tr key={i} className="border-t border-[--color-border]">
                <td className="whitespace-nowrap py-1.5 pr-4 font-mono text-xs">
                  {new Date(e.time).toLocaleString()}
                </td>
                <td className="py-1.5 pr-4"><Badge tone={tone(e.severity)}>{e.severity || "info"}</Badge></td>
                <td className="py-1.5 pr-4">{e.entity}</td>
                <td className="py-1.5 pr-4">{e.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

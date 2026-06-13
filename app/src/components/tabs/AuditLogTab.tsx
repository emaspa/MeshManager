import { useQuery } from "@tanstack/react-query";
import { RotateCcw } from "lucide-react";
import { api } from "../../lib/api";
import { Button, Spinner } from "../../lib/ui";

export function AuditLogTab({ id }: { id: string }) {
  const log = useQuery({ queryKey: ["auditlog", id], queryFn: () => api.auditLog(id) });

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
              <th className="pb-2 pr-4 font-medium">App</th>
              <th className="pb-2 pr-4 font-medium">Event</th>
              <th className="pb-2 pr-4 font-medium">Initiator</th>
              <th className="pb-2 pr-4 font-medium">Address</th>
            </tr>
          </thead>
          <tbody>
            {log.data.map((e, i) => (
              <tr key={i} className="border-t border-[--color-border]">
                <td className="whitespace-nowrap py-1.5 pr-4 font-mono text-xs">
                  {new Date(e.time).toLocaleString()}
                </td>
                <td className="py-1.5 pr-4">{e.app}</td>
                <td className="py-1.5 pr-4">{e.event}{e.extended ? ` — ${e.extended}` : ""}</td>
                <td className="py-1.5 pr-4">{e.initiator}</td>
                <td className="py-1.5 pr-4 font-mono text-xs">{e.netAddress}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

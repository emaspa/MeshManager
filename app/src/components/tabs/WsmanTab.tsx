import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Braces, RotateCcw } from "lucide-react";
import { api } from "../../lib/api";
import { Button, Spinner } from "../../lib/ui";

// Read-only WS-MAN browser: enumerate any supported AMT/CIM/IPS class and show
// the structured result. Useful for inspecting certs, Wi-Fi, CIRA, opt-in, TLS,
// etc. before/without dedicated panels.
export function WsmanTab({ id }: { id: string }) {
  const [cls, setCls] = useState("");

  const classes = useQuery({ queryKey: ["browse-classes", id], queryFn: () => api.browseClasses(id) });
  const result = useQuery({
    queryKey: ["browse", id, cls],
    queryFn: () => api.browse(id, cls),
    enabled: cls !== "",
  });

  return (
    <div className="flex h-full flex-col">
      <div className="mb-3 flex items-center gap-2">
        <Braces className="h-5 w-5 text-[--color-accent]" />
        <span className="font-medium">WS-MAN Browser</span>
        <select
          value={cls}
          onChange={(e) => setCls(e.target.value)}
          className="ml-2 rounded-md border border-[--color-border] bg-[--color-bg] px-2 py-1.5 text-sm outline-none focus:border-[--color-accent]"
        >
          <option value="">Select a class…</option>
          {classes.data?.map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
        {cls && (
          <Button onClick={() => result.refetch()} title="Re-fetch">
            <RotateCcw className="h-4 w-4" />
          </Button>
        )}
      </div>

      <div className="flex-1 overflow-auto rounded-lg border border-[--color-border] bg-[--color-bg] p-3">
        {!cls && <p className="text-sm text-[--color-muted]">Pick a class to enumerate it.</p>}
        {cls && result.isLoading && <div className="flex justify-center py-8"><Spinner /></div>}
        {cls && result.isError && (
          <p className="text-[--color-bad]">{(result.error as Error).message}</p>
        )}
        {cls && result.data !== undefined && !result.isLoading && (
          <pre className="whitespace-pre-wrap break-words font-mono text-xs leading-relaxed">
            {JSON.stringify(result.data, null, 2)}
          </pre>
        )}
      </div>
    </div>
  );
}

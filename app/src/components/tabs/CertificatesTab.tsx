import { useQuery } from "@tanstack/react-query";
import { RotateCcw } from "lucide-react";
import { api } from "../../lib/api";
import { Badge, Button, Spinner } from "../../lib/ui";

export function CertificatesTab({ id }: { id: string }) {
  const certs = useQuery({ queryKey: ["certs", id], queryFn: () => api.certificates(id) });

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm text-(--color-muted)">{certs.data?.length ?? 0} certificate(s)</span>
        <Button onClick={() => certs.refetch()}>
          <RotateCcw className="h-4 w-4" /> Refresh
        </Button>
      </div>
      {certs.isLoading && <div className="flex justify-center py-8"><Spinner /></div>}
      {certs.isError && <p className="text-(--color-bad)">{(certs.error as Error).message}</p>}
      {certs.data?.length === 0 && (
        <p className="text-sm text-(--color-muted)">No certificates stored on this device.</p>
      )}
      {certs.data && certs.data.length > 0 && (
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="text-xs uppercase text-(--color-muted)">
              <th className="pb-2 pr-4 font-medium">Subject</th>
              <th className="pb-2 pr-4 font-medium">Issuer</th>
              <th className="pb-2 pr-4 font-medium">Type</th>
            </tr>
          </thead>
          <tbody>
            {certs.data.map((c) => (
              <tr key={c.instanceId} className="border-t border-(--color-border) align-top">
                <td className="py-2 pr-4 font-mono text-xs">{c.subject || c.name || "-"}</td>
                <td className="py-2 pr-4 font-mono text-xs">{c.issuer || "-"}</td>
                <td className="py-2 pr-4">
                  <Badge tone={c.trustedRoot ? "good" : "muted"}>
                    {c.trustedRoot ? "Trusted root" : "Certificate"}
                  </Badge>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

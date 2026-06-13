import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { Card, Spinner } from "../../lib/ui";

export function Overview({ id }: { id: string }) {
  const info = useQuery({ queryKey: ["info", id], queryFn: () => api.info(id) });

  if (info.isLoading) return <Centered><Spinner /></Centered>;
  if (info.isError)
    return <Centered><span className="text-(--color-bad)">{(info.error as Error).message}</span></Centered>;

  const d = info.data!;
  const rows: [string, string][] = [
    ["Hostname", d.hostname || "—"],
    ["Domain", d.domainName || "—"],
    ["UUID", d.uuid || "—"],
    ["Digest Realm", d.digestRealm || "—"],
    ["Control Mode", d.controlMode || "—"],
    ["Provisioning", d.provisioningState || "—"],
    ["Network", d.networkEnabled ? "Enabled" : "Disabled"],
  ];

  return (
    <div className="grid grid-cols-2 gap-4">
      <Card>
        <h3 className="mb-3 font-medium">System</h3>
        <dl className="space-y-2 text-sm">
          {rows.map(([k, v]) => (
            <div key={k} className="flex justify-between gap-4">
              <dt className="text-(--color-muted)">{k}</dt>
              <dd className="truncate text-right font-mono">{v}</dd>
            </div>
          ))}
        </dl>
      </Card>

      <Card>
        <h3 className="mb-3 font-medium">Firmware Versions</h3>
        {Object.keys(d.versions).length === 0 ? (
          <p className="text-sm text-(--color-muted)">No version data reported.</p>
        ) : (
          <dl className="space-y-2 text-sm">
            {Object.entries(d.versions).map(([k, v]) => (
              <div key={k} className="flex justify-between gap-4">
                <dt className="text-(--color-muted)">{k}</dt>
                <dd className="text-right font-mono">{v}</dd>
              </div>
            ))}
          </dl>
        )}
      </Card>
    </div>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return <div className="flex h-40 items-center justify-center">{children}</div>;
}

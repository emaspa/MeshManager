import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { Card, Spinner } from "../../lib/ui";

export function Overview({ id }: { id: string }) {
  const info = useQuery({ queryKey: ["info", id], queryFn: () => api.info(id) });

  if (info.isLoading) return <Centered><Spinner /></Centered>;
  if (info.isError)
    return <Centered><span className="text-(--color-bad)">{(info.error as Error).message}</span></Centered>;

  const d = info.data!;
  const amt = d.versions["AMT"] || d.versions["AMTApps"];
  const me = amt ? `v${amt}${d.controlMode ? ` · ${d.controlMode}` : ""}` : d.controlMode || "-";
  const rows: [string, string][] = [
    ["Hostname", d.hostname || "-"],
    ["Domain", d.domainName || "-"],
    ["System ID (UUID)", d.uuid || "-"],
    ["Intel ME", me],
    ["Provisioning", d.provisioningState || "-"],
    ["User Consent", d.userConsent || "-"],
    ["Network", d.networkEnabled ? "Enabled" : "Disabled"],
    ["Device Time", d.deviceTime || "-"],
    ["Digest Realm", d.digestRealm || "-"],
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
        {d.activeFeatures?.length > 0 && (
          <div className="mt-3 border-t border-(--color-border) pt-3">
            <div className="mb-2 text-(--color-muted)">Active Features</div>
            <div className="flex flex-wrap gap-1">
              {d.activeFeatures.map((f) => (
                <span key={f} className="rounded bg-(--color-good)/15 px-2 py-0.5 text-xs text-(--color-good)">{f}</span>
              ))}
            </div>
          </div>
        )}
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

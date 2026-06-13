import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { Badge, Card, Spinner } from "../../lib/ui";

export function NetworkTab({ id }: { id: string }) {
  const net = useQuery({ queryKey: ["network", id], queryFn: () => api.network(id) });

  if (net.isLoading)
    return <div className="flex h-40 items-center justify-center"><Spinner /></div>;
  if (net.isError) return <p className="text-(--color-bad)">{(net.error as Error).message}</p>;
  if (!net.data?.length)
    return <p className="text-sm text-(--color-muted)">No network interfaces reported.</p>;

  return (
    <div className="space-y-4">
      {net.data.map((n) => (
        <Card key={n.instanceId || n.name}>
          <div className="mb-3 flex items-center gap-2">
            <h3 className="font-medium">{n.name || n.instanceId}</h3>
            <Badge tone={n.linkUp ? "good" : "muted"}>{n.linkUp ? "link up" : "link down"}</Badge>
            <Badge tone="muted">{n.dhcpEnabled ? "DHCP" : "Static"}</Badge>
            {n.sharedMac && <Badge tone="muted">shared MAC</Badge>}
          </div>
          <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm md:grid-cols-3">
            <KV k="IP Address" v={n.ipAddress} />
            <KV k="Subnet Mask" v={n.subnetMask} />
            <KV k="Gateway" v={n.defaultGateway} />
            <KV k="MAC" v={n.macAddress} />
            <KV k="Primary DNS" v={n.primaryDns} />
            <KV k="Secondary DNS" v={n.secondaryDns} />
          </div>
        </Card>
      ))}
    </div>
  );
}

function KV({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-xs text-(--color-muted)">{k}</span>
      <span className="font-mono">{v || "-"}</span>
    </div>
  );
}

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Wifi, Trash2, Plus, Pencil } from "lucide-react";
import { api, ApiError, type NetworkInterface } from "../../lib/api";
import { Badge, Button, Card, Field, Input, Spinner } from "../../lib/ui";

export function NetworkTab({ id }: { id: string }) {
  const net = useQuery({ queryKey: ["network", id], queryFn: () => api.network(id) });
  const info = useQuery({ queryKey: ["info", id], queryFn: () => api.info(id) });

  return (
    <div className="space-y-4">
      {info.data && (
        <Card>
          <h3 className="mb-3 font-medium">General</h3>
          <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm md:grid-cols-3">
            <KV k="Domain" v={info.data.domainName} />
            <KV k="Respond to ping" v={info.data.respondToPing ? "Yes" : "No"} />
            <KV k="Dynamic DNS" v={info.data.dynamicDns ? "Enabled" : "Disabled"} />
          </div>
        </Card>
      )}
      <h3 className="font-medium">Wired interfaces</h3>
      {net.isLoading && <div className="flex justify-center py-6"><Spinner /></div>}
      {net.isError && <p className="text-(--color-bad)">{(net.error as Error).message}</p>}
      {net.data?.length === 0 && (
        <p className="text-sm text-(--color-muted)">No wired interfaces reported.</p>
      )}
      {net.data?.map((n) => (
        <WiredInterface key={n.instanceId || n.name} id={id} n={n} />
      ))}

      <Wireless id={id} />
    </div>
  );
}

function WiredInterface({ id, n }: { id: string; n: NetworkInterface }) {
  const qc = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [dhcp, setDhcp] = useState(n.dhcpEnabled);
  const [form, setForm] = useState({
    ipAddress: n.ipAddress,
    subnetMask: n.subnetMask,
    defaultGateway: n.defaultGateway,
    primaryDns: n.primaryDns,
    secondaryDns: n.secondaryDns,
  });

  const save = useMutation({
    mutationFn: () => api.setNetwork(id, { instanceId: n.instanceId, dhcp, ...form }),
    onSuccess: () => {
      setEditing(false);
      qc.invalidateQueries({ queryKey: ["network", id] });
    },
  });

  return (
    <Card>
      <div className="mb-3 flex items-center gap-2">
        <h3 className="font-medium">{n.name || n.instanceId}</h3>
        <Badge tone={n.linkUp ? "good" : "muted"}>{n.linkUp ? "link up" : "link down"}</Badge>
        <Badge tone="muted">{n.dhcpEnabled ? "DHCP" : "Static"}</Badge>
        {n.sharedMac && <Badge tone="muted">shared MAC</Badge>}
        {!editing && (
          <Button variant="ghost" className="ml-auto px-1.5 py-1" onClick={() => setEditing(true)} title="Edit IP settings">
            <Pencil className="h-4 w-4" />
          </Button>
        )}
      </div>

      {!editing ? (
        <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm md:grid-cols-3">
          <KV k="IP Address" v={n.ipAddress} />
          <KV k="Subnet Mask" v={n.subnetMask} />
          <KV k="Gateway" v={n.defaultGateway} />
          <KV k="MAC" v={n.macAddress} />
          <KV k="Primary DNS" v={n.primaryDns} />
          <KV k="Secondary DNS" v={n.secondaryDns} />
        </div>
      ) : (
        <form
          className="space-y-3"
          onSubmit={(e) => {
            e.preventDefault();
            const msg = dhcp
              ? "Switch this interface to DHCP? The device's AMT address may change and could become unreachable on the current IP."
              : `Set a static AMT IP (${form.ipAddress})? A wrong value can make the device unreachable.`;
            if (confirm(msg)) save.mutate();
          }}
        >
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={dhcp} onChange={(e) => setDhcp(e.target.checked)} /> Use DHCP
          </label>
          {!dhcp && (
            <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
              <Field label="IP Address">
                <Input value={form.ipAddress} onChange={(e) => setForm({ ...form, ipAddress: e.target.value })} required />
              </Field>
              <Field label="Subnet Mask">
                <Input value={form.subnetMask} onChange={(e) => setForm({ ...form, subnetMask: e.target.value })} required />
              </Field>
              <Field label="Gateway">
                <Input value={form.defaultGateway} onChange={(e) => setForm({ ...form, defaultGateway: e.target.value })} />
              </Field>
              <Field label="Primary DNS">
                <Input value={form.primaryDns} onChange={(e) => setForm({ ...form, primaryDns: e.target.value })} />
              </Field>
              <Field label="Secondary DNS">
                <Input value={form.secondaryDns} onChange={(e) => setForm({ ...form, secondaryDns: e.target.value })} />
              </Field>
            </div>
          )}
          <div className="rounded-md bg-(--color-warn)/15 px-3 py-2 text-xs text-(--color-warn)">
            Changing the AMT IP can make this device unreachable at its current address. You may need to reconnect by its new IP.
          </div>
          {save.isError && (
            <div className="rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
              {save.error instanceof ApiError ? save.error.message : "Failed to apply"}
            </div>
          )}
          <div className="flex gap-2">
            <Button type="button" onClick={() => setEditing(false)}>Cancel</Button>
            <Button type="submit" variant="primary" disabled={save.isPending}>
              {save.isPending ? "Applying…" : "Apply"}
            </Button>
          </div>
        </form>
      )}
    </Card>
  );
}

function Wireless({ id }: { id: string }) {
  const qc = useQueryClient();
  const wifi = useQuery({ queryKey: ["wifi", id], queryFn: () => api.wifi(id) });
  const [ssid, setSsid] = useState("");
  const [passphrase, setPass] = useState("");

  const invalidate = () => qc.invalidateQueries({ queryKey: ["wifi", id] });
  const add = useMutation({
    mutationFn: () => api.addWifi(id, { ssid, passphrase, priority: 0 }),
    onSuccess: () => {
      setSsid("");
      setPass("");
      invalidate();
    },
  });
  const remove = useMutation({
    mutationFn: (instanceId: string) => api.deleteWifi(id, instanceId),
    onSuccess: invalidate,
  });

  return (
    <Card>
      <h3 className="mb-3 flex items-center gap-2 font-medium">
        <Wifi className="h-4 w-4 text-(--color-accent)" /> Wireless profiles
      </h3>
      {wifi.isLoading && <div className="flex justify-center py-4"><Spinner /></div>}
      {wifi.isError && <p className="text-sm text-(--color-muted)">Wireless not available on this device.</p>}
      {wifi.data?.length === 0 && (
        <p className="text-sm text-(--color-muted)">No wireless profiles.</p>
      )}
      {wifi.data && wifi.data.length > 0 && (
        <table className="mb-4 w-full text-left text-sm">
          <thead>
            <tr className="text-xs uppercase text-(--color-muted)">
              <th className="pb-2 pr-4 font-medium">SSID</th>
              <th className="pb-2 pr-4 font-medium">Auth</th>
              <th className="pb-2 pr-4 font-medium">Priority</th>
              <th className="pb-2 pr-4 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            {wifi.data.map((p) => (
              <tr key={p.instanceId} className="border-t border-(--color-border)">
                <td className="py-2 pr-4">{p.ssid || p.name}</td>
                <td className="py-2 pr-4 font-mono text-xs">{p.auth}</td>
                <td className="py-2 pr-4">{p.priority}</td>
                <td className="py-2 pr-4">
                  <Button
                    variant="ghost"
                    className="px-1.5 py-1 text-(--color-bad)"
                    onClick={() => {
                      if (confirm(`Remove wireless profile "${p.ssid || p.name}"?`)) remove.mutate(p.instanceId);
                    }}
                    title="Remove profile"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <form
        className="flex items-end gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          add.mutate();
        }}
      >
        <div className="flex-1">
          <Field label="SSID">
            <Input value={ssid} onChange={(e) => setSsid(e.target.value)} />
          </Field>
        </div>
        <div className="flex-1">
          <Field label="Passphrase (WPA2-PSK)">
            <Input type="password" value={passphrase} onChange={(e) => setPass(e.target.value)} />
          </Field>
        </div>
        <Button type="submit" variant="primary" disabled={!ssid || !passphrase || add.isPending}>
          <Plus className="h-4 w-4" /> {add.isPending ? "Adding…" : "Add"}
        </Button>
      </form>
      {add.isError && (
        <div className="mt-3 rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
          {add.error instanceof ApiError ? add.error.message : "Failed to add profile"}
        </div>
      )}
    </Card>
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

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Cloud, Trash2, Plus, RotateCcw } from "lucide-react";
import { api, ApiError } from "../../lib/api";
import { Button, Card, Field, Input, Spinner } from "../../lib/ui";

const TRIGGERS = [
  { value: 0, label: "User initiated" },
  { value: 1, label: "Alert" },
  { value: 2, label: "Periodic" },
];

export function RemoteAccessTab({ id }: { id: string }) {
  const qc = useQueryClient();
  const ra = useQuery({ queryKey: ["remoteaccess", id], queryFn: () => api.remoteAccess(id) });
  const invalidate = () => qc.invalidateQueries({ queryKey: ["remoteaccess", id] });

  const [mps, setMps] = useState({ accessInfo: "", port: 4433, username: "", password: "", commonName: "" });
  const [policyMps, setPolicyMps] = useState("");
  const [trigger, setTrigger] = useState(0);

  const addMps = useMutation({
    mutationFn: () => api.addMps(id, mps),
    onSuccess: () => {
      setMps({ accessInfo: "", port: 4433, username: "", password: "", commonName: "" });
      invalidate();
    },
  });
  const delMps = useMutation({ mutationFn: (n: string) => api.deleteMps(id, n), onSuccess: invalidate });
  const addPolicy = useMutation({
    mutationFn: () => api.addPolicy(id, { mpsName: policyMps, trigger, tunnelLifeSeconds: 0 }),
    onSuccess: invalidate,
  });
  const delPolicy = useMutation({ mutationFn: (n: string) => api.deletePolicy(id, n), onSuccess: invalidate });

  const servers = ra.data?.mpsServers ?? [];
  const policies = ra.data?.policies ?? [];

  return (
    <div className="max-w-3xl space-y-4">
      <div className="rounded-md bg-(--color-warn)/15 px-3 py-2 text-sm text-(--color-warn)">
        Client-Initiated Remote Access (CIRA) lets the device call home to a
        Management Presence Server so you can reach it outside the LAN. These are
        device config changes; verify against your MPS setup.
      </div>

      {ra.data && (
        <Card>
          <div className="flex items-center justify-between text-sm">
            <span className="text-(--color-muted)">Environment detection</span>
            <span className="font-mono">{ra.data.environmentDetection || "-"}</span>
          </div>
        </Card>
      )}

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="flex items-center gap-2 font-medium">
            <Cloud className="h-5 w-5 text-(--color-accent)" /> MPS Servers
          </h3>
          <Button onClick={() => ra.refetch()}>
            <RotateCcw className="h-4 w-4" /> Refresh
          </Button>
        </div>
        {ra.isLoading && <div className="flex justify-center py-4"><Spinner /></div>}
        {ra.isError && <p className="text-(--color-bad)">{(ra.error as Error).message}</p>}
        {ra.data && servers.length === 0 && <p className="text-sm text-(--color-muted)">No MPS servers.</p>}
        {servers.length > 0 && (
          <table className="mb-4 w-full text-left text-sm">
            <thead>
              <tr className="text-xs uppercase text-(--color-muted)">
                <th className="pb-2 pr-4 font-medium">Server</th>
                <th className="pb-2 pr-4 font-medium">Port</th>
                <th className="pb-2 pr-4 font-medium">Common name</th>
                <th className="pb-2 pr-4 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {servers.map((m) => (
                <tr key={m.name} className="border-t border-(--color-border)">
                  <td className="py-2 pr-4 font-mono text-xs">{m.accessInfo}</td>
                  <td className="py-2 pr-4">{m.port}</td>
                  <td className="py-2 pr-4 font-mono text-xs">{m.commonName}</td>
                  <td className="py-2 pr-4">
                    <Button
                      variant="ghost"
                      className="px-1.5 py-1 text-(--color-bad)"
                      onClick={() => confirm(`Remove MPS "${m.accessInfo}"?`) && delMps.mutate(m.name)}
                      title="Remove MPS"
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
          className="grid grid-cols-2 gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            addMps.mutate();
          }}
        >
          <Field label="Server (FQDN or IP)">
            <Input value={mps.accessInfo} onChange={(e) => setMps({ ...mps, accessInfo: e.target.value })} />
          </Field>
          <Field label="Port">
            <Input type="number" value={mps.port} onChange={(e) => setMps({ ...mps, port: Number(e.target.value) })} />
          </Field>
          <Field label="Username">
            <Input value={mps.username} onChange={(e) => setMps({ ...mps, username: e.target.value })} />
          </Field>
          <Field label="Password">
            <Input type="password" value={mps.password} onChange={(e) => setMps({ ...mps, password: e.target.value })} />
          </Field>
          <Field label="Common name (optional)">
            <Input value={mps.commonName} onChange={(e) => setMps({ ...mps, commonName: e.target.value })} />
          </Field>
          <div className="flex items-end">
            <Button type="submit" variant="primary" disabled={!mps.accessInfo || !mps.username || !mps.password || addMps.isPending}>
              <Plus className="h-4 w-4" /> Add MPS
            </Button>
          </div>
          {addMps.isError && (
            <div className="col-span-2 rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
              {addMps.error instanceof ApiError ? addMps.error.message : "Failed to add MPS"}
            </div>
          )}
        </form>
      </Card>

      <Card>
        <h3 className="mb-3 font-medium">CIRA Policies</h3>
        {ra.data && policies.length === 0 && <p className="text-sm text-(--color-muted)">No policies.</p>}
        {policies.length > 0 && (
          <table className="mb-4 w-full text-left text-sm">
            <thead>
              <tr className="text-xs uppercase text-(--color-muted)">
                <th className="pb-2 pr-4 font-medium">Policy</th>
                <th className="pb-2 pr-4 font-medium">Trigger</th>
                <th className="pb-2 pr-4 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {policies.map((p) => (
                <tr key={p.name} className="border-t border-(--color-border)">
                  <td className="py-2 pr-4">{p.name}</td>
                  <td className="py-2 pr-4">{p.trigger}</td>
                  <td className="py-2 pr-4">
                    <Button
                      variant="ghost"
                      className="px-1.5 py-1 text-(--color-bad)"
                      onClick={() => confirm(`Remove policy "${p.name}"?`) && delPolicy.mutate(p.name)}
                      title="Remove policy"
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
            addPolicy.mutate();
          }}
        >
          <div className="flex-1">
            <Field label="MPS server">
              <select
                value={policyMps}
                onChange={(e) => setPolicyMps(e.target.value)}
                className="rounded-md border border-(--color-border) bg-(--color-bg) px-3 py-1.5 text-sm outline-none focus:border-(--color-accent)"
              >
                <option value="">Select…</option>
                {servers.map((m) => (
                  <option key={m.name} value={m.name}>{m.accessInfo}</option>
                ))}
              </select>
            </Field>
          </div>
          <Field label="Trigger">
            <select
              value={trigger}
              onChange={(e) => setTrigger(Number(e.target.value))}
              className="rounded-md border border-(--color-border) bg-(--color-bg) px-3 py-1.5 text-sm outline-none focus:border-(--color-accent)"
            >
              {TRIGGERS.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
          </Field>
          <Button type="submit" variant="primary" disabled={!policyMps || addPolicy.isPending}>
            <Plus className="h-4 w-4" /> Add policy
          </Button>
        </form>
        {addPolicy.isError && (
          <div className="mt-3 rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
            {addPolicy.error instanceof ApiError ? addPolicy.error.message : "Failed to add policy"}
          </div>
        )}
      </Card>
    </div>
  );
}

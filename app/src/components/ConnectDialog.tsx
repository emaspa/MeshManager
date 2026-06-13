import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import { api, ApiError, type ConnectParams } from "../lib/api";
import { useUi } from "../store";
import { Button, Field, Input } from "../lib/ui";

export function ConnectDialog() {
  const setConnectOpen = useUi((s) => s.setConnectOpen);
  const select = useUi((s) => s.select);
  const qc = useQueryClient();

  const [form, setForm] = useState<ConnectParams>({
    host: "",
    username: "admin",
    password: "",
    tls: false,
    insecure: true,
    name: "",
  });

  const connect = useMutation({
    mutationFn: () => api.connect(form),
    onSuccess: (device) => {
      qc.invalidateQueries({ queryKey: ["devices"] });
      select(device.id);
      setConnectOpen(false);
    },
  });

  const set = <K extends keyof ConnectParams>(k: K, v: ConnectParams[K]) =>
    setForm((f) => ({ ...f, [k]: v }));

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-[420px] rounded-xl border border-[--color-border] bg-[--color-panel] p-5 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">Add AMT Device</h2>
          <Button variant="ghost" className="px-1.5 py-1" onClick={() => setConnectOpen(false)}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            connect.mutate();
          }}
        >
          <Field label="Name (optional)">
            <Input value={form.name} onChange={(e) => set("name", e.target.value)} placeholder="Lab PC #1" />
          </Field>
          <div className="grid grid-cols-[1fr_110px] gap-3">
            <Field label="Host / IP">
              <Input
                required
                value={form.host}
                onChange={(e) => set("host", e.target.value)}
                placeholder="192.168.1.50"
              />
            </Field>
            <Field label="Port">
              <Input
                type="number"
                value={form.port ?? ""}
                onChange={(e) => set("port", e.target.value ? Number(e.target.value) : undefined)}
                placeholder={form.tls ? "16993" : "16992"}
              />
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Username">
              <Input value={form.username} onChange={(e) => set("username", e.target.value)} />
            </Field>
            <Field label="Password">
              <Input
                type="password"
                value={form.password}
                onChange={(e) => set("password", e.target.value)}
              />
            </Field>
          </div>
          <div className="flex gap-4 text-sm">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={form.tls}
                onChange={(e) => set("tls", e.target.checked)}
              />
              Use TLS
            </label>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={form.insecure}
                onChange={(e) => set("insecure", e.target.checked)}
              />
              Allow self-signed
            </label>
          </div>

          {connect.isError && (
            <div className="rounded-md bg-[--color-bad]/15 px-3 py-2 text-sm text-[--color-bad]">
              {connect.error instanceof ApiError
                ? connect.error.message
                : "Connection failed"}
            </div>
          )}

          <div className="mt-2 flex justify-end gap-2">
            <Button type="button" onClick={() => setConnectOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" disabled={connect.isPending}>
              {connect.isPending ? "Connecting…" : "Connect"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

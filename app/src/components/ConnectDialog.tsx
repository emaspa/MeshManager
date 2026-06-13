import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import { api, ApiError, type ConnectParams } from "../lib/api";
import { useUi } from "../store";
import { useBookmarks } from "../lib/bookmarks";
import { Button, Field, Input } from "../lib/ui";

export function ConnectDialog() {
  const closeConnect = useUi((s) => s.closeConnect);
  const prefill = useUi((s) => s.connectPrefill);
  const select = useUi((s) => s.select);
  const bookmarks = useBookmarks();
  const qc = useQueryClient();

  const editing = !!prefill?.edit;

  const [form, setForm] = useState({
    name: prefill?.name ?? "",
    group: prefill?.group ?? "",
    host: prefill?.host ?? "",
    port: prefill?.port as number | undefined,
    username: prefill?.username ?? "admin",
    password: prefill?.password ?? "",
    tls: prefill?.tls ?? false,
    insecure: prefill?.insecure ?? true,
  });
  const [remember, setRemember] = useState(!!prefill?.password);

  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) =>
    setForm((f) => ({ ...f, [k]: v }));

  function saveBookmark() {
    bookmarks.upsert({
      name: form.name,
      group: form.group,
      host: form.host,
      port: form.port,
      tls: form.tls,
      insecure: form.insecure,
      username: form.username,
      password: remember ? form.password : undefined,
    });
  }

  // Edit mode just persists the bookmark (no connect).
  function onSave() {
    if (prefill?.bookmarkId) {
      bookmarks.update(prefill.bookmarkId, {
        name: form.name,
        group: form.group,
        host: form.host,
        port: form.port,
        tls: form.tls,
        insecure: form.insecure,
        username: form.username,
        password: remember ? form.password : undefined,
      });
    } else {
      saveBookmark();
    }
    closeConnect();
  }

  const connect = useMutation({
    mutationFn: () => api.connect(form as ConnectParams),
    onSuccess: (device) => {
      saveBookmark(); // persist the connection as a bookmark
      qc.invalidateQueries({ queryKey: ["devices"] });
      select(device.id);
      closeConnect();
    },
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-[420px] rounded-xl border border-(--color-border) bg-(--color-panel) p-5 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">{editing ? "Edit Bookmark" : "Add Device"}</h2>
          <Button variant="ghost" className="px-1.5 py-1" onClick={() => closeConnect()}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            if (editing) onSave();
            else connect.mutate();
          }}
        >
          <div className="grid grid-cols-2 gap-3">
            <Field label="Name (optional)">
              <Input value={form.name} onChange={(e) => set("name", e.target.value)} placeholder="Lab PC #1" />
            </Field>
            <Field label="Group (optional)">
              <Input value={form.group} onChange={(e) => set("group", e.target.value)} placeholder="Lab" />
            </Field>
          </div>
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
          <div className="flex flex-wrap gap-4 text-sm">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={form.tls} onChange={(e) => set("tls", e.target.checked)} />
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
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={remember} onChange={(e) => setRemember(e.target.checked)} />
              Remember password
            </label>
          </div>

          {connect.isError && (
            <div className="rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
              {connect.error instanceof ApiError ? connect.error.message : "Connection failed"}
            </div>
          )}

          <div className="mt-2 flex justify-end gap-2">
            <Button type="button" onClick={() => closeConnect()}>
              Cancel
            </Button>
            {editing ? (
              <Button type="submit" variant="primary">Save</Button>
            ) : (
              <Button type="submit" variant="primary" disabled={connect.isPending}>
                {connect.isPending ? "Connecting…" : "Connect"}
              </Button>
            )}
          </div>
        </form>
      </div>
    </div>
  );
}

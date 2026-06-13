import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { UserPlus, Trash2, RotateCcw } from "lucide-react";
import { api, ApiError } from "../../lib/api";
import { Badge, Button, Card, Field, Input, Spinner } from "../../lib/ui";

// AMT realm numbers (from AMT_AuthorizationService) with friendly labels.
const REALMS: { value: number; label: string }[] = [
  { value: 2, label: "Redirection (SOL/IDE-R)" },
  { value: 3, label: "PT Administration" },
  { value: 4, label: "Hardware Asset" },
  { value: 5, label: "Remote Control (power)" },
  { value: 6, label: "Storage" },
  { value: 7, label: "Event Manager" },
  { value: 8, label: "Storage Admin" },
  { value: 12, label: "Network Time" },
  { value: 13, label: "General Info" },
  { value: 16, label: "KVM (User)" },
  { value: 19, label: "Event Log Reader" },
  { value: 20, label: "Audit Log" },
  { value: 21, label: "ACL (account mgmt)" },
];
const REALM_LABEL = new Map(REALMS.map((r) => [r.value, r.label]));
const DEFAULT_REALMS = [2, 5, 16]; // Redirection + Remote Control + KVM: a useful operator

const ACCESS = [
  { value: 2, label: "Any (local + network)" },
  { value: 1, label: "Network only" },
  { value: 0, label: "Local only" },
];

export function AccountsTab({ id }: { id: string }) {
  const qc = useQueryClient();
  const accounts = useQuery({ queryKey: ["accounts", id], queryFn: () => api.accounts(id) });

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [accessPermission, setAccess] = useState(2);
  const [realms, setRealms] = useState<number[]>(DEFAULT_REALMS);

  const invalidate = () => qc.invalidateQueries({ queryKey: ["accounts", id] });

  const add = useMutation({
    mutationFn: () => api.addAccount(id, { username, password, accessPermission, realms }),
    onSuccess: () => {
      setUsername("");
      setPassword("");
      setRealms(DEFAULT_REALMS);
      invalidate();
    },
  });
  const remove = useMutation({
    mutationFn: (handle: number) => api.removeAccount(id, handle),
    onSuccess: invalidate,
  });
  const toggle = useMutation({
    mutationFn: (v: { handle: number; enabled: boolean }) =>
      api.setAccountEnabled(id, v.handle, v.enabled),
    onSuccess: invalidate,
  });

  const toggleRealm = (v: number) =>
    setRealms((rs) => (rs.includes(v) ? rs.filter((r) => r !== v) : [...rs, v]));

  return (
    <div className="space-y-4">
      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-medium">User Accounts</h3>
          <Button onClick={() => accounts.refetch()}>
            <RotateCcw className="h-4 w-4" /> Refresh
          </Button>
        </div>
        {accounts.isLoading && <div className="flex justify-center py-6"><Spinner /></div>}
        {accounts.isError && (
          <p className="text-(--color-bad)">{(accounts.error as Error).message}</p>
        )}
        {accounts.data?.length === 0 && (
          <p className="text-sm text-(--color-muted)">No user accounts.</p>
        )}
        {accounts.data && accounts.data.length > 0 && (
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="text-xs uppercase text-(--color-muted)">
                <th className="pb-2 pr-4 font-medium">User</th>
                <th className="pb-2 pr-4 font-medium">Access</th>
                <th className="pb-2 pr-4 font-medium">Realms</th>
                <th className="pb-2 pr-4 font-medium">State</th>
                <th className="pb-2 pr-4 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {accounts.data.map((a) => (
                <tr key={a.handle} className="border-t border-(--color-border) align-top">
                  <td className="py-2 pr-4 font-mono">{a.username || `#${a.handle}`}</td>
                  <td className="py-2 pr-4">
                    {ACCESS.find((x) => x.value === a.accessPermission)?.label ?? a.accessPermission}
                  </td>
                  <td className="py-2 pr-4">
                    <div className="flex flex-wrap gap-1">
                      {a.realms.map((r) => (
                        <Badge key={r} tone="muted">{REALM_LABEL.get(r) ?? `realm ${r}`}</Badge>
                      ))}
                    </div>
                  </td>
                  <td className="py-2 pr-4">
                    <button
                      onClick={() => toggle.mutate({ handle: a.handle, enabled: !a.enabled })}
                      className="underline decoration-dotted underline-offset-2"
                    >
                      <Badge tone={a.enabled ? "good" : "muted"}>
                        {a.enabled ? "enabled" : "disabled"}
                      </Badge>
                    </button>
                  </td>
                  <td className="py-2 pr-4">
                    <Button
                      variant="ghost"
                      className="px-1.5 py-1 text-(--color-bad)"
                      onClick={() => remove.mutate(a.handle)}
                      title="Delete user"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>

      <Card>
        <h3 className="mb-3 flex items-center gap-2 font-medium">
          <UserPlus className="h-4 w-4 text-(--color-accent)" /> Add Digest User
        </h3>
        <form
          className="space-y-3"
          onSubmit={(e) => {
            e.preventDefault();
            add.mutate();
          }}
        >
          <div className="grid grid-cols-3 gap-3">
            <Field label="Username (≤16 chars)">
              <Input value={username} maxLength={16} onChange={(e) => setUsername(e.target.value)} />
            </Field>
            <Field label="Password">
              <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
            </Field>
            <Field label="Access">
              <select
                value={accessPermission}
                onChange={(e) => setAccess(Number(e.target.value))}
                className="rounded-md border border-(--color-border) bg-(--color-bg) px-3 py-1.5 text-sm outline-none focus:border-(--color-accent)"
              >
                {ACCESS.map((a) => (
                  <option key={a.value} value={a.value}>{a.label}</option>
                ))}
              </select>
            </Field>
          </div>
          <div>
            <div className="mb-1 text-sm text-(--color-muted)">Realms</div>
            <div className="grid grid-cols-3 gap-x-4 gap-y-1">
              {REALMS.map((r) => (
                <label key={r.value} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={realms.includes(r.value)}
                    onChange={() => toggleRealm(r.value)}
                  />
                  {r.label}
                </label>
              ))}
            </div>
          </div>
          {add.isError && (
            <div className="rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
              {add.error instanceof ApiError ? add.error.message : "Failed to add user"}
            </div>
          )}
          <Button
            type="submit"
            variant="primary"
            disabled={!username || !password || realms.length === 0 || add.isPending}
          >
            {add.isPending ? "Adding…" : "Add User"}
          </Button>
        </form>
      </Card>
    </div>
  );
}

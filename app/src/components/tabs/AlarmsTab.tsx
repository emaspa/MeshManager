import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { AlarmClock, Trash2, RotateCcw, Plus } from "lucide-react";
import { api, ApiError } from "../../lib/api";
import { Button, Card, Field, Input, Spinner } from "../../lib/ui";

const RECURRENCE = [
  { label: "One time", minutes: 0 },
  { label: "Hourly", minutes: 60 },
  { label: "Daily", minutes: 1440 },
  { label: "Weekly", minutes: 10080 },
];

export function AlarmsTab({ id }: { id: string }) {
  const qc = useQueryClient();
  const alarms = useQuery({ queryKey: ["alarms", id], queryFn: () => api.alarms(id) });

  const [name, setName] = useState("");
  const [when, setWhen] = useState("");
  const [intervalMinutes, setInterval] = useState(0);
  const [deleteOnCompletion, setDeleteOnCompletion] = useState(true);

  const invalidate = () => qc.invalidateQueries({ queryKey: ["alarms", id] });

  const add = useMutation({
    mutationFn: () =>
      api.addAlarm(id, {
        name,
        startTime: new Date(when).toISOString(), // local -> RFC3339 (UTC)
        intervalMinutes,
        deleteOnCompletion,
      }),
    onSuccess: () => {
      setName("");
      setWhen("");
      invalidate();
    },
  });
  const remove = useMutation({
    mutationFn: (instanceId: string) => api.deleteAlarm(id, instanceId),
    onSuccess: invalidate,
  });

  return (
    <div className="max-w-3xl space-y-4">
      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="flex items-center gap-2 font-medium">
            <AlarmClock className="h-5 w-5 text-(--color-accent)" /> Scheduled Wake-ups
          </h3>
          <Button onClick={() => alarms.refetch()}>
            <RotateCcw className="h-4 w-4" /> Refresh
          </Button>
        </div>
        {alarms.isLoading && <div className="flex justify-center py-6"><Spinner /></div>}
        {alarms.isError && <p className="text-(--color-bad)">{(alarms.error as Error).message}</p>}
        {alarms.data?.length === 0 && (
          <p className="text-sm text-(--color-muted)">No scheduled wake-ups.</p>
        )}
        {alarms.data && alarms.data.length > 0 && (
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="text-xs uppercase text-(--color-muted)">
                <th className="pb-2 pr-4 font-medium">Name</th>
                <th className="pb-2 pr-4 font-medium">Next wake</th>
                <th className="pb-2 pr-4 font-medium">Repeat</th>
                <th className="pb-2 pr-4 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {alarms.data.map((a) => (
                <tr key={a.instanceId} className="border-t border-(--color-border)">
                  <td className="py-2 pr-4">{a.name || a.instanceId}</td>
                  <td className="py-2 pr-4 font-mono text-xs">
                    {a.startTime ? new Date(a.startTime).toLocaleString() : "-"}
                  </td>
                  <td className="py-2 pr-4 font-mono text-xs">
                    {a.interval && a.interval !== "P0DT0H0M" ? a.interval : "once"}
                  </td>
                  <td className="py-2 pr-4">
                    <Button
                      variant="ghost"
                      className="px-1.5 py-1 text-(--color-bad)"
                      onClick={() => remove.mutate(a.instanceId)}
                      title="Delete"
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
          <Plus className="h-4 w-4 text-(--color-accent)" /> Schedule a Wake-up
        </h3>
        <form
          className="space-y-3"
          onSubmit={(e) => {
            e.preventDefault();
            add.mutate();
          }}
        >
          <div className="grid grid-cols-3 gap-3">
            <Field label="Name (optional)">
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Nightly patch" />
            </Field>
            <Field label="Wake at">
              <Input type="datetime-local" value={when} onChange={(e) => setWhen(e.target.value)} required />
            </Field>
            <Field label="Repeat">
              <select
                value={intervalMinutes}
                onChange={(e) => setInterval(Number(e.target.value))}
                className="rounded-md border border-(--color-border) bg-(--color-bg) px-3 py-1.5 text-sm outline-none focus:border-(--color-accent)"
              >
                {RECURRENCE.map((r) => (
                  <option key={r.minutes} value={r.minutes}>{r.label}</option>
                ))}
              </select>
            </Field>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={deleteOnCompletion}
              onChange={(e) => setDeleteOnCompletion(e.target.checked)}
            />
            Delete after it fires (one-time alarms)
          </label>
          {add.isError && (
            <div className="rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
              {add.error instanceof ApiError ? add.error.message : "Failed to add alarm"}
            </div>
          )}
          <Button type="submit" variant="primary" disabled={!when || add.isPending}>
            {add.isPending ? "Scheduling…" : "Schedule"}
          </Button>
        </form>
      </Card>
    </div>
  );
}

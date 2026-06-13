import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { RotateCcw, Trash2, Plus } from "lucide-react";
import { api, ApiError } from "../../lib/api";
import { Badge, Button, Card, Field, Spinner } from "../../lib/ui";

export function CertificatesTab({ id }: { id: string }) {
  const qc = useQueryClient();
  const certs = useQuery({ queryKey: ["certs", id], queryFn: () => api.certificates(id) });
  const [pem, setPem] = useState("");

  const invalidate = () => qc.invalidateQueries({ queryKey: ["certs", id] });

  const add = useMutation({
    mutationFn: () => api.addTrustedRoot(id, pem),
    onSuccess: () => {
      setPem("");
      invalidate();
    },
  });
  const remove = useMutation({
    mutationFn: (instanceId: string) => api.deleteCertificate(id, instanceId),
    onSuccess: invalidate,
  });

  return (
    <div className="space-y-4">
      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-medium">Certificates</h3>
          <Button onClick={() => certs.refetch()}>
            <RotateCcw className="h-4 w-4" /> Refresh
          </Button>
        </div>
        {certs.isLoading && <div className="flex justify-center py-6"><Spinner /></div>}
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
                <th className="pb-2 pr-4 font-medium"></th>
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
                  <td className="py-2 pr-4">
                    <Button
                      variant="ghost"
                      className="px-1.5 py-1 text-(--color-bad)"
                      onClick={() => {
                        if (confirm("Delete this certificate from the device? This cannot be undone.")) {
                          remove.mutate(c.instanceId);
                        }
                      }}
                      title="Delete certificate"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {remove.isError && (
          <div className="mt-3 rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
            {remove.error instanceof ApiError ? remove.error.message : "Delete failed"}
          </div>
        )}
      </Card>

      <Card>
        <h3 className="mb-3 flex items-center gap-2 font-medium">
          <Plus className="h-4 w-4 text-(--color-accent)" /> Add Trusted Root Certificate
        </h3>
        <form
          className="space-y-3"
          onSubmit={(e) => {
            e.preventDefault();
            add.mutate();
          }}
        >
          <Field label="Certificate (PEM or base64 DER)">
            <textarea
              value={pem}
              onChange={(e) => setPem(e.target.value)}
              rows={6}
              placeholder={"-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----"}
              className="rounded-md border border-(--color-border) bg-(--color-bg) px-3 py-2 font-mono text-xs outline-none focus:border-(--color-accent)"
            />
          </Field>
          {add.isError && (
            <div className="rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
              {add.error instanceof ApiError ? add.error.message : "Failed to add certificate"}
            </div>
          )}
          <Button type="submit" variant="primary" disabled={!pem.trim() || add.isPending}>
            {add.isPending ? "Adding…" : "Add Trusted Root"}
          </Button>
        </form>
      </Card>
    </div>
  );
}

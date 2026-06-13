import { MonitorSmartphone } from "lucide-react";

export function KvmTab({ id: _id }: { id: string }) {
  return (
    <div className="flex h-full flex-col items-center justify-center text-center text-[--color-muted]">
      <MonitorSmartphone className="h-10 w-10" />
      <h3 className="mt-3 font-medium text-[--color-text]">Remote Desktop (KVM)</h3>
      <p className="mt-1 max-w-sm text-sm">
        KVM renders the device's framebuffer via Intel AMT's RFB-based redirection. The canvas
        viewer and input forwarding are the next milestone.
      </p>
    </div>
  );
}

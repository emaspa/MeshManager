import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Power,
  RotateCcw,
  Unplug,
  ChevronDown,
  Cpu,
  ScrollText,
  ShieldCheck,
  TerminalSquare,
  MonitorSmartphone,
  LayoutDashboard,
  Network,
  Disc,
  Users,
  Braces,
  AlarmClock,
  FileKey,
  Cloud,
} from "lucide-react";
import { api, type PowerAction } from "../lib/api";
import { useUi } from "../store";
import { Badge, Button } from "../lib/ui";
import { Overview } from "./tabs/Overview";
import { HardwareTab } from "./tabs/HardwareTab";
import { NetworkTab } from "./tabs/NetworkTab";
import { EventLogTab } from "./tabs/EventLogTab";
import { AuditLogTab } from "./tabs/AuditLogTab";
import { SerialTab } from "./tabs/SerialTab";
import { KvmTab } from "./tabs/KvmTab";
import { IderTab } from "./tabs/IderTab";
import { AccountsTab } from "./tabs/AccountsTab";
import { WsmanTab } from "./tabs/WsmanTab";
import { AlarmsTab } from "./tabs/AlarmsTab";
import { CertificatesTab } from "./tabs/CertificatesTab";
import { RemoteAccessTab } from "./tabs/RemoteAccessTab";

const TABS = [
  { id: "overview", label: "Overview", icon: LayoutDashboard },
  { id: "hardware", label: "Hardware", icon: Cpu },
  { id: "network", label: "Network", icon: Network },
  { id: "cira", label: "Remote Access", icon: Cloud },
  { id: "accounts", label: "Accounts", icon: Users },
  { id: "certs", label: "Certificates", icon: FileKey },
  { id: "alarms", label: "Wake", icon: AlarmClock },
  { id: "events", label: "Event Log", icon: ScrollText },
  { id: "audit", label: "Audit Log", icon: ShieldCheck },
  { id: "serial", label: "Serial", icon: TerminalSquare },
  { id: "kvm", label: "Remote Desktop", icon: MonitorSmartphone },
  { id: "ider", label: "Boot Media", icon: Disc },
  { id: "wsman", label: "WS-MAN", icon: Braces },
] as const;

type TabId = (typeof TABS)[number]["id"];

const POWER_MENU: { action: PowerAction; label: string }[] = [
  { action: "on", label: "Power On" },
  { action: "off", label: "Power Off" },
  { action: "off-graceful", label: "Power Off (graceful)" },
  { action: "reset", label: "Reset" },
  { action: "cycle", label: "Power Cycle" },
  { action: "sleep", label: "Sleep" },
  { action: "hibernate", label: "Hibernate" },
];

const BOOT_MENU: { device: string; label: string }[] = [
  { device: "pxe", label: "Reset → PXE / Network" },
  { device: "cd", label: "Reset → CD / DVD" },
  { device: "hdd", label: "Reset → Hard Disk" },
  { device: "bios", label: "Reset → BIOS Setup" },
];

export function DeviceView({ id }: { id: string }) {
  const [tab, setTab] = useState<TabId>("overview");
  const [menuOpen, setMenuOpen] = useState(false);
  const select = useUi((s) => s.select);
  const qc = useQueryClient();

  const devices = useQuery({ queryKey: ["devices"], queryFn: api.listDevices });
  const device = devices.data?.find((d) => d.id === id);
  const power = useQuery({
    queryKey: ["power", id],
    queryFn: () => api.powerState(id),
    refetchInterval: 8000,
  });

  const powerAction = useMutation({
    mutationFn: (action: PowerAction) => api.power(id, action),
    onSettled: () => {
      setMenuOpen(false);
      setTimeout(() => qc.invalidateQueries({ queryKey: ["power", id] }), 1500);
    },
  });

  const bootAction = useMutation({
    mutationFn: (device: string) => api.boot(id, device, "reset"),
    onSettled: () => {
      setMenuOpen(false);
      setTimeout(() => qc.invalidateQueries({ queryKey: ["power", id] }), 1500);
    },
  });

  const disconnect = useMutation({
    mutationFn: () => api.disconnect(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["devices"] });
      select(null);
    },
  });

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <header className="flex items-center gap-3 border-b border-(--color-border) px-5 py-3">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-lg font-semibold">{device?.name || device?.host}</h1>
            {power.data &&
              (power.data.on ? (
                <Badge tone="good">{power.data.stateName}</Badge>
              ) : (
                <Badge tone="bad">{power.data.stateName}</Badge>
              ))}
          </div>
          <div className="text-xs text-(--color-muted)">
            {device?.host}:{device?.port} · {device?.username}
          </div>
        </div>

        <div className="relative ml-auto">
          <Button variant="primary" onClick={() => setMenuOpen((o) => !o)}>
            <Power className="h-4 w-4" />
            Power
            <ChevronDown className="h-3.5 w-3.5" />
          </Button>
          {menuOpen && (
            <div className="absolute right-0 z-20 mt-1 w-52 rounded-md border border-(--color-border) bg-(--color-panel-2) py-1 shadow-xl">
              {POWER_MENU.map((m) => (
                <button
                  key={m.action}
                  onClick={() => powerAction.mutate(m.action)}
                  disabled={powerAction.isPending}
                  className="flex w-full items-center px-3 py-1.5 text-left text-sm hover:bg-(--color-border) disabled:opacity-40"
                >
                  {m.label}
                </button>
              ))}
              <div className="my-1 border-t border-(--color-border)" />
              <div className="px-3 py-1 text-xs uppercase tracking-wide text-(--color-muted)">
                Boot to
              </div>
              {BOOT_MENU.map((m) => (
                <button
                  key={m.device}
                  onClick={() => bootAction.mutate(m.device)}
                  disabled={bootAction.isPending}
                  className="flex w-full items-center px-3 py-1.5 text-left text-sm hover:bg-(--color-border) disabled:opacity-40"
                >
                  {m.label}
                </button>
              ))}
            </div>
          )}
        </div>

        <Button onClick={() => power.refetch()} title="Refresh power state">
          <RotateCcw className="h-4 w-4" />
        </Button>
        <Button onClick={() => disconnect.mutate()} title="Disconnect">
          <Unplug className="h-4 w-4" /> Disconnect
        </Button>
      </header>

      {/* Tabs */}
      <nav className="flex gap-1 border-b border-(--color-border) px-3">
        {TABS.map((t) => {
          const Icon = t.icon;
          return (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={
                "flex items-center gap-2 border-b-2 px-3 py-2.5 text-sm transition-colors " +
                (tab === t.id
                  ? "border-(--color-accent) text-(--color-text)"
                  : "border-transparent text-(--color-muted) hover:text-(--color-text)")
              }
            >
              <Icon className="h-4 w-4" />
              {t.label}
            </button>
          );
        })}
      </nav>

      {/* Body */}
      <div className="flex-1 overflow-y-auto p-5">
        {tab === "overview" && <Overview id={id} />}
        {tab === "hardware" && <HardwareTab id={id} />}
        {tab === "network" && <NetworkTab id={id} />}
        {tab === "cira" && <RemoteAccessTab id={id} />}
        {tab === "accounts" && <AccountsTab id={id} />}
        {tab === "certs" && <CertificatesTab id={id} />}
        {tab === "alarms" && <AlarmsTab id={id} />}
        {tab === "events" && <EventLogTab id={id} />}
        {tab === "audit" && <AuditLogTab id={id} />}
        {tab === "serial" && <SerialTab id={id} />}
        {tab === "kvm" && <KvmTab id={id} />}
        {tab === "ider" && <IderTab id={id} />}
        {tab === "wsman" && <WsmanTab id={id} />}
      </div>
    </div>
  );
}

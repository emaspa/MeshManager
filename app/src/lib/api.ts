import { sidecar } from "./sidecar";
import { clientLog } from "./logger";

// --- Types mirrored from the Go sidecar JSON responses ---

export interface Device {
  id: string;
  host: string;
  port: number;
  name: string;
  tls: boolean;
  username: string;
  createdAt: string;
}

export interface ConnectParams {
  host: string;
  port?: number;
  username: string;
  password: string;
  tls?: boolean;
  insecure?: boolean;
  name?: string;
}

export interface DeviceInfo {
  uuid: string;
  hostname: string;
  domainName: string;
  digestRealm: string;
  networkEnabled: boolean;
  versions: Record<string, string>;
  provisioningState: string;
  controlMode: string;
}

export interface PowerStatus {
  state: number;
  stateName: string;
  on: boolean;
}

export interface Hardware {
  system: {
    manufacturer: string;
    model: string;
    serialNumber: string;
    version: string;
    chassisType: number;
  };
  processors: {
    id: string;
    family: number;
    maxClockMhz: number;
    currentClockMhz: number;
    upgradeMethod: number;
  }[];
  memory: {
    bankLabel: string;
    capacityMb: number;
    speedMhz: number;
    memoryType: number;
    manufacturer: string;
    partNumber: string;
    serialNumber: string;
  }[];
  disks: { deviceId: string; maxMediaKb: number; elementName: string }[];
}

export interface EventLogEntry {
  time: string;
  description: string;
  entity: string;
  severity: string;
}

export interface AuditLogEntry {
  time: string;
  app: string;
  event: string;
  initiator: string;
  netAddress: string;
  extended: string;
}

export interface NetworkInterface {
  name: string;
  instanceId: string;
  macAddress: string;
  linkUp: boolean;
  dhcpEnabled: boolean;
  ipAddress: string;
  subnetMask: string;
  defaultGateway: string;
  primaryDns: string;
  secondaryDns: string;
  sharedMac: boolean;
}

export interface Discovered {
  host: string;
  port: number;
  tls: boolean;
  server: string;
  isAmt: boolean;
}

export interface Account {
  handle: number;
  username: string;
  accessPermission: number; // 0=local, 1=network, 2=any
  realms: number[];
  enabled: boolean;
}

export interface Certificate {
  instanceId: string;
  name: string;
  subject: string;
  issuer: string;
  trustedRoot: boolean;
}

export interface Alarm {
  instanceId: string;
  name: string;
  startTime: string;
  interval: string;
  deleteOnCompletion: boolean;
}

export interface IderStats {
  connected: boolean;
  bytesToAmt: number;
  sectorsRead: number;
  isoSize: number;
  error: string;
}

export type PowerAction =
  | "on"
  | "off"
  | "off-graceful"
  | "reset"
  | "reset-graceful"
  | "cycle"
  | "sleep"
  | "hibernate"
  | "nmi";

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const { baseUrl, token } = await sidecar();
  const headers = new Headers(init?.headers);
  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (init?.body) headers.set("Content-Type", "application/json");

  let res: Response;
  try {
    res = await fetch(`${baseUrl}${path}`, { ...init, headers });
  } catch (e) {
    // Network-level failure the sidecar never sees - capture it client-side.
    void clientLog("error", `request to ${path} failed: ${String(e)}`, "api.fetch");
    throw e;
  }
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const message = data?.error ?? res.statusText;
    void clientLog("warn", `${init?.method ?? "GET"} ${path} → ${res.status}: ${message}`, "api");
    throw new ApiError(res.status, message);
  }
  return data as T;
}

export const api = {
  health: () => req<{ ok: boolean; version: string }>("/api/health"),
  listDevices: () => req<Device[]>("/api/devices"),
  connect: (p: ConnectParams) =>
    req<Device>("/api/connect", { method: "POST", body: JSON.stringify(p) }),
  discover: (cidr: string, port: number | undefined, tls: boolean) =>
    req<Discovered[]>("/api/discover", {
      method: "POST",
      body: JSON.stringify({ cidr, port, tls }),
    }),
  disconnect: (id: string) =>
    req<{ ok: boolean }>(`/api/devices/${id}/disconnect`, { method: "POST" }),
  info: (id: string) => req<DeviceInfo>(`/api/devices/${id}/info`),
  powerState: (id: string) => req<PowerStatus>(`/api/devices/${id}/power`),
  power: (id: string, action: PowerAction) =>
    req<{ ok: boolean; returnValue: number }>(`/api/devices/${id}/power`, {
      method: "POST",
      body: JSON.stringify({ action }),
    }),
  boot: (id: string, device: string, power = "reset") =>
    req<{ ok: boolean }>(`/api/devices/${id}/boot`, {
      method: "POST",
      body: JSON.stringify({ device, power }),
    }),
  hardware: (id: string) => req<Hardware>(`/api/devices/${id}/hardware`),
  network: (id: string) => req<NetworkInterface[]>(`/api/devices/${id}/network`),
  browseClasses: (id: string) => req<string[]>(`/api/devices/${id}/browse/classes`),
  browse: (id: string, className: string) =>
    req<unknown>(`/api/devices/${id}/browse?class=${encodeURIComponent(className)}`),
  certificates: (id: string) => req<Certificate[]>(`/api/devices/${id}/certificates`),
  addTrustedRoot: (id: string, certificate: string) =>
    req<{ ok: boolean }>(`/api/devices/${id}/certificates`, {
      method: "POST",
      body: JSON.stringify({ certificate }),
    }),
  deleteCertificate: (id: string, instanceId: string) =>
    req<{ ok: boolean }>(`/api/devices/${id}/certificates/${encodeURIComponent(instanceId)}`, {
      method: "DELETE",
    }),
  alarms: (id: string) => req<Alarm[]>(`/api/devices/${id}/alarms`),
  addAlarm: (
    id: string,
    body: { name: string; startTime: string; intervalMinutes: number; deleteOnCompletion: boolean },
  ) => req<{ ok: boolean }>(`/api/devices/${id}/alarms`, { method: "POST", body: JSON.stringify(body) }),
  deleteAlarm: (id: string, instanceId: string) =>
    req<{ ok: boolean }>(`/api/devices/${id}/alarms/${encodeURIComponent(instanceId)}`, {
      method: "DELETE",
    }),
  accounts: (id: string) => req<Account[]>(`/api/devices/${id}/accounts`),
  addAccount: (
    id: string,
    body: { username: string; password: string; accessPermission: number; realms: number[] },
  ) =>
    req<{ ok: boolean }>(`/api/devices/${id}/accounts`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  removeAccount: (id: string, handle: number) =>
    req<{ ok: boolean }>(`/api/devices/${id}/accounts/${handle}`, { method: "DELETE" }),
  setAccountEnabled: (id: string, handle: number, enabled: boolean) =>
    req<{ ok: boolean }>(`/api/devices/${id}/accounts/${handle}`, {
      method: "POST",
      body: JSON.stringify({ enabled }),
    }),
  iderStart: (id: string, isoPath: string, boot: boolean) =>
    req<{ ok: boolean }>(`/api/devices/${id}/ider/start`, {
      method: "POST",
      body: JSON.stringify({ isoPath, boot }),
    }),
  iderStop: (id: string) =>
    req<{ ok: boolean }>(`/api/devices/${id}/ider/stop`, { method: "POST" }),
  iderStatus: (id: string) =>
    req<{ active: boolean; stats: IderStats }>(`/api/devices/${id}/ider/status`),
  eventLog: (id: string) => req<EventLogEntry[]>(`/api/devices/${id}/eventlog`),
  auditLog: (id: string) => req<AuditLogEntry[]>(`/api/devices/${id}/auditlog`),
};

// WebSocket URL builder for redirection (SOL/KVM), token via query param.
export async function wsUrl(path: string): Promise<string> {
  const { baseUrl, token } = await sidecar();
  const u = new URL(baseUrl);
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  u.pathname = path;
  if (token) u.searchParams.set("access_token", token);
  return u.toString();
}

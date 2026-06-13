import { useEffect, useRef, useState } from "react";
import {
  MonitorSmartphone,
  Play,
  Square,
  Keyboard,
  Type,
  Camera,
  Expand,
  Maximize2,
  Minimize2,
  Video,
  ChevronDown,
} from "lucide-react";
import { wsUrl } from "../../lib/api";
import {
  AmtKvmClient,
  amtKeyFromEvent,
  type ColorDepth,
  type Compression,
} from "../../lib/amtKvm";
import { Badge, Button } from "../../lib/ui";

type State = "idle" | "connecting" | "running" | "error" | "closed";

const COLOR_DEPTHS: { value: ColorDepth; label: string }[] = [
  { value: "16", label: "16-bit color (best)" },
  { value: "8", label: "8-bit color (faster)" },
  { value: "gray8", label: "8-bit grayscale" },
  { value: "gray4", label: "4-bit grayscale (fastest)" },
];
const COMPRESSIONS: { value: Compression; label: string }[] = [
  { value: "none", label: "None (RLE)" },
  { value: "zlib", label: "ZLib (more compression)" },
];

// Special key chords (X11 keysyms). Modifiers: Ctrl 0xffe3, Alt 0xffe9,
// Shift 0xffe1, Win 0xffe7.
const SPECIAL_KEYS: { label: string; keys: number[] }[] = [
  { label: "Ctrl+Alt+Del", keys: [0xffe3, 0xffe9, 0xffff] },
  { label: "Alt+Tab", keys: [0xffe9, 0xff09] },
  { label: "Alt+F4", keys: [0xffe9, 0xffc1] },
  { label: "Ctrl+Esc", keys: [0xffe3, 0xff1b] },
  { label: "Ctrl+Shift+Esc", keys: [0xffe3, 0xffe1, 0xff1b] },
  { label: "Windows", keys: [0xffe7] },
  { label: "Escape", keys: [0xff1b] },
  { label: "Delete", keys: [0xffff] },
  { label: "F2 (Setup)", keys: [0xffbf] },
  { label: "F8 (Boot menu)", keys: [0xffc5] },
  { label: "F10", keys: [0xffc7] },
  { label: "F12 (Network boot)", keys: [0xffc9] },
];

export function KvmTab({ id }: { id: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const clientRef = useRef<AmtKvmClient | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const maskRef = useRef(0);
  const [state, setState] = useState<State>("idle");
  const [error, setError] = useState("");
  const [colorDepth, setColorDepth] = useState<ColorDepth>("16");
  const [compression, setCompression] = useState<Compression>("none");
  const [viewOnly, setViewOnly] = useState(false);
  const [actualSize, setActualSize] = useState(false);
  const [keysOpen, setKeysOpen] = useState(false);
  const [recording, setRecording] = useState(false);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  // Data-activity LED: lights up while screen data is arriving, dims when idle.
  const [active, setActive] = useState(false);
  const activeTimer = useRef<number | null>(null);

  function pulseActivity() {
    setActive(true);
    if (activeTimer.current) clearTimeout(activeTimer.current);
    activeTimer.current = window.setTimeout(() => setActive(false), 150);
  }

  useEffect(
    () => () => {
      wsRef.current?.close();
      if (activeTimer.current) clearTimeout(activeTimer.current);
    },
    [],
  );

  function toggleRecord() {
    if (recording) {
      recorderRef.current?.stop();
      return;
    }
    const canvas = canvasRef.current as (HTMLCanvasElement & { captureStream?: (fps: number) => MediaStream }) | null;
    if (!canvas?.captureStream) return;
    try {
      const stream = canvas.captureStream(15);
      const rec = new MediaRecorder(stream, { mimeType: "video/webm" });
      chunksRef.current = [];
      rec.ondataavailable = (e) => {
        if (e.data.size) chunksRef.current.push(e.data);
      };
      rec.onstop = () => {
        const blob = new Blob(chunksRef.current, { type: "video/webm" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `kvm-${Date.now()}.webm`;
        a.click();
        URL.revokeObjectURL(url);
        setRecording(false);
      };
      rec.start();
      recorderRef.current = rec;
      setRecording(true);
    } catch {
      /* recording unsupported */
    }
  }

  function screenshot() {
    const url = canvasRef.current?.toDataURL("image/png");
    if (!url) return;
    const a = document.createElement("a");
    a.href = url;
    a.download = `kvm-${Date.now()}.png`;
    a.click();
  }

  function typeText() {
    const text = prompt("Type text into the remote screen:");
    if (text) clientRef.current?.sendText(text);
  }

  function toggleFullscreen() {
    const el = containerRef.current;
    if (!el) return;
    if (document.fullscreenElement) document.exitFullscreen();
    else el.requestFullscreen?.();
  }

  async function connect() {
    const canvas = canvasRef.current;
    if (!canvas) return;
    setError("");
    setState("connecting");

    const client = new AmtKvmClient(
      canvas,
      (b) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) wsRef.current.send(b);
      },
      (s, detail) => {
        if (s === "running") setState("running");
        else if (s === "error") {
          setState("error");
          setError(detail ?? "KVM protocol error");
        }
      },
      { colorDepth, compression },
    );
    clientRef.current = client;

    const url = await wsUrl(`/api/devices/${id}/kvm`);
    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;
    ws.onmessage = (ev) => {
      pulseActivity();
      client.processData(ev.data as ArrayBuffer);
    };
    ws.onerror = () => {
      setState("error");
      setError("WebSocket error - is the device powered on with KVM enabled in MEBx?");
    };
    ws.onclose = () => setState((s) => (s === "error" ? s : "closed"));
  }

  function disconnect() {
    wsRef.current?.close();
    setState("closed");
  }

  // --- input forwarding ---

  function pointerPos(e: React.MouseEvent<HTMLCanvasElement>) {
    const c = canvasRef.current!;
    const rect = c.getBoundingClientRect();
    const x = Math.round(((e.clientX - rect.left) * c.width) / rect.width);
    const y = Math.round(((e.clientY - rect.top) * c.height) / rect.height);
    return { x, y };
  }

  function onMouse(e: React.MouseEvent<HTMLCanvasElement>, mask: number) {
    if (state !== "running" || viewOnly) return;
    const { x, y } = pointerPos(e);
    clientRef.current?.sendPointer(x, y, mask);
  }

  function onKey(e: React.KeyboardEvent, down: boolean) {
    if (state !== "running" || viewOnly) return;
    const k = amtKeyFromEvent(e.nativeEvent);
    if (k != null) {
      clientRef.current?.sendKey(k, down);
      e.preventDefault();
    }
  }

  const tone = state === "running" ? "good" : state === "error" ? "bad" : "muted";

  return (
    <div className="flex h-full flex-col">
      <div className="mb-3 flex items-center gap-3">
        <MonitorSmartphone className="h-5 w-5 text-(--color-accent)" />
        <span className="font-medium">Remote Desktop (KVM)</span>
        <Badge tone={tone}>{state}</Badge>
        {state === "running" && (
          <span
            title={active ? "Receiving screen data" : "Connected (idle)"}
            className={
              "h-2.5 w-2.5 rounded-full bg-(--color-good) transition-opacity duration-150 " +
              (active
                ? "opacity-100 shadow-[0_0_6px_var(--color-good)]"
                : "opacity-25")
            }
          />
        )}
        <div className="ml-auto flex items-center gap-2">
          {(state === "idle" || state === "closed" || state === "error") && (
            <>
              <select
                value={colorDepth}
                onChange={(e) => setColorDepth(e.target.value as ColorDepth)}
                title="Color depth"
                className="rounded-md border border-(--color-border) bg-(--color-bg) px-2 py-1.5 text-sm outline-none focus:border-(--color-accent)"
              >
                {COLOR_DEPTHS.map((c) => (
                  <option key={c.value} value={c.value}>{c.label}</option>
                ))}
              </select>
              <select
                value={compression}
                onChange={(e) => setCompression(e.target.value as Compression)}
                title="Compression"
                className="rounded-md border border-(--color-border) bg-(--color-bg) px-2 py-1.5 text-sm outline-none focus:border-(--color-accent)"
              >
                {COMPRESSIONS.map((c) => (
                  <option key={c.value} value={c.value}>{c.label}</option>
                ))}
              </select>
            </>
          )}
          {state === "running" && (
            <>
              <label className="flex items-center gap-1 text-sm text-(--color-muted)" title="Stop sending input">
                <input type="checkbox" checked={viewOnly} onChange={(e) => setViewOnly(e.target.checked)} />
                View only
              </label>
              <Button onClick={() => setActualSize((v) => !v)} title="Toggle fit / actual size">
                {actualSize ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
              </Button>
              <Button onClick={typeText} title="Type text">
                <Type className="h-4 w-4" />
              </Button>
              <Button onClick={screenshot} title="Screenshot">
                <Camera className="h-4 w-4" />
              </Button>
              <Button onClick={toggleFullscreen} title="Fullscreen">
                <Expand className="h-4 w-4" />
              </Button>
              <Button onClick={toggleRecord} title="Record video" variant={recording ? "danger" : "default"}>
                <Video className="h-4 w-4" /> {recording ? "Stop" : "Rec"}
              </Button>
              <div className="relative">
                <Button onClick={() => setKeysOpen((o) => !o)} title="Send key combination">
                  <Keyboard className="h-4 w-4" /> Keys <ChevronDown className="h-3.5 w-3.5" />
                </Button>
                {keysOpen && (
                  <div className="absolute right-0 z-20 mt-1 w-52 rounded-md border border-(--color-border) bg-(--color-panel-2) py-1 shadow-xl">
                    {SPECIAL_KEYS.map((k) => (
                      <button
                        key={k.label}
                        onClick={() => {
                          clientRef.current?.sendCombo(k.keys);
                          setKeysOpen(false);
                        }}
                        className="flex w-full items-center px-3 py-1.5 text-left text-sm hover:bg-(--color-border)"
                      >
                        {k.label}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
          {state === "running" ? (
            <Button variant="danger" onClick={disconnect}>
              <Square className="h-4 w-4" /> Disconnect
            </Button>
          ) : (
            <Button variant="primary" onClick={connect} disabled={state === "connecting"}>
              <Play className="h-4 w-4" /> Connect
            </Button>
          )}
        </div>
      </div>
      {error && (
        <div className="mb-3 rounded-md bg-(--color-bad)/15 px-3 py-2 text-sm text-(--color-bad)">
          {error}
        </div>
      )}
      <div
        ref={containerRef}
        className="flex flex-1 items-center justify-center overflow-auto rounded-lg border border-(--color-border) bg-black"
      >
        <canvas
          ref={canvasRef}
          width={640}
          height={400}
          tabIndex={0}
          className={actualSize ? "outline-none" : "max-h-full max-w-full outline-none"}
          style={{ imageRendering: "pixelated" }}
          onMouseDown={(e) => {
            canvasRef.current?.focus();
            maskRef.current |= 1 << e.button;
            onMouse(e, maskRef.current);
          }}
          onMouseUp={(e) => {
            maskRef.current &= ~(1 << e.button);
            onMouse(e, maskRef.current);
          }}
          onMouseMove={(e) => onMouse(e, maskRef.current)}
          onWheel={(e) => {
            if (state !== "running") return;
            const bit = e.deltaY < 0 ? 3 : 4;
            const { x, y } = pointerPos(e as unknown as React.MouseEvent<HTMLCanvasElement>);
            clientRef.current?.sendPointer(x, y, maskRef.current | (1 << bit));
            clientRef.current?.sendPointer(x, y, maskRef.current);
          }}
          onContextMenu={(e) => e.preventDefault()}
          onKeyDown={(e) => onKey(e, true)}
          onKeyUp={(e) => onKey(e, false)}
        />
      </div>
      {state === "running" && (
        <p className="mt-2 text-center text-xs text-(--color-muted)">
          Click the screen to capture keyboard &amp; mouse.
        </p>
      )}
    </div>
  );
}

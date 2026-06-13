import { useEffect, useRef, useState } from "react";
import { MonitorSmartphone, Play, Square, Keyboard } from "lucide-react";
import { wsUrl } from "../../lib/api";
import { AmtKvmClient, amtKeyFromEvent } from "../../lib/amtKvm";
import { Badge, Button } from "../../lib/ui";

type State = "idle" | "connecting" | "running" | "error" | "closed";

export function KvmTab({ id }: { id: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const clientRef = useRef<AmtKvmClient | null>(null);
  const maskRef = useRef(0);
  const [state, setState] = useState<State>("idle");
  const [error, setError] = useState("");

  useEffect(() => () => wsRef.current?.close(), []);

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
    );
    clientRef.current = client;

    const url = await wsUrl(`/api/devices/${id}/kvm`);
    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;
    ws.onmessage = (ev) => client.processData(ev.data as ArrayBuffer);
    ws.onerror = () => {
      setState("error");
      setError("WebSocket error — is the device powered on with KVM enabled in MEBx?");
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
    if (state !== "running") return;
    const { x, y } = pointerPos(e);
    clientRef.current?.sendPointer(x, y, mask);
  }

  function onKey(e: React.KeyboardEvent, down: boolean) {
    if (state !== "running") return;
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
        <MonitorSmartphone className="h-5 w-5 text-[--color-accent]" />
        <span className="font-medium">Remote Desktop (KVM)</span>
        <Badge tone={tone}>{state}</Badge>
        <div className="ml-auto flex gap-2">
          {state === "running" && (
            <Button onClick={() => clientRef.current?.sendCtrlAltDel()} title="Send Ctrl+Alt+Del">
              <Keyboard className="h-4 w-4" /> Ctrl+Alt+Del
            </Button>
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
        <div className="mb-3 rounded-md bg-[--color-bad]/15 px-3 py-2 text-sm text-[--color-bad]">
          {error}
        </div>
      )}
      <div className="flex flex-1 items-center justify-center overflow-auto rounded-lg border border-[--color-border] bg-black">
        <canvas
          ref={canvasRef}
          width={640}
          height={400}
          tabIndex={0}
          className="max-h-full max-w-full outline-none"
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
        <p className="mt-2 text-center text-xs text-[--color-muted]">
          Click the screen to capture keyboard &amp; mouse.
        </p>
      )}
    </div>
  );
}

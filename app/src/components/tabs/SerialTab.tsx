import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { Play, Square, TerminalSquare } from "lucide-react";
import { wsUrl } from "../../lib/api";
import { Badge, Button } from "../../lib/ui";

type State = "idle" | "connecting" | "connected" | "closed" | "error";

export function SerialTab({ id }: { id: string }) {
  const mountRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [state, setState] = useState<State>("idle");
  const [error, setError] = useState("");

  // Create the terminal once when the tab mounts.
  useEffect(() => {
    if (!mountRef.current) return;
    const term = new Terminal({
      cursorBlink: true,
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
      fontSize: 13,
      theme: { background: "#0b0e14", foreground: "#e4e9f2", cursor: "#4c8dff" },
      convertEol: false,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(mountRef.current);
    fit.fit();
    termRef.current = term;
    fitRef.current = fit;

    const onResize = () => fit.fit();
    window.addEventListener("resize", onResize);
    return () => {
      window.removeEventListener("resize", onResize);
      wsRef.current?.close();
      term.dispose();
    };
  }, []);

  async function connect() {
    const term = termRef.current;
    if (!term) return;
    setError("");
    setState("connecting");
    term.clear();
    term.writeln("\x1b[90mConnecting to serial console…\x1b[0m");

    const url = await wsUrl(`/api/devices/${id}/sol`);
    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;

    ws.onopen = () => {
      setState("connected");
      fitRef.current?.fit();
      term.focus();
    };
    ws.onmessage = (ev) => {
      if (typeof ev.data === "string") term.write(ev.data);
      else term.write(new Uint8Array(ev.data));
    };
    ws.onerror = () => {
      setState("error");
      setError("WebSocket error — is the device powered on and SOL enabled?");
    };
    ws.onclose = () => setState((s) => (s === "error" ? s : "closed"));

    term.onData((d) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(d);
    });
  }

  function disconnect() {
    wsRef.current?.close();
    setState("closed");
  }

  const tone = state === "connected" ? "good" : state === "error" ? "bad" : "muted";

  return (
    <div className="flex h-full flex-col">
      <div className="mb-3 flex items-center gap-3">
        <TerminalSquare className="h-5 w-5 text-[--color-accent]" />
        <span className="font-medium">Serial-over-LAN</span>
        <Badge tone={tone}>{state}</Badge>
        <div className="ml-auto flex gap-2">
          {state === "connected" ? (
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
      <div
        ref={mountRef}
        className="flex-1 overflow-hidden rounded-lg border border-[--color-border] bg-[#0b0e14] p-2"
      />
    </div>
  );
}

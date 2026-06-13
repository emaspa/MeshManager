import { useQuery } from "@tanstack/react-query";
import { X, Server, Github } from "lucide-react";
import { api } from "../lib/api";
import { useUi } from "../store";
import { APP_NAME, REPO_URL, openExternal } from "../lib/native";
import { Button } from "../lib/ui";

export function AboutDialog() {
  const setAboutOpen = useUi((s) => s.setAboutOpen);
  const health = useQuery({ queryKey: ["health"], queryFn: api.health });
  const version = health.data?.version ?? "0.1.0";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-[400px] rounded-xl border border-(--color-border) bg-(--color-panel) p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">About</h2>
          <Button variant="ghost" className="px-1.5 py-1" onClick={() => setAboutOpen(false)}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <div className="flex flex-col items-center text-center">
          <div className="rounded-2xl bg-(--color-accent)/15 p-3">
            <Server className="h-8 w-8 text-(--color-accent)" />
          </div>
          <div className="mt-3 text-xl font-semibold">{APP_NAME}</div>
          <div className="font-mono text-sm text-(--color-muted)">v{version}</div>
          <p className="mt-3 text-sm text-(--color-muted)">
            A modern Intel® AMT / vPro management console - power, boot, hardware,
            Serial-over-LAN, KVM, IDE-R and more.
          </p>

          <Button
            variant="primary"
            className="mt-5"
            onClick={() => openExternal(REPO_URL)}
          >
            <Github className="h-4 w-4" /> View on GitHub
          </Button>
          <button
            onClick={() => openExternal(REPO_URL)}
            className="mt-2 text-xs text-(--color-muted) underline decoration-dotted underline-offset-2 hover:text-(--color-text)"
          >
            {REPO_URL.replace("https://", "")}
          </button>
        </div>
      </div>
    </div>
  );
}

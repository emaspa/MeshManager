import { Sidebar } from "./components/Sidebar";
import { ConnectDialog } from "./components/ConnectDialog";
import { DeviceView } from "./components/DeviceView";
import { useUi } from "./store";

export default function App() {
  const selectedId = useUi((s) => s.selectedId);
  const connectOpen = useUi((s) => s.connectOpen);

  return (
    <div className="flex h-full w-full">
      <Sidebar />
      <main className="flex-1 overflow-hidden">
        {selectedId ? (
          <DeviceView key={selectedId} id={selectedId} />
        ) : (
          <EmptyState />
        )}
      </main>
      {connectOpen && <ConnectDialog />}
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center text-center text-[--color-muted]">
      <div className="text-5xl">🖧</div>
      <h2 className="mt-4 text-lg font-medium text-[--color-text]">No device selected</h2>
      <p className="mt-1 max-w-sm text-sm">
        Add an Intel AMT / vPro device from the sidebar to view power state, hardware inventory,
        logs, and open a remote session.
      </p>
    </div>
  );
}

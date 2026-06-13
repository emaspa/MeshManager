import { create } from "zustand";

export interface ConnectPrefill {
  host?: string;
  port?: number;
  tls?: boolean;
}

interface UiState {
  selectedId: string | null;
  select: (id: string | null) => void;

  connectOpen: boolean;
  connectPrefill: ConnectPrefill | null;
  openConnect: (prefill?: ConnectPrefill) => void;
  closeConnect: () => void;

  discoverOpen: boolean;
  setDiscoverOpen: (open: boolean) => void;
}

export const useUi = create<UiState>((set) => ({
  selectedId: null,
  select: (id) => set({ selectedId: id }),

  connectOpen: false,
  connectPrefill: null,
  openConnect: (prefill) => set({ connectOpen: true, connectPrefill: prefill ?? null }),
  closeConnect: () => set({ connectOpen: false, connectPrefill: null }),

  discoverOpen: false,
  setDiscoverOpen: (open) => set({ discoverOpen: open }),
}));

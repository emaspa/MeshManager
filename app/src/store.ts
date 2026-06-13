import { create } from "zustand";

interface UiState {
  selectedId: string | null;
  select: (id: string | null) => void;
  connectOpen: boolean;
  setConnectOpen: (open: boolean) => void;
}

export const useUi = create<UiState>((set) => ({
  selectedId: null,
  select: (id) => set({ selectedId: id }),
  connectOpen: false,
  setConnectOpen: (open) => set({ connectOpen: open }),
}));

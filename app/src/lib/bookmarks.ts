import { create } from "zustand";
import { persist } from "zustand/middleware";

// A saved connection. Persisted to localStorage so devices survive disconnects
// and app restarts. Password is only stored when the user opts in.
export interface Bookmark {
  id: string;
  name: string;
  host: string;
  port?: number;
  tls: boolean;
  insecure: boolean;
  username: string;
  password?: string;
}

interface BookmarkState {
  bookmarks: Bookmark[];
  /** Insert or update by host+effective-port; returns the stored bookmark. */
  upsert: (b: Omit<Bookmark, "id">) => Bookmark;
  update: (id: string, patch: Partial<Bookmark>) => void;
  remove: (id: string) => void;
}

/** The port a bookmark connects on (explicit, or the AMT default for its mode). */
export function effectivePort(b: { port?: number; tls: boolean }): number {
  return b.port ?? (b.tls ? 16993 : 16992);
}

function newId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) return crypto.randomUUID();
  return Math.random().toString(36).slice(2);
}

export const useBookmarks = create<BookmarkState>()(
  persist(
    (set, get) => ({
      bookmarks: [],
      upsert: (b) => {
        const existing = get().bookmarks.find(
          (x) => x.host === b.host && effectivePort(x) === effectivePort(b),
        );
        if (existing) {
          const merged = { ...existing, ...b };
          set({ bookmarks: get().bookmarks.map((x) => (x.id === existing.id ? merged : x)) });
          return merged;
        }
        const created: Bookmark = { ...b, id: newId() };
        set({ bookmarks: [...get().bookmarks, created] });
        return created;
      },
      update: (id, patch) =>
        set({ bookmarks: get().bookmarks.map((x) => (x.id === id ? { ...x, ...patch } : x)) }),
      remove: (id) => set({ bookmarks: get().bookmarks.filter((x) => x.id !== id) }),
    }),
    { name: "meshmanager.bookmarks" },
  ),
);

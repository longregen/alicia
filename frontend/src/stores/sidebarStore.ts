import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface SidebarState {
  isCollapsed: boolean;
  width: number;
  setCollapsed: (collapsed: boolean) => void;
  toggleCollapsed: () => void;
  setWidth: (width: number) => void;
}

const MIN_WIDTH = 200;
const MAX_WIDTH = 480;
const COLLAPSED_WIDTH = 64;
const DEFAULT_WIDTH = 300;

export const useSidebarStore = create<SidebarState>()(
  persist(
    (set) => ({
      isCollapsed: false,
      width: DEFAULT_WIDTH,
      setCollapsed: (collapsed: boolean) => set({ isCollapsed: collapsed }),
      toggleCollapsed: () => set((state) => ({ isCollapsed: !state.isCollapsed })),
      setWidth: (width: number) => {
        // Clamp width between min and max
        const clampedWidth = Math.min(Math.max(width, MIN_WIDTH), MAX_WIDTH);
        set({ width: clampedWidth });
      },
    }),
    {
      name: 'sidebar-storage',
    }
  )
);

export { MIN_WIDTH, MAX_WIDTH, COLLAPSED_WIDTH, DEFAULT_WIDTH };

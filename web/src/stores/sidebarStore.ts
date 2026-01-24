import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';

interface SidebarState {
  // Desktop collapsed state (icon-only mode)
  isCollapsed: boolean;
  width: number;
  setCollapsed: (collapsed: boolean) => void;
  toggleCollapsed: () => void;
  setWidth: (width: number) => void;
  // Mobile open/close state
  isOpen: boolean;
  setOpen: (open: boolean) => void;
  toggleOpen: () => void;
}

const MIN_WIDTH = 200;
const MAX_WIDTH = 480;
const COLLAPSED_WIDTH = 64;
const DEFAULT_WIDTH = 300;

export const useSidebarStore = create<SidebarState>()(
  persist(
    immer((set) => ({
      isCollapsed: false,
      width: DEFAULT_WIDTH,
      isOpen: false,
      setCollapsed: (collapsed: boolean) =>
        set((state) => {
          state.isCollapsed = collapsed;
        }),
      toggleCollapsed: () =>
        set((state) => {
          state.isCollapsed = !state.isCollapsed;
        }),
      setWidth: (width: number) => {
        // Clamp width between min and max
        const clampedWidth = Math.min(Math.max(width, MIN_WIDTH), MAX_WIDTH);
        set((state) => {
          state.width = clampedWidth;
        });
      },
      setOpen: (open: boolean) =>
        set((state) => {
          state.isOpen = open;
        }),
      toggleOpen: () =>
        set((state) => {
          state.isOpen = !state.isOpen;
        }),
    })),
    {
      name: 'sidebar-storage',
      // Only persist collapsed state and width, not open state
      partialize: (state) => ({ isCollapsed: state.isCollapsed, width: state.width }),
    }
  )
);

export { MIN_WIDTH, MAX_WIDTH, COLLAPSED_WIDTH, DEFAULT_WIDTH };

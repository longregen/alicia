import { describe, it, expect, beforeEach } from 'vitest';
import { useSidebarStore, MIN_WIDTH, MAX_WIDTH, COLLAPSED_WIDTH, DEFAULT_WIDTH } from './sidebarStore';

describe('sidebarStore', () => {
  beforeEach(() => {
    // Reset store to initial state
    const { isCollapsed, width, setCollapsed, setWidth } = useSidebarStore.getState();
    if (isCollapsed) {
      setCollapsed(false);
    }
    if (width !== DEFAULT_WIDTH) {
      setWidth(DEFAULT_WIDTH);
    }
  });

  it('should have correct initial state', () => {
    const { isCollapsed, width } = useSidebarStore.getState();
    expect(isCollapsed).toBe(false);
    expect(width).toBe(DEFAULT_WIDTH);
  });

  it('should toggle collapsed state', () => {
    const { toggleCollapsed } = useSidebarStore.getState();

    toggleCollapsed();
    expect(useSidebarStore.getState().isCollapsed).toBe(true);

    toggleCollapsed();
    expect(useSidebarStore.getState().isCollapsed).toBe(false);
  });

  it('should set collapsed state', () => {
    const { setCollapsed } = useSidebarStore.getState();

    setCollapsed(true);
    expect(useSidebarStore.getState().isCollapsed).toBe(true);

    setCollapsed(false);
    expect(useSidebarStore.getState().isCollapsed).toBe(false);
  });

  it('should clamp width to MIN_WIDTH', () => {
    const { setWidth } = useSidebarStore.getState();

    setWidth(MIN_WIDTH - 50);
    expect(useSidebarStore.getState().width).toBe(MIN_WIDTH);
  });

  it('should clamp width to MAX_WIDTH', () => {
    const { setWidth } = useSidebarStore.getState();

    setWidth(MAX_WIDTH + 50);
    expect(useSidebarStore.getState().width).toBe(MAX_WIDTH);
  });

  it('should set width within valid range', () => {
    const { setWidth } = useSidebarStore.getState();
    const validWidth = 350;

    setWidth(validWidth);
    expect(useSidebarStore.getState().width).toBe(validWidth);
  });

  it('should export correct constants', () => {
    expect(MIN_WIDTH).toBe(200);
    expect(MAX_WIDTH).toBe(480);
    expect(COLLAPSED_WIDTH).toBe(64);
    expect(DEFAULT_WIDTH).toBe(300);
  });
});

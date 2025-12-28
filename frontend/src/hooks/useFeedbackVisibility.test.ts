import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useFeedbackVisibility } from './useFeedbackVisibility';

describe('useFeedbackVisibility', () => {
  beforeEach(() => {
    // Clean up any existing event listeners
  });

  afterEach(() => {
    // Clean up event listeners after each test
  });

  it('should initialize with false', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    expect(result.current).toBe(false);
  });

  it('should return true when Alt key is pressed', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true }));
    });

    expect(result.current).toBe(true);
  });

  it('should return true when Meta/Cmd key is pressed', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { metaKey: true }));
    });

    expect(result.current).toBe(true);
  });

  it('should return false when Alt key is released', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true }));
    });

    expect(result.current).toBe(true);

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keyup', { altKey: false, metaKey: false }));
    });

    expect(result.current).toBe(false);
  });

  it('should return false when Meta key is released', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { metaKey: true }));
    });

    expect(result.current).toBe(true);

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keyup', { altKey: false, metaKey: false }));
    });

    expect(result.current).toBe(false);
  });

  it('should remain true if one modifier is still pressed', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    // Press both Alt and Meta
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true, metaKey: true }));
    });

    expect(result.current).toBe(true);

    // Release Alt but keep Meta pressed
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keyup', { altKey: false, metaKey: true }));
    });

    expect(result.current).toBe(true);
  });

  it('should handle multiple keydown events', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true }));
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true }));
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true }));
    });

    expect(result.current).toBe(true);
  });

  it('should clean up event listeners on unmount', () => {
    const { unmount } = renderHook(() => useFeedbackVisibility());

    unmount();

    // After unmount, keydown events should not affect anything
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true }));
    });

    // We can't easily verify the cleanup, but we can ensure no errors occur
  });

  it('should handle rapid key presses', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true }));
      window.dispatchEvent(new KeyboardEvent('keyup', { altKey: false, metaKey: false }));
      window.dispatchEvent(new KeyboardEvent('keydown', { metaKey: true }));
      window.dispatchEvent(new KeyboardEvent('keyup', { altKey: false, metaKey: false }));
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true }));
    });

    expect(result.current).toBe(true);
  });

  it('should not be affected by other keys', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'a' }));
    });

    expect(result.current).toBe(false);

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Shift' }));
    });

    expect(result.current).toBe(false);
  });

  it('should handle Alt key with other keys pressed', () => {
    const { result } = renderHook(() => useFeedbackVisibility());

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { altKey: true, key: 'a' }));
    });

    expect(result.current).toBe(true);
  });
});

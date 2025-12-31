import { renderHook, act } from '@testing-library/react';
import { useTheme } from './useTheme';

const THEME_STORAGE_KEY = 'alicia-theme';

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove('dark');
  });

  it('should initialize with system theme by default', () => {
    const { result } = renderHook(() => useTheme());
    expect(result.current.theme).toBe('system');
  });

  it('should initialize with stored theme from localStorage', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'dark');
    const { result } = renderHook(() => useTheme());
    expect(result.current.theme).toBe('dark');
  });

  it('should apply dark class when theme is set to dark', () => {
    const { result } = renderHook(() => useTheme());

    act(() => {
      result.current.setTheme('dark');
    });

    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
  });

  it('should remove dark class when theme is set to light', () => {
    document.documentElement.classList.add('dark');
    const { result } = renderHook(() => useTheme());

    act(() => {
      result.current.setTheme('light');
    });

    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light');
  });

  it('should persist theme changes to localStorage', () => {
    const { result } = renderHook(() => useTheme());

    act(() => {
      result.current.setTheme('dark');
    });

    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');

    act(() => {
      result.current.setTheme('light');
    });

    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light');
  });

  it('should handle system theme preference', () => {
    const { result } = renderHook(() => useTheme());

    act(() => {
      result.current.setTheme('system');
    });

    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('system');
  });
});

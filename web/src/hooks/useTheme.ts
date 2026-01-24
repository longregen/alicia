import { useEffect } from 'react';
import { usePreferencesStore } from '../stores/preferencesStore';

export type Theme = 'light' | 'dark' | 'system';

function getSystemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function getEffectiveTheme(theme: Theme): 'light' | 'dark' {
  return theme === 'system' ? getSystemTheme() : theme;
}

function applyTheme(effectiveTheme: 'light' | 'dark') {
  const root = document.documentElement;
  if (effectiveTheme === 'dark') {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
}

export function useTheme() {
  const theme = usePreferencesStore((state) => state.theme);
  const setTheme = usePreferencesStore((state) => state.updatePreference);

  useEffect(() => {
    const effectiveTheme = getEffectiveTheme(theme);
    applyTheme(effectiveTheme);
  }, [theme]);

  useEffect(() => {
    if (theme !== 'system') return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      const effectiveTheme = getEffectiveTheme('system');
      applyTheme(effectiveTheme);
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [theme]);

  return {
    theme,
    setTheme: (newTheme: Theme) => setTheme('theme', newTheme),
  };
}

// Initialize theme on module load to prevent flash
// Read from localStorage directly since store may not be initialized yet
if (typeof window !== 'undefined') {
  try {
    const stored = localStorage.getItem('alicia-preferences');
    if (stored) {
      const parsed = JSON.parse(stored);
      const theme = parsed?.state?.theme || 'system';
      const effectiveTheme = getEffectiveTheme(theme);
      applyTheme(effectiveTheme);
    } else {
      // Default to system theme
      applyTheme(getSystemTheme());
    }
  } catch {
    applyTheme(getSystemTheme());
  }
}

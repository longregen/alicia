import { useState, useEffect } from 'react';

/**
 * Hook to detect if a media query matches.
 * Useful for responsive behavior that can't be handled with CSS alone.
 *
 * @param query - CSS media query string (e.g., '(max-width: 1023px)')
 * @returns boolean indicating if the media query matches
 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === 'undefined') return false;
    return window.matchMedia(query).matches;
  });

  useEffect(() => {
    const mediaQuery = window.matchMedia(query);
    setMatches(mediaQuery.matches);

    const handler = (event: MediaQueryListEvent) => {
      setMatches(event.matches);
    };

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [query]);

  return matches;
}

/**
 * Hook to detect if viewport is mobile/tablet (< lg breakpoint).
 * Matches Tailwind's lg breakpoint (1024px).
 */
export function useIsMobile(): boolean {
  return useMediaQuery('(max-width: 1023px)');
}

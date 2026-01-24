import { useState, useEffect } from 'react';

// Uses multiple detection: CSS media query (hover: none), touch events, max touch points
export function useIsTouchDevice(): boolean {
  const [isTouchDevice, setIsTouchDevice] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false;

    const hasHover = window.matchMedia('(hover: hover)').matches;
    const hasPointer = window.matchMedia('(pointer: fine)').matches;

    if (hasHover && hasPointer) return false;

    const hasTouchEvents = 'ontouchstart' in window;
    const hasMaxTouchPoints = navigator.maxTouchPoints > 0;

    return hasTouchEvents || hasMaxTouchPoints;
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;

    // Supports convertible devices that switch between touch/pointer modes
    const hoverQuery = window.matchMedia('(hover: hover)');
    const pointerQuery = window.matchMedia('(pointer: fine)');

    const updateTouchStatus = () => {
      const hasHover = hoverQuery.matches;
      const hasPointer = pointerQuery.matches;

      if (hasHover && hasPointer) {
        setIsTouchDevice(false);
        return;
      }

      const hasTouchEvents = 'ontouchstart' in window;
      const hasMaxTouchPoints = navigator.maxTouchPoints > 0;

      setIsTouchDevice(hasTouchEvents || hasMaxTouchPoints);
    };

    hoverQuery.addEventListener('change', updateTouchStatus);
    pointerQuery.addEventListener('change', updateTouchStatus);

    return () => {
      hoverQuery.removeEventListener('change', updateTouchStatus);
      pointerQuery.removeEventListener('change', updateTouchStatus);
    };
  }, []);

  return isTouchDevice;
}


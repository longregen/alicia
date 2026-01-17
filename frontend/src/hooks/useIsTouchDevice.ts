import { useState, useEffect } from 'react';

/**
 * Hook to detect if the current device is a touch device.
 * Uses multiple detection methods for accuracy:
 * 1. CSS media query (hover: none)
 * 2. Touch event support
 * 3. Max touch points
 *
 * Returns true for touch-only devices (phones, tablets)
 * Returns false for devices with hover capability (desktop, laptop)
 */
export function useIsTouchDevice(): boolean {
  const [isTouchDevice, setIsTouchDevice] = useState<boolean>(() => {
    // Initial check on client side
    if (typeof window === 'undefined') return false;

    // Check if the device has hover capability
    const hasHover = window.matchMedia('(hover: hover)').matches;
    const hasPointer = window.matchMedia('(pointer: fine)').matches;

    // If device has hover and fine pointer, it's not touch-only
    if (hasHover && hasPointer) return false;

    // Check for touch support
    const hasTouchEvents = 'ontouchstart' in window;
    const hasMaxTouchPoints = navigator.maxTouchPoints > 0;

    return hasTouchEvents || hasMaxTouchPoints;
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;

    // Listen for changes in hover capability (useful for convertible devices)
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

export default useIsTouchDevice;

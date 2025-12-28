import { useState, useEffect } from 'react';

/**
 * Hook that returns true when Alt or Cmd/Meta key is held down.
 * This is used to reveal feedback controls (votes, notes) in the UI.
 *
 * @returns boolean - true when modifier key is pressed, false otherwise
 */
export function useFeedbackVisibility(): boolean {
  const [showFeedback, setShowFeedback] = useState(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.altKey || e.metaKey) {
        setShowFeedback(true);
      }
    };

    const handleKeyUp = (e: KeyboardEvent) => {
      if (!e.altKey && !e.metaKey) {
        setShowFeedback(false);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
    };
  }, []);

  return showFeedback;
}

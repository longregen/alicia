import { useCallback, useEffect, useRef } from 'react';
import { usePreferencesStore, UserPreferences } from '../stores/preferencesStore';
import { api, UpdatePreferencesRequest } from '../services/api';

const SAVE_DEBOUNCE_MS = 500;

export function usePreferences() {
  const store = usePreferencesStore();
  const { isLoaded, isLoading, setLoading, loadFromServer, setError,
    updatePreference: storeUpdatePreference } = store;
  const saveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingUpdatesRef = useRef<UpdatePreferencesRequest>({});

  useEffect(() => {
    if (!isLoaded && !isLoading) {
      setLoading(true);
      api.getPreferences()
        .then(({ user_id: _, created_at: __, updated_at: ___, ...prefs }) => {
          loadFromServer(prefs);
        })
        .catch((err) => {
          console.error('Failed to load preferences:', err);
          setError(err.message);
        });
    }
  }, [isLoaded, isLoading, setLoading, loadFromServer, setError]);

  const saveToServer = useCallback(() => {
    if (Object.keys(pendingUpdatesRef.current).length === 0) return;

    const updates = { ...pendingUpdatesRef.current };
    pendingUpdatesRef.current = {};

    api.updatePreferences(updates).catch((err) => {
      console.error('Failed to save preferences:', err);
    });
  }, []);

  const updatePreference = useCallback(
    <K extends keyof UserPreferences>(key: K, value: UserPreferences[K]) => {
      storeUpdatePreference(key, value);
      (pendingUpdatesRef.current as Record<string, unknown>)[key] = value ?? undefined;

      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
      saveTimeoutRef.current = setTimeout(saveToServer, SAVE_DEBOUNCE_MS);
    },
    [storeUpdatePreference, saveToServer]
  );

  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
        saveToServer();
      }
    };
  }, [saveToServer]);

  return {
    ...store,
    updatePreference,
  };
}

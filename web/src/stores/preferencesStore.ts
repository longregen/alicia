import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import sharedDefaults from '@shared/preferences/defaults.json';

export interface UserPreferences {
  theme: 'light' | 'dark' | 'system';
  audio_output_enabled: boolean;
  voice_speed: number;
  memory_min_importance: number | null;
  memory_min_historical: number | null;
  memory_min_personal: number | null;
  memory_min_factual: number | null;
  memory_retrieval_count: number;
  max_tokens: number;
  max_tool_iterations: number;
  temperature: number;
  pareto_target_score: number;
  pareto_max_generations: number;
  pareto_branches_per_gen: number;
  pareto_archive_size: number;
  pareto_enable_crossover: boolean;
  confirm_delete_memory: boolean;
  show_relevance_scores: boolean;
}

interface PreferencesState extends UserPreferences {
  isLoading: boolean;
  isLoaded: boolean;
  error: string | null;
}

interface PreferencesActions {
  updatePreference: <K extends keyof UserPreferences>(key: K, value: UserPreferences[K]) => void;
  loadFromServer: (prefs: UserPreferences) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

const DEFAULT_PREFERENCES: UserPreferences = sharedDefaults as UserPreferences;

export const usePreferencesStore = create<PreferencesState & PreferencesActions>()(
  immer((set) => ({
    ...DEFAULT_PREFERENCES,
    isLoading: false,
    isLoaded: false,
    error: null,

    updatePreference: (key, value) =>
      set((state) => {
        (state as Record<keyof UserPreferences, unknown>)[key] = value;
      }),

    loadFromServer: (prefs) =>
      set((state) => {
        Object.assign(state, prefs);
        state.isLoaded = true;
        state.isLoading = false;
        state.error = null;
      }),

    setLoading: (loading) =>
      set((state) => {
        state.isLoading = loading;
      }),

    setError: (error) =>
      set((state) => {
        state.error = error;
        state.isLoading = false;
      }),
  }))
);

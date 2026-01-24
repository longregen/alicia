import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

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

const DEFAULT_PREFERENCES: UserPreferences = {
  theme: 'system',
  audio_output_enabled: false,
  voice_speed: 1.0,
  memory_min_importance: 3,
  memory_min_historical: 2,
  memory_min_personal: 2,
  memory_min_factual: 2,
  memory_retrieval_count: 4,
  max_tokens: 16384,
  pareto_target_score: 3.0,
  pareto_max_generations: 7,
  pareto_branches_per_gen: 3,
  pareto_archive_size: 50,
  pareto_enable_crossover: true,
  confirm_delete_memory: true,
  show_relevance_scores: false,
};

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

import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import type {
  DimensionWeights,
  DimensionScores,
  EliteSummary,
} from '../types/protocol';

export type PresetId = 'accuracy' | 'speed' | 'reliable' | 'creative' | 'balanced';

export interface PivotPreset {
  id: PresetId;
  label: string;
  icon: string;
  weights: DimensionWeights;
  description: string;
}

// Default preset definitions matching frontend-ux-plan.md
export const PIVOT_PRESETS: PivotPreset[] = [
  {
    id: 'accuracy',
    label: 'Accurate',
    icon: '✓',
    weights: {
      successRate: 0.4,
      quality: 0.25,
      efficiency: 0.1,
      robustness: 0.1,
      generalization: 0.1,
      diversity: 0.03,
      innovation: 0.02,
    },
    description: 'Prioritize correct answers over speed',
  },
  {
    id: 'speed',
    label: 'Fast',
    icon: '⚡',
    weights: {
      successRate: 0.2,
      quality: 0.15,
      efficiency: 0.35,
      robustness: 0.15,
      generalization: 0.1,
      diversity: 0.03,
      innovation: 0.02,
    },
    description: 'Quick responses with reasonable accuracy',
  },
  {
    id: 'reliable',
    label: 'Reliable',
    icon: '🛡️',
    weights: {
      successRate: 0.25,
      quality: 0.2,
      efficiency: 0.1,
      robustness: 0.3,
      generalization: 0.1,
      diversity: 0.03,
      innovation: 0.02,
    },
    description: 'Consistent results across different inputs',
  },
  {
    id: 'creative',
    label: 'Creative',
    icon: '🎨',
    weights: {
      successRate: 0.15,
      quality: 0.2,
      efficiency: 0.1,
      robustness: 0.1,
      generalization: 0.1,
      diversity: 0.2,
      innovation: 0.15,
    },
    description: 'Novel approaches and varied solutions',
  },
  {
    id: 'balanced',
    label: 'Balanced',
    icon: '⚖️',
    weights: {
      successRate: 0.25,
      quality: 0.2,
      efficiency: 0.15,
      robustness: 0.15,
      generalization: 0.1,
      diversity: 0.1,
      innovation: 0.05,
    },
    description: 'Equal emphasis on all dimensions',
  },
];

// Get preset by ID
export const getPresetById = (id: PresetId): PivotPreset | undefined =>
  PIVOT_PRESETS.find((p) => p.id === id);

// Default weights (balanced preset)
export const DEFAULT_WEIGHTS: DimensionWeights = PIVOT_PRESETS.find(
  (p) => p.id === 'balanced'
)!.weights;

interface DimensionStoreState {
  // Current weights configuration
  weights: DimensionWeights;
  presetId: PresetId | null;

  // Elite solutions from Pareto archive
  elites: EliteSummary[];
  currentEliteId: string | null;

  // Loading state
  isLoading: boolean;
}

interface DimensionStoreActions {
  // Preset selection
  setPreset: (presetId: PresetId) => void;

  // Custom weight adjustment
  setCustomWeights: (weights: DimensionWeights) => void;
  setDimensionWeight: (
    dimension: keyof DimensionWeights,
    value: number
  ) => void;

  // Elite selection
  selectElite: (eliteId: string) => void;
  updateElites: (elites: EliteSummary[], currentEliteId?: string) => void;

  // Reset
  resetToBalanced: () => void;

  // Loading state
  setLoading: (loading: boolean) => void;
}

type DimensionStore = DimensionStoreState & DimensionStoreActions;

const initialState: DimensionStoreState = {
  weights: DEFAULT_WEIGHTS,
  presetId: 'balanced',
  elites: [],
  currentEliteId: null,
  isLoading: false,
};

// Normalize weights to sum to 1.0
const normalizeWeights = (weights: DimensionWeights): DimensionWeights => {
  const sum =
    weights.successRate +
    weights.quality +
    weights.efficiency +
    weights.robustness +
    weights.generalization +
    weights.diversity +
    weights.innovation;

  if (sum === 0) return DEFAULT_WEIGHTS;

  return {
    successRate: weights.successRate / sum,
    quality: weights.quality / sum,
    efficiency: weights.efficiency / sum,
    robustness: weights.robustness / sum,
    generalization: weights.generalization / sum,
    diversity: weights.diversity / sum,
    innovation: weights.innovation / sum,
  };
};

// Calculate weighted score for an elite solution
export const calculateWeightedScore = (
  scores: DimensionScores,
  weights: DimensionWeights
): number => {
  return (
    scores.successRate * weights.successRate +
    scores.quality * weights.quality +
    scores.efficiency * weights.efficiency +
    scores.robustness * weights.robustness +
    scores.generalization * weights.generalization +
    scores.diversity * weights.diversity +
    scores.innovation * weights.innovation
  );
};

export const useDimensionStore = create<DimensionStore>()(
  immer((set) => ({
    ...initialState,

    setPreset: (presetId) =>
      set((state) => {
        const preset = getPresetById(presetId);
        if (preset) {
          state.weights = preset.weights;
          state.presetId = presetId;
        }
      }),

    setCustomWeights: (weights) =>
      set((state) => {
        state.weights = normalizeWeights(weights);
        state.presetId = null; // Custom weights, no preset
      }),

    setDimensionWeight: (dimension, value) =>
      set((state) => {
        // Clamp value between 0 and 1
        const clampedValue = Math.max(0, Math.min(1, value));
        state.weights[dimension] = clampedValue;
        // Normalize to maintain sum = 1.0
        state.weights = normalizeWeights(state.weights);
        state.presetId = null; // Custom adjustment
      }),

    selectElite: (eliteId) =>
      set((state) => {
        if (state.elites.some((e) => e.id === eliteId)) {
          state.currentEliteId = eliteId;
        }
      }),

    updateElites: (elites, currentEliteId) =>
      set((state) => {
        state.elites = elites;
        if (currentEliteId) {
          state.currentEliteId = currentEliteId;
        } else if (elites.length > 0 && !state.currentEliteId) {
          // Auto-select first elite if none selected
          state.currentEliteId = elites[0].id;
        }
      }),

    resetToBalanced: () =>
      set((state) => {
        state.weights = DEFAULT_WEIGHTS;
        state.presetId = 'balanced';
      }),

    setLoading: (loading) =>
      set((state) => {
        state.isLoading = loading;
      }),
  }))
);

// Selectors
export const selectCurrentPreset = (state: DimensionStore): PivotPreset | null =>
  state.presetId ? getPresetById(state.presetId) || null : null;

export const selectCurrentElite = (state: DimensionStore): EliteSummary | null =>
  state.elites.find((e) => e.id === state.currentEliteId) || null;

export const selectBestEliteForWeights = (
  state: DimensionStore
): EliteSummary | null => {
  if (state.elites.length === 0) return null;

  let bestElite: EliteSummary | null = null;
  let bestScore = -1;

  for (const elite of state.elites) {
    const score = calculateWeightedScore(elite.scores, state.weights);
    if (score > bestScore) {
      bestScore = score;
      bestElite = elite;
    }
  }

  return bestElite;
};

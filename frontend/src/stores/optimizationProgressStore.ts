import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import type { DimensionScores } from '../types/protocol';
import { api, OptimizationRun } from '../services/api';

export type OptimizationStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface OptimizationProgress {
  runId: string;
  runName: string;
  status: OptimizationStatus;
  iteration: number;
  maxIterations: number;
  currentScore: number;
  bestScore: number;
  dimensionScores?: DimensionScores;
  dimensionWeights?: Record<string, number>;
  startedAt?: string;
  completedAt?: string;
  errorMessage?: string;
}

interface OptimizationProgressStoreState {
  // Progress keyed by runId
  progressByRunId: Record<string, OptimizationProgress>;
  // Loading state for fetch operations
  isLoading: boolean;
  // Error message from fetch operations
  fetchError: string | null;
}

interface OptimizationProgressStoreActions {
  // Update progress for a run
  updateProgress: (runId: string, progress: Partial<OptimizationProgress>) => void;

  // Set full progress data
  setProgress: (runId: string, progress: OptimizationProgress) => void;

  // Get progress for a run
  getProgress: (runId: string) => OptimizationProgress | undefined;

  // Remove progress tracking for a run
  removeProgress: (runId: string) => void;

  // Clear all progress
  clearAllProgress: () => void;

  // Fetch optimization run data from API
  fetchRun: (runId: string) => Promise<void>;

  // Fetch all running/pending optimization runs
  fetchRunningOptimizations: () => Promise<void>;

  // Clear fetch error
  clearFetchError: () => void;
}

type OptimizationProgressStore = OptimizationProgressStoreState & OptimizationProgressStoreActions;

const initialState: OptimizationProgressStoreState = {
  progressByRunId: {},
  isLoading: false,
  fetchError: null,
};

// Helper function to convert API OptimizationRun to OptimizationProgress
function runToProgress(run: OptimizationRun): OptimizationProgress {
  return {
    runId: run.id,
    runName: run.name,
    status: run.status as OptimizationStatus,
    iteration: run.iterations,
    maxIterations: run.max_iterations,
    currentScore: run.best_score,
    bestScore: run.best_score,
    dimensionScores: run.best_dim_scores as DimensionScores | undefined,
    dimensionWeights: run.dimension_weights,
    startedAt: run.created_at,
    completedAt: run.completed_at,
  };
}

export const useOptimizationProgressStore = create<OptimizationProgressStore>()(
  immer((set, get) => ({
    ...initialState,

    updateProgress: (runId, progress) =>
      set((state) => {
        const existing = state.progressByRunId[runId];
        if (existing) {
          state.progressByRunId[runId] = { ...existing, ...progress };
        } else {
          // Create new entry with defaults
          state.progressByRunId[runId] = {
            runId,
            runName: progress.runName || runId,
            status: progress.status || 'pending',
            iteration: progress.iteration || 0,
            maxIterations: progress.maxIterations || 0,
            currentScore: progress.currentScore || 0,
            bestScore: progress.bestScore || 0,
            ...progress,
          };
        }
      }),

    setProgress: (runId, progress) =>
      set((state) => {
        state.progressByRunId[runId] = progress;
      }),

    getProgress: (runId) => {
      return get().progressByRunId[runId];
    },

    removeProgress: (runId) =>
      set((state) => {
        delete state.progressByRunId[runId];
      }),

    clearAllProgress: () =>
      set((state) => {
        state.progressByRunId = {};
      }),

    fetchRun: async (runId: string) => {
      set((state) => {
        state.isLoading = true;
        state.fetchError = null;
      });

      try {
        const run = await api.getOptimizationRun(runId);
        const progress = runToProgress(run);
        set((state) => {
          state.progressByRunId[runId] = progress;
          state.isLoading = false;
        });
      } catch (error) {
        set((state) => {
          state.isLoading = false;
          state.fetchError = error instanceof Error ? error.message : 'Failed to fetch optimization run';
        });
      }
    },

    fetchRunningOptimizations: async () => {
      set((state) => {
        state.isLoading = true;
        state.fetchError = null;
      });

      try {
        // Fetch both running and pending runs
        const [runningRuns, pendingRuns] = await Promise.all([
          api.listOptimizationRuns({ status: 'running' }),
          api.listOptimizationRuns({ status: 'pending' }),
        ]);

        const allActiveRuns = [...runningRuns, ...pendingRuns];

        set((state) => {
          // Update progress for each active run
          for (const run of allActiveRuns) {
            state.progressByRunId[run.id] = runToProgress(run);
          }
          state.isLoading = false;
        });
      } catch (error) {
        set((state) => {
          state.isLoading = false;
          state.fetchError = error instanceof Error ? error.message : 'Failed to fetch running optimizations';
        });
      }
    },

    clearFetchError: () =>
      set((state) => {
        state.fetchError = null;
      }),
  }))
);

// Selectors
export const selectProgress = (state: OptimizationProgressStore, runId: string) =>
  state.progressByRunId[runId];

export const selectAllProgress = (state: OptimizationProgressStore) =>
  Object.values(state.progressByRunId);

export const selectRunningOptimizations = (state: OptimizationProgressStore) =>
  Object.values(state.progressByRunId).filter((p) => p.status === 'running');

export const selectIsLoading = (state: OptimizationProgressStore) => state.isLoading;

export const selectFetchError = (state: OptimizationProgressStore) => state.fetchError;

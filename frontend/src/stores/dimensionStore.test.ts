import { describe, it, expect, beforeEach } from 'vitest';
import {
  useDimensionStore,
  PIVOT_PRESETS,
  getPresetById,
  DEFAULT_WEIGHTS,
  calculateWeightedScore,
  selectCurrentPreset,
  selectCurrentElite,
  selectBestEliteForWeights,
  type PresetId,
} from './dimensionStore';
import type { EliteSummary, DimensionScores } from '../types/protocol';

describe('dimensionStore', () => {
  beforeEach(() => {
    useDimensionStore.getState().resetToBalanced();
  });

  describe('initial state', () => {
    it('should start with balanced preset', () => {
      const state = useDimensionStore.getState();
      expect(state.presetId).toBe('balanced');
      expect(state.weights).toEqual(DEFAULT_WEIGHTS);
    });

    it('should start with empty elites', () => {
      const state = useDimensionStore.getState();
      expect(state.elites).toEqual([]);
      expect(state.currentEliteId).toBeNull();
    });

    it('should not be loading initially', () => {
      const state = useDimensionStore.getState();
      expect(state.isLoading).toBe(false);
    });
  });

  describe('setPreset', () => {
    it('should set weights for accuracy preset', () => {
      useDimensionStore.getState().setPreset('accuracy');

      const state = useDimensionStore.getState();
      const accuracyPreset = getPresetById('accuracy')!;
      expect(state.presetId).toBe('accuracy');
      expect(state.weights).toEqual(accuracyPreset.weights);
      expect(state.weights.successRate).toBe(0.4);
    });

    it('should set weights for speed preset', () => {
      useDimensionStore.getState().setPreset('speed');

      const state = useDimensionStore.getState();
      const speedPreset = getPresetById('speed')!;
      expect(state.presetId).toBe('speed');
      expect(state.weights).toEqual(speedPreset.weights);
      expect(state.weights.efficiency).toBe(0.35);
    });

    it('should set weights for reliable preset', () => {
      useDimensionStore.getState().setPreset('reliable');

      const state = useDimensionStore.getState();
      const reliablePreset = getPresetById('reliable')!;
      expect(state.presetId).toBe('reliable');
      expect(state.weights).toEqual(reliablePreset.weights);
      expect(state.weights.robustness).toBe(0.3);
    });

    it('should set weights for creative preset', () => {
      useDimensionStore.getState().setPreset('creative');

      const state = useDimensionStore.getState();
      const creativePreset = getPresetById('creative')!;
      expect(state.presetId).toBe('creative');
      expect(state.weights).toEqual(creativePreset.weights);
      expect(state.weights.diversity).toBe(0.2);
      expect(state.weights.innovation).toBe(0.15);
    });

    it('should handle all preset ids', () => {
      const presetIds: PresetId[] = ['accuracy', 'speed', 'reliable', 'creative', 'balanced'];

      presetIds.forEach((presetId) => {
        useDimensionStore.getState().setPreset(presetId);
        const state = useDimensionStore.getState();
        expect(state.presetId).toBe(presetId);
      });
    });
  });

  describe('setCustomWeights', () => {
    it('should set custom weights and clear preset id', () => {
      const customWeights = {
        successRate: 0.3,
        quality: 0.2,
        efficiency: 0.2,
        robustness: 0.1,
        generalization: 0.1,
        diversity: 0.05,
        innovation: 0.05,
      };

      useDimensionStore.getState().setCustomWeights(customWeights);

      const state = useDimensionStore.getState();
      expect(state.presetId).toBeNull();
      expect(state.weights).toEqual(customWeights);
    });

    it('should normalize weights to sum to 1.0', () => {
      const unnormalizedWeights = {
        successRate: 1,
        quality: 1,
        efficiency: 1,
        robustness: 1,
        generalization: 1,
        diversity: 1,
        innovation: 1,
      };

      useDimensionStore.getState().setCustomWeights(unnormalizedWeights);

      const state = useDimensionStore.getState();
      const sum = Object.values(state.weights).reduce((a, b) => a + b, 0);
      expect(sum).toBeCloseTo(1.0, 5);
      expect(state.weights.successRate).toBeCloseTo(1 / 7, 5);
    });

    it('should handle zero sum by returning default weights', () => {
      const zeroWeights = {
        successRate: 0,
        quality: 0,
        efficiency: 0,
        robustness: 0,
        generalization: 0,
        diversity: 0,
        innovation: 0,
      };

      useDimensionStore.getState().setCustomWeights(zeroWeights);

      const state = useDimensionStore.getState();
      expect(state.weights).toEqual(DEFAULT_WEIGHTS);
    });
  });

  describe('setDimensionWeight', () => {
    it('should update a single dimension weight', () => {
      useDimensionStore.getState().setDimensionWeight('successRate', 0.5);

      const state = useDimensionStore.getState();
      expect(state.presetId).toBeNull();

      // Should normalize weights after adjustment
      const sum = Object.values(state.weights).reduce((a, b) => a + b, 0);
      expect(sum).toBeCloseTo(1.0, 5);
    });

    it('should clamp values between 0 and 1 before normalization', () => {
      useDimensionStore.getState().setDimensionWeight('quality', 1.5);
      expect(useDimensionStore.getState().weights.quality).toBeLessThanOrEqual(1);

      useDimensionStore.getState().setDimensionWeight('efficiency', -0.5);
      const state = useDimensionStore.getState();

      // After clamping to 0 and normalizing, the value should be >= 0
      expect(state.weights.efficiency).toBeGreaterThanOrEqual(0);
    });

    it('should normalize weights after adjustment', () => {
      useDimensionStore.getState().setDimensionWeight('robustness', 0.8);

      const state = useDimensionStore.getState();
      const sum = Object.values(state.weights).reduce((a, b) => a + b, 0);
      expect(sum).toBeCloseTo(1.0, 5);
    });

    it('should clear preset id when adjusting dimension', () => {
      useDimensionStore.getState().setPreset('accuracy');
      useDimensionStore.getState().setDimensionWeight('diversity', 0.3);

      const state = useDimensionStore.getState();
      expect(state.presetId).toBeNull();
    });
  });

  describe('selectElite', () => {
    it('should select an elite by id', () => {
      const elites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Elite One',
          scores: {
            successRate: 0.9,
            quality: 0.8,
            efficiency: 0.7,
            robustness: 0.85,
            generalization: 0.75,
            diversity: 0.6,
            innovation: 0.5,
          },
          description: 'Elite 1',
          bestFor: 'High success rate scenarios',
        },
        {
          id: 'elite-2',
          label: 'Elite Two',
          scores: {
            successRate: 0.85,
            quality: 0.9,
            efficiency: 0.8,
            robustness: 0.75,
            generalization: 0.8,
            diversity: 0.7,
            innovation: 0.6,
          },
          description: 'Elite 2',
          bestFor: 'High quality scenarios',
        },
      ];

      useDimensionStore.getState().updateElites(elites);
      useDimensionStore.getState().selectElite('elite-2');

      const state = useDimensionStore.getState();
      expect(state.currentEliteId).toBe('elite-2');
    });

    it('should not select non-existent elite', () => {
      const elites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Elite One',
          scores: {
            successRate: 0.9,
            quality: 0.8,
            efficiency: 0.7,
            robustness: 0.85,
            generalization: 0.75,
            diversity: 0.6,
            innovation: 0.5,
          },
          description: 'Elite 1',
          bestFor: 'High success rate scenarios',
        },
      ];

      useDimensionStore.getState().updateElites(elites, 'elite-1');
      useDimensionStore.getState().selectElite('non-existent');

      const state = useDimensionStore.getState();
      expect(state.currentEliteId).toBe('elite-1');
    });
  });

  describe('updateElites', () => {
    it('should update elites array', () => {
      const elites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Elite One',
          scores: {
            successRate: 0.9,
            quality: 0.8,
            efficiency: 0.7,
            robustness: 0.85,
            generalization: 0.75,
            diversity: 0.6,
            innovation: 0.5,
          },
          description: 'Elite 1',
          bestFor: 'High success rate scenarios',
        },
      ];

      useDimensionStore.getState().updateElites(elites);

      const state = useDimensionStore.getState();
      expect(state.elites).toEqual(elites);
    });

    it('should set current elite id when provided', () => {
      const elites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Elite One',
          scores: {
            successRate: 0.9,
            quality: 0.8,
            efficiency: 0.7,
            robustness: 0.85,
            generalization: 0.75,
            diversity: 0.6,
            innovation: 0.5,
          },
          description: 'Elite 1',
          bestFor: 'High success rate scenarios',
        },
      ];

      useDimensionStore.getState().updateElites(elites, 'elite-1');

      const state = useDimensionStore.getState();
      expect(state.currentEliteId).toBe('elite-1');
    });

    it('should auto-select first elite if none selected', () => {
      const elites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Elite One',
          scores: {
            successRate: 0.9,
            quality: 0.8,
            efficiency: 0.7,
            robustness: 0.85,
            generalization: 0.75,
            diversity: 0.6,
            innovation: 0.5,
          },
          description: 'Elite 1',
          bestFor: 'High success rate scenarios',
        },
        {
          id: 'elite-2',
          label: 'Elite Two',
          scores: {
            successRate: 0.85,
            quality: 0.9,
            efficiency: 0.8,
            robustness: 0.75,
            generalization: 0.8,
            diversity: 0.7,
            innovation: 0.6,
          },
          description: 'Elite 2',
          bestFor: 'High quality scenarios',
        },
      ];

      useDimensionStore.getState().updateElites(elites);

      const state = useDimensionStore.getState();
      expect(state.currentEliteId).toBe('elite-1');
    });

    it('should not auto-select if an elite is already selected', () => {
      const initialElites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Elite One',
          scores: {
            successRate: 0.9,
            quality: 0.8,
            efficiency: 0.7,
            robustness: 0.85,
            generalization: 0.75,
            diversity: 0.6,
            innovation: 0.5,
          },
          description: 'Elite 1',
          bestFor: 'High success rate scenarios',
        },
      ];

      useDimensionStore.getState().updateElites(initialElites, 'elite-1');

      const newElites: EliteSummary[] = [
        {
          id: 'elite-2',
          label: 'Elite Two',
          scores: {
            successRate: 0.85,
            quality: 0.9,
            efficiency: 0.8,
            robustness: 0.75,
            generalization: 0.8,
            diversity: 0.7,
            innovation: 0.6,
          },
          description: 'Elite 2',
          bestFor: 'High quality scenarios',
        },
      ];

      useDimensionStore.getState().updateElites(newElites);

      const state = useDimensionStore.getState();
      expect(state.currentEliteId).toBe('elite-1');
    });
  });

  describe('resetToBalanced', () => {
    it('should reset to balanced preset', () => {
      useDimensionStore.getState().setPreset('accuracy');
      useDimensionStore.getState().resetToBalanced();

      const state = useDimensionStore.getState();
      expect(state.presetId).toBe('balanced');
      expect(state.weights).toEqual(DEFAULT_WEIGHTS);
    });

    it('should reset custom weights to balanced', () => {
      const customWeights = {
        successRate: 0.5,
        quality: 0.2,
        efficiency: 0.1,
        robustness: 0.1,
        generalization: 0.05,
        diversity: 0.03,
        innovation: 0.02,
      };

      useDimensionStore.getState().setCustomWeights(customWeights);
      useDimensionStore.getState().resetToBalanced();

      const state = useDimensionStore.getState();
      expect(state.presetId).toBe('balanced');
      expect(state.weights).toEqual(DEFAULT_WEIGHTS);
    });
  });

  describe('setLoading', () => {
    it('should set loading state to true', () => {
      useDimensionStore.getState().setLoading(true);

      const state = useDimensionStore.getState();
      expect(state.isLoading).toBe(true);
    });

    it('should set loading state to false', () => {
      useDimensionStore.getState().setLoading(true);
      useDimensionStore.getState().setLoading(false);

      const state = useDimensionStore.getState();
      expect(state.isLoading).toBe(false);
    });
  });

  describe('calculateWeightedScore', () => {
    it('should calculate weighted score correctly', () => {
      const scores: DimensionScores = {
        successRate: 0.9,
        quality: 0.8,
        efficiency: 0.7,
        robustness: 0.85,
        generalization: 0.75,
        diversity: 0.6,
        innovation: 0.5,
      };

      const weights = {
        successRate: 0.3,
        quality: 0.2,
        efficiency: 0.15,
        robustness: 0.15,
        generalization: 0.1,
        diversity: 0.05,
        innovation: 0.05,
      };

      const score = calculateWeightedScore(scores, weights);

      const expected =
        0.9 * 0.3 +
        0.8 * 0.2 +
        0.7 * 0.15 +
        0.85 * 0.15 +
        0.75 * 0.1 +
        0.6 * 0.05 +
        0.5 * 0.05;

      expect(score).toBeCloseTo(expected, 5);
    });

    it('should return high score when all dimensions are 1.0', () => {
      const scores: DimensionScores = {
        successRate: 1.0,
        quality: 1.0,
        efficiency: 1.0,
        robustness: 1.0,
        generalization: 1.0,
        diversity: 1.0,
        innovation: 1.0,
      };

      const score = calculateWeightedScore(scores, DEFAULT_WEIGHTS);
      expect(score).toBeCloseTo(1.0, 5);
    });

    it('should return low score when all dimensions are 0', () => {
      const scores: DimensionScores = {
        successRate: 0,
        quality: 0,
        efficiency: 0,
        robustness: 0,
        generalization: 0,
        diversity: 0,
        innovation: 0,
      };

      const score = calculateWeightedScore(scores, DEFAULT_WEIGHTS);
      expect(score).toBe(0);
    });
  });

  describe('selectCurrentPreset', () => {
    it('should return current preset', () => {
      useDimensionStore.getState().setPreset('accuracy');

      const result = selectCurrentPreset(useDimensionStore.getState());
      expect(result?.id).toBe('accuracy');
      expect(result?.label).toBe('Accurate');
    });

    it('should return null when using custom weights', () => {
      const customWeights = {
        successRate: 0.5,
        quality: 0.2,
        efficiency: 0.1,
        robustness: 0.1,
        generalization: 0.05,
        diversity: 0.03,
        innovation: 0.02,
      };

      useDimensionStore.getState().setCustomWeights(customWeights);

      const result = selectCurrentPreset(useDimensionStore.getState());
      expect(result).toBeNull();
    });
  });

  describe('selectCurrentElite', () => {
    it('should return current elite', () => {
      const elites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Elite One',
          scores: {
            successRate: 0.9,
            quality: 0.8,
            efficiency: 0.7,
            robustness: 0.85,
            generalization: 0.75,
            diversity: 0.6,
            innovation: 0.5,
          },
          description: 'Elite 1',
          bestFor: 'High success rate scenarios',
        },
      ];

      useDimensionStore.getState().updateElites(elites, 'elite-1');

      const result = selectCurrentElite(useDimensionStore.getState());
      expect(result).toEqual(elites[0]);
    });

    it('should return null when no elite is selected', () => {
      // Manually set to avoid auto-selection
      useDimensionStore.setState({ elites: [], currentEliteId: null });

      const result = selectCurrentElite(useDimensionStore.getState());
      expect(result).toBeNull();
    });

    it('should return null when current elite id does not exist', () => {
      const elites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Elite One',
          scores: {
            successRate: 0.9,
            quality: 0.8,
            efficiency: 0.7,
            robustness: 0.85,
            generalization: 0.75,
            diversity: 0.6,
            innovation: 0.5,
          },
          description: 'Elite 1',
          bestFor: 'High success rate scenarios',
        },
      ];

      // Set elites but manually set a non-existent currentEliteId
      useDimensionStore.setState({ elites, currentEliteId: 'non-existent' });

      const result = selectCurrentElite(useDimensionStore.getState());
      expect(result).toBeNull();
    });
  });

  describe('selectBestEliteForWeights', () => {
    it('should return elite with highest weighted score', () => {
      const elites: EliteSummary[] = [
        {
          id: 'elite-1',
          label: 'Success Focused',
          scores: {
            successRate: 0.9,
            quality: 0.5,
            efficiency: 0.5,
            robustness: 0.5,
            generalization: 0.5,
            diversity: 0.5,
            innovation: 0.5,
          },
          description: 'High success rate',
          bestFor: 'Accuracy-critical tasks',
        },
        {
          id: 'elite-2',
          label: 'Efficiency Focused',
          scores: {
            successRate: 0.5,
            quality: 0.5,
            efficiency: 0.95,
            robustness: 0.5,
            generalization: 0.5,
            diversity: 0.5,
            innovation: 0.5,
          },
          description: 'High efficiency',
          bestFor: 'Speed-critical tasks',
        },
      ];

      useDimensionStore.getState().updateElites(elites);

      // Set speed preset which prioritizes efficiency
      useDimensionStore.getState().setPreset('speed');

      const result = selectBestEliteForWeights(useDimensionStore.getState());
      expect(result?.id).toBe('elite-2');
    });

    it('should return null when no elites exist', () => {
      // Manually set to avoid auto-selection from beforeEach
      useDimensionStore.setState({ elites: [], currentEliteId: null });

      const result = selectBestEliteForWeights(useDimensionStore.getState());
      expect(result).toBeNull();
    });

    it('should handle empty elites array', () => {
      useDimensionStore.getState().updateElites([]);

      const result = selectBestEliteForWeights(useDimensionStore.getState());
      expect(result).toBeNull();
    });
  });

  describe('PIVOT_PRESETS', () => {
    it('should have all expected presets', () => {
      expect(PIVOT_PRESETS).toHaveLength(5);

      const presetIds = PIVOT_PRESETS.map((p) => p.id);
      expect(presetIds).toContain('accuracy');
      expect(presetIds).toContain('speed');
      expect(presetIds).toContain('reliable');
      expect(presetIds).toContain('creative');
      expect(presetIds).toContain('balanced');
    });

    it('should have normalized weights for all presets', () => {
      PIVOT_PRESETS.forEach((preset) => {
        const sum = Object.values(preset.weights).reduce((a, b) => a + b, 0);
        expect(sum).toBeCloseTo(1.0, 5);
      });
    });
  });

  describe('getPresetById', () => {
    it('should return preset by id', () => {
      const preset = getPresetById('accuracy');
      expect(preset?.id).toBe('accuracy');
      expect(preset?.label).toBe('Accurate');
    });

    it('should return undefined for non-existent preset', () => {
      const preset = getPresetById('non-existent' as PresetId);
      expect(preset).toBeUndefined();
    });
  });
});

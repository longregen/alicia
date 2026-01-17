import React, { useState, useCallback } from 'react';
import { Input } from '../../atoms/Input';
import { Label } from '../../atoms/Label';
import { Textarea } from '../../atoms/Textarea';
import Button from '../../atoms/Button';
import { PromptTypeSelector, type PromptType } from '../PromptTypeSelector';
import { cls } from '../../../utils/cls';

/**
 * Request data for creating a new optimization run.
 */
export interface CreateOptimizationRequest {
  name: string;
  prompt_type: string;
  baseline_prompt?: string;
}

export interface CreateOptimizationFormProps {
  /** Callback when form is submitted */
  onSubmit: (data: CreateOptimizationRequest) => Promise<void>;
  /** Callback when form is cancelled */
  onCancel: () => void;
  /** Whether the form is disabled */
  disabled?: boolean;
  /** Whether form submission is in progress */
  submitting?: boolean;
}

/**
 * Form for creating a new optimization run.
 * Includes name input, prompt type selector, and optional baseline prompt.
 */
export const CreateOptimizationForm: React.FC<CreateOptimizationFormProps> = ({
  onSubmit,
  onCancel,
  disabled = false,
  submitting = false,
}) => {
  const [name, setName] = useState('');
  const [promptType, setPromptType] = useState<PromptType>('conversation');
  const [baselinePrompt, setBaselinePrompt] = useState('');
  const [error, setError] = useState<string | null>(null);

  const isDisabled = disabled || submitting;

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);

      // Validate required fields
      if (!name.trim()) {
        setError('Optimization name is required');
        return;
      }

      try {
        const data: CreateOptimizationRequest = {
          name: name.trim(),
          prompt_type: promptType,
        };

        // Only include baseline prompt if provided
        if (baselinePrompt.trim()) {
          data.baseline_prompt = baselinePrompt.trim();
        }

        await onSubmit(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to create optimization');
      }
    },
    [name, promptType, baselinePrompt, onSubmit]
  );

  return (
    <form
      className={cls(
        'bg-elevated border border-border rounded-lg p-5',
        isDisabled && 'opacity-60'
      )}
      onSubmit={handleSubmit}
    >
      <h3 className="m-0 mb-4 text-lg font-semibold text-foreground">
        Create Optimization Run
      </h3>

      {error && (
        <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-md mb-4 text-sm">
          {error}
        </div>
      )}

      <div className="mb-4">
        <Label htmlFor="optimization-name" className="block mb-1.5">
          Optimization Name *
        </Label>
        <Input
          id="optimization-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="My optimization run"
          disabled={isDisabled}
          required
        />
      </div>

      <div className="mb-4">
        <Label className="block mb-1.5">Prompt Type *</Label>
        <PromptTypeSelector
          value={promptType}
          onChange={setPromptType}
          disabled={isDisabled}
        />
      </div>

      <div className="mb-4">
        <Label htmlFor="baseline-prompt" className="block mb-1.5">
          Baseline Prompt (optional)
        </Label>
        <Textarea
          id="baseline-prompt"
          value={baselinePrompt}
          onChange={(e) => setBaselinePrompt(e.target.value)}
          placeholder="Enter the baseline prompt to optimize from..."
          disabled={isDisabled}
          rows={4}
          className="resize-y min-h-[100px]"
        />
        <p className="text-xs text-muted-foreground mt-1">
          If not provided, the system will use the current default prompt for this type.
        </p>
      </div>

      <div className="flex gap-3 justify-end mt-5">
        <Button
          type="button"
          variant="secondary"
          onClick={onCancel}
          disabled={isDisabled}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          variant="default"
          disabled={isDisabled}
          loading={submitting}
        >
          {submitting ? 'Creating...' : 'Create Optimization'}
        </Button>
      </div>
    </form>
  );
};

export default CreateOptimizationForm;

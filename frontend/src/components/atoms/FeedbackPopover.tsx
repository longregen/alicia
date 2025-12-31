import React, { useState } from 'react';
import { Popover, PopoverTrigger, PopoverContent } from './Popover';
import { RadioGroup, RadioGroupItem } from './RadioGroup';
import { Textarea } from './Textarea';
import { Label } from './Label';
import Button from './Button';
import { cls } from '../../utils/cls';

/**
 * FeedbackPopover atom component for collecting detailed feedback.
 *
 * Provides a popover interface with feedback type selection and optional comments.
 */

export type FeedbackType = 'helpful' | 'not-helpful' | 'incorrect' | 'harmful';

export interface FeedbackPopoverProps {
  /** Callback when feedback is submitted */
  onSubmit: (feedback: { type: FeedbackType; comment?: string }) => void;
  /** Whether the submission is being processed */
  isLoading?: boolean;
  /** Additional CSS classes for trigger button */
  className?: string;
  /** Button variant - 'icon' for icon-only, 'default' for text button */
  variant?: 'icon' | 'default';
  /** Label for the trigger button (used in default variant) */
  triggerLabel?: string;
  /** Whether the popover is open (controlled) */
  open?: boolean;
  /** Callback when popover open state changes (controlled) */
  onOpenChange?: (open: boolean) => void;
}

const feedbackOptions: { value: FeedbackType; label: string }[] = [
  { value: 'helpful', label: 'Helpful' },
  { value: 'not-helpful', label: 'Not helpful' },
  { value: 'incorrect', label: 'Incorrect' },
  { value: 'harmful', label: 'Harmful' },
];

const FeedbackPopover: React.FC<FeedbackPopoverProps> = ({
  onSubmit,
  isLoading = false,
  className = '',
  variant = 'icon',
  triggerLabel = 'Give Feedback',
  open,
  onOpenChange,
}) => {
  const [selectedType, setSelectedType] = useState<FeedbackType | ''>('');
  const [comment, setComment] = useState('');
  const [internalOpen, setInternalOpen] = useState(false);

  const isOpen = open !== undefined ? open : internalOpen;
  const setIsOpen = onOpenChange || setInternalOpen;

  const handleSubmit = () => {
    if (selectedType) {
      onSubmit({
        type: selectedType as FeedbackType,
        comment: comment.trim() || undefined,
      });

      // Reset form
      setSelectedType('');
      setComment('');
      setIsOpen(false);
    }
  };

  const handleCancel = () => {
    setSelectedType('');
    setComment('');
    setIsOpen(false);
  };

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        {variant === 'icon' ? (
          <button
            aria-label="Give feedback"
            className={cls(
              'flex items-center justify-center rounded-md transition-colors',
              'text-muted hover:text-default hover:bg-accent',
              'size-8',
              className
            )}
          >
            <svg
              className="w-4 h-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"
              />
            </svg>
          </button>
        ) : (
          <Button variant="outline" size="sm" className={className}>
            {triggerLabel}
          </Button>
        )}
      </PopoverTrigger>

      <PopoverContent className="w-80" align="start">
        <div className="space-y-4">
          <div>
            <h3 className="font-medium text-sm mb-3">Provide Feedback</h3>

            <RadioGroup
              value={selectedType}
              onValueChange={(value) => setSelectedType(value as FeedbackType)}
            >
              {feedbackOptions.map((option) => (
                <div key={option.value} className="flex items-center space-x-2">
                  <RadioGroupItem value={option.value} id={option.value} />
                  <Label
                    htmlFor={option.value}
                    className="text-sm font-normal cursor-pointer"
                  >
                    {option.label}
                  </Label>
                </div>
              ))}
            </RadioGroup>
          </div>

          <div>
            <Label htmlFor="feedback-comment" className="text-sm mb-2 block">
              Additional comments (optional)
            </Label>
            <Textarea
              id="feedback-comment"
              placeholder="Tell us more..."
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              className="min-h-20 resize-none"
              disabled={isLoading}
            />
          </div>

          <div className="flex gap-2 justify-end">
            <Button
              variant="outline"
              size="sm"
              onClick={handleCancel}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleSubmit}
              disabled={!selectedType || isLoading}
              loading={isLoading}
            >
              Submit
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};

export default FeedbackPopover;

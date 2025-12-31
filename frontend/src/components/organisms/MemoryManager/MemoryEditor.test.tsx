import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import { MemoryEditor } from './MemoryEditor';
import type { Memory } from '../../../stores/memoryStore';

describe('MemoryEditor', () => {
  const mockMemory: Memory = {
    id: 'memory-1',
    content: 'Test memory content',
    category: 'fact',
    pinned: false,
    archived: false,
    createdAt: Date.now(),
    updatedAt: Date.now(),
    tags: [],
    importance: 0.5,
    usageCount: 0,
  };

  const mockOnSave = vi.fn();
  const mockOnCancel = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders modal when open', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByText('Create Memory')).toBeInTheDocument();
    });

    it('does not render when closed', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={false}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.queryByText('Create Memory')).not.toBeInTheDocument();
    });

    it('renders with custom className', () => {
      const { container } = render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          className="custom-class"
        />
      );

      const modal = container.querySelector('.custom-class');
      expect(modal).toBeInTheDocument();
    });
  });

  describe('Create Mode', () => {
    it('shows "Create Memory" title in create mode', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByText('Create Memory')).toBeInTheDocument();
      expect(screen.queryByText('Edit Memory')).not.toBeInTheDocument();
    });

    it('shows "Create" button in create mode', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByRole('button', { name: /create/i })).toBeInTheDocument();
    });

    it('has empty content by default', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      expect(textarea).toHaveValue('');
    });

    it('has "fact" category selected by default', () => {
      const { container } = render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const factButton = Array.from(container.querySelectorAll('button')).find(btn =>
        btn.textContent?.includes('Fact') && btn.textContent?.includes('Factual information'));
      expect(factButton).toHaveClass('bg-accent-subtle');
    });
  });

  describe('Edit Mode', () => {
    it('shows "Edit Memory" title in edit mode', () => {
      render(
        <MemoryEditor
          memory={mockMemory}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByText('Edit Memory')).toBeInTheDocument();
      expect(screen.queryByText('Create Memory')).not.toBeInTheDocument();
    });

    it('shows "Update" button in edit mode', () => {
      render(
        <MemoryEditor
          memory={mockMemory}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByRole('button', { name: /update/i })).toBeInTheDocument();
    });

    it('populates content from memory', () => {
      render(
        <MemoryEditor
          memory={mockMemory}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      expect(textarea).toHaveValue('Test memory content');
    });

    it('selects category from memory', () => {
      const preferenceMemory: Memory = {
        ...mockMemory,
        category: 'preference',
      };

      const { container } = render(
        <MemoryEditor
          memory={preferenceMemory}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const preferenceButton = Array.from(container.querySelectorAll('button')).find(btn =>
        btn.textContent?.includes('Preference'));
      expect(preferenceButton).toHaveClass('bg-accent-subtle');
    });
  });

  describe('Category Selection', () => {
    it('renders all category options', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByText('Preference')).toBeInTheDocument();
      expect(screen.getByText('Fact')).toBeInTheDocument();
      expect(screen.getByText('Context')).toBeInTheDocument();
      expect(screen.getByText('Instruction')).toBeInTheDocument();
    });

    it('shows category descriptions', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByText('User preferences and settings')).toBeInTheDocument();
      expect(screen.getByText('Factual information about the user')).toBeInTheDocument();
      expect(screen.getByText('Contextual information for conversations')).toBeInTheDocument();
      expect(screen.getByText('Instructions for the assistant')).toBeInTheDocument();
    });

    it('allows changing category', async () => {
      const user = userEvent.setup();
      const { container } = render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const preferenceButton = Array.from(container.querySelectorAll('button')).find(btn =>
        btn.textContent?.includes('Preference'));

      await user.click(preferenceButton!);

      expect(preferenceButton).toHaveClass('bg-accent-subtle');
    });
  });

  describe('Form Interactions', () => {
    it('allows typing in content textarea', async () => {
      const user = userEvent.setup();
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      await user.type(textarea, 'New memory content');

      expect(textarea).toHaveValue('New memory content');
    });

    it('disables submit button when content is empty', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const submitButton = screen.getByRole('button', { name: /create/i });
      expect(submitButton).toBeDisabled();
    });

    it('enables submit button when content is provided', async () => {
      const user = userEvent.setup();
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      await user.type(textarea, 'New content');

      const submitButton = screen.getByRole('button', { name: /create/i });
      expect(submitButton).toBeEnabled();
    });

    it('disables submit button for whitespace-only content', async () => {
      const user = userEvent.setup();
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      await user.type(textarea, '   ');

      const submitButton = screen.getByRole('button', { name: /create/i });
      expect(submitButton).toBeDisabled();
    });
  });

  describe('Save Action', () => {
    it('calls onSave with trimmed content and category', async () => {
      const user = userEvent.setup();
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      await user.type(textarea, '  Test content  ');

      const submitButton = screen.getByRole('button', { name: /create/i });
      await user.click(submitButton);

      expect(mockOnSave).toHaveBeenCalledWith('Test content', 'fact');
    });

    it('calls onSave with selected category', async () => {
      const user = userEvent.setup();
      const { container } = render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      await user.type(textarea, 'Test content');

      const preferenceButton = Array.from(container.querySelectorAll('button')).find(btn =>
        btn.textContent?.includes('Preference'));
      await user.click(preferenceButton!);

      const submitButton = screen.getByRole('button', { name: /create/i });
      await user.click(submitButton);

      expect(mockOnSave).toHaveBeenCalledWith('Test content', 'preference');
    });

    it('does not call onSave when content is empty', async () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const submitButton = screen.getByRole('button', { name: /create/i });

      // Button should be disabled, but try to submit anyway
      expect(submitButton).toBeDisabled();
      expect(mockOnSave).not.toHaveBeenCalled();
    });
  });

  describe('Cancel Action', () => {
    it('calls onCancel when cancel button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      expect(mockOnCancel).toHaveBeenCalled();
    });

    it('calls onCancel when close button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const closeButton = screen.getByLabelText('Close');
      await user.click(closeButton);

      expect(mockOnCancel).toHaveBeenCalled();
    });

    it('calls onCancel when backdrop is clicked', async () => {
      const user = userEvent.setup();
      const { container } = render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const backdrop = container.querySelector('.fixed.inset-0.bg-overlay');
      await user.click(backdrop!);

      expect(mockOnCancel).toHaveBeenCalled();
    });

    it('calls onCancel when Escape key is pressed', async () => {
      const user = userEvent.setup();
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      await user.keyboard('{Escape}');

      expect(mockOnCancel).toHaveBeenCalled();
    });
  });

  describe('Loading State', () => {
    it('shows loading spinner when isLoading is true', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          isLoading={true}
        />
      );

      const spinner = screen.getByRole('button', { name: /create/i }).querySelector('svg.animate-spin');
      expect(spinner).toBeInTheDocument();
    });

    it('disables cancel button when loading', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          isLoading={true}
        />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      expect(cancelButton).toBeDisabled();
    });

    it('disables submit button when loading', async () => {
      const user = userEvent.setup();
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          isLoading={true}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      await user.type(textarea, 'Test content');

      const submitButton = screen.getByRole('button', { name: /create/i });
      expect(submitButton).toBeDisabled();
    });
  });

  describe('Form Reset', () => {
    it('resets form when memory changes from null to populated', () => {
      const { rerender } = render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      expect(textarea).toHaveValue('');

      rerender(
        <MemoryEditor
          memory={mockMemory}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(textarea).toHaveValue('Test memory content');
    });

    it('resets form when memory changes from populated to null', () => {
      const { rerender } = render(
        <MemoryEditor
          memory={mockMemory}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');
      expect(textarea).toHaveValue('Test memory content');

      rerender(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(textarea).toHaveValue('');
    });
  });

  describe('Accessibility', () => {
    it('focuses textarea on open', async () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter memory content...');

      await waitFor(() => {
        expect(textarea).toHaveFocus();
      });
    });

    it('has proper labels for form fields', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByLabelText('Content')).toBeInTheDocument();
      expect(screen.getByText('Category')).toBeInTheDocument();
    });

    it('has aria-label for close button', () => {
      render(
        <MemoryEditor
          memory={null}
          isOpen={true}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
        />
      );

      expect(screen.getByLabelText('Close')).toBeInTheDocument();
    });
  });
});

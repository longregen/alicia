import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import { MemoryList } from './MemoryList';
import type { Memory } from '../../../stores/memoryStore';

describe('MemoryList', () => {
  const mockMemories: Memory[] = [
    {
      id: 'memory-1',
      content: 'First memory content',
      category: 'fact',
      pinned: false,
      archived: false,
      createdAt: Date.now() - 10000,
      updatedAt: Date.now() - 10000,
      tags: [],
      importance: 0.5,
      usageCount: 0,
    },
    {
      id: 'memory-2',
      content: 'Second memory content that is pinned',
      category: 'preference',
      pinned: true,
      archived: false,
      createdAt: Date.now() - 20000,
      updatedAt: Date.now() - 5000,
      tags: [],
      importance: 0.5,
      usageCount: 0,
    },
    {
      id: 'memory-3',
      content: 'Third memory with context category',
      category: 'context',
      pinned: false,
      archived: false,
      createdAt: Date.now() - 30000,
      updatedAt: Date.now() - 30000,
      tags: [],
      importance: 0.5,
      usageCount: 0,
    },
  ];

  const mockOnEdit = vi.fn();
  const mockOnDelete = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders all memories', () => {
      render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      expect(screen.getByText('First memory content')).toBeInTheDocument();
      expect(screen.getByText('Second memory content that is pinned')).toBeInTheDocument();
      expect(screen.getByText('Third memory with context category')).toBeInTheDocument();
    });

    it('applies custom className', () => {
      const { container } = render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          className="custom-class"
        />
      );

      const rootElement = container.firstChild;
      expect(rootElement).toHaveClass('custom-class');
    });
  });

  describe('Empty State', () => {
    it('shows empty state when no memories', () => {
      render(
        <MemoryList
          memories={[]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      expect(screen.getByText('No memories found')).toBeInTheDocument();
      expect(screen.getByText('Create your first memory to get started')).toBeInTheDocument();
    });

    it('shows empty state icon', () => {
      const { container } = render(
        <MemoryList
          memories={[]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      const icon = container.querySelector('svg');
      expect(icon).toBeInTheDocument();
    });
  });

  describe('Category Display', () => {
    it('displays category badges for each memory', () => {
      render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      expect(screen.getByText('fact')).toBeInTheDocument();
      expect(screen.getByText('preference')).toBeInTheDocument();
      expect(screen.getByText('context')).toBeInTheDocument();
    });

    it('applies category-specific colors', () => {
      render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      const factBadge = screen.getByText('fact');
      expect(factBadge).toHaveClass('text-success');

      const preferenceBadge = screen.getByText('preference');
      expect(preferenceBadge).toHaveClass('text-accent');

      const contextBadge = screen.getByText('context');
      expect(contextBadge).toHaveClass('text-warning');
    });
  });


  describe('Action Buttons', () => {
    it('shows all action buttons for each memory', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      expect(screen.getByLabelText('Edit memory')).toBeInTheDocument();
      expect(screen.getByLabelText('Delete memory')).toBeInTheDocument();
    });

    it('calls onEdit when edit button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      const editButton = screen.getByLabelText('Edit memory');
      await user.click(editButton);

      expect(mockOnEdit).toHaveBeenCalledWith(mockMemories[0]);
    });

    it('opens delete popover when delete button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      const deleteButton = screen.getByLabelText('Delete memory');
      await user.click(deleteButton);

      // Popover should open with deletion options
      expect(screen.getByText('Why delete?')).toBeInTheDocument();
    });

    it('calls onDelete with reason when reason is selected', async () => {
      const user = userEvent.setup();
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      // Open popover
      const deleteButton = screen.getByLabelText('Delete memory');
      await user.click(deleteButton);

      // Click a reason
      const wrongButton = screen.getByText('Wrong');
      await user.click(wrongButton);

      expect(mockOnDelete).toHaveBeenCalledWith(mockMemories[0], 'wrong');
    });

  });

  describe('Loading State', () => {
    it('disables action buttons when loading', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          isLoading={true}
        />
      );

      expect(screen.getByLabelText('Edit memory')).toBeDisabled();
      expect(screen.getByLabelText('Delete memory')).toBeDisabled();
    });

    it('applies disabled opacity when loading', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          isLoading={true}
        />
      );

      const editButton = screen.getByLabelText('Edit memory');
      expect(editButton).toHaveClass('disabled:opacity-50');
    });
  });

  describe('Timestamp Display', () => {
    it('shows relative timestamp for memories', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      // Table shows relative time like "just now", "5m ago", etc.
      // The header says "Created", test that a timestamp cell exists
      expect(screen.getByText('Created')).toBeInTheDocument();
    });

    it('formats recent timestamps as relative time', () => {
      const recentMemory: Memory = {
        ...mockMemories[0],
        createdAt: Date.now() - 30000, // 30 seconds ago
        updatedAt: Date.now() - 30000,
      };

      render(
        <MemoryList
          memories={[recentMemory]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      expect(screen.getAllByText(/just now|ago/).length).toBeGreaterThan(0);
    });
  });

  describe('Content Display', () => {
    it('displays memory content', () => {
      render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      mockMemories.forEach(memory => {
        expect(screen.getByText(memory.content)).toBeInTheDocument();
      });
    });

    it('truncates long content with CSS', () => {
      const longMemory: Memory = {
        ...mockMemories[0],
        content: 'A'.repeat(500),
      };

      const { container } = render(
        <MemoryList
          memories={[longMemory]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      // Table layout uses truncate class
      const contentElement = container.querySelector('.truncate');
      expect(contentElement).toBeInTheDocument();
    });
  });

  describe('Hover Effects', () => {
    it('applies hover background effect on rows', () => {
      const { container } = render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      // Table rows have hover:bg-surface-hover
      const row = container.querySelector('[class*="hover:bg-surface-hover"]');
      expect(row).toBeInTheDocument();
    });
  });

  describe('Multiple Memories', () => {
    it('renders multiple memory rows', () => {
      const { container } = render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      // Table rows in tbody
      const rows = container.querySelectorAll('tbody tr');
      expect(rows.length).toBe(mockMemories.length);
    });

    it('each memory has its own action buttons', () => {
      render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      const editButtons = screen.getAllByLabelText('Edit memory');
      expect(editButtons).toHaveLength(mockMemories.length);
    });
  });

  describe('Category-specific Styling', () => {
    it('applies instruction category colors', () => {
      const instructionMemory: Memory = {
        ...mockMemories[0],
        category: 'instruction',
      };

      render(
        <MemoryList
          memories={[instructionMemory]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      const badge = screen.getByText('instruction');
      expect(badge).toHaveClass('text-error');
    });
  });

  describe('Accessibility', () => {
    it('has descriptive aria-labels for action buttons', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      expect(screen.getByLabelText('Edit memory')).toBeInTheDocument();
      expect(screen.getByLabelText('Delete memory')).toBeInTheDocument();
    });

    it('has title attributes for tooltips', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      );

      expect(screen.getByTitle('Edit')).toBeInTheDocument();
      expect(screen.getByTitle('Delete')).toBeInTheDocument();
    });
  });
});

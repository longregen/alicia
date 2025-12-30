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
    },
    {
      id: 'memory-2',
      content: 'Second memory content that is pinned',
      category: 'preference',
      pinned: true,
      archived: false,
      createdAt: Date.now() - 20000,
      updatedAt: Date.now() - 5000,
    },
    {
      id: 'memory-3',
      content: 'Third memory with context category',
      category: 'context',
      pinned: false,
      archived: false,
      createdAt: Date.now() - 30000,
      updatedAt: Date.now() - 30000,
    },
  ];

  const mockOnEdit = vi.fn();
  const mockOnDelete = vi.fn();
  const mockOnPin = vi.fn();
  const mockOnArchive = vi.fn();

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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const factBadge = screen.getByText('fact');
      expect(factBadge).toHaveClass('text-blue-700');

      const preferenceBadge = screen.getByText('preference');
      expect(preferenceBadge).toHaveClass('text-purple-700');

      const contextBadge = screen.getByText('context');
      expect(contextBadge).toHaveClass('text-green-700');
    });
  });

  describe('Pinned Indicator', () => {
    it('shows pin icon for pinned memories', () => {
      const { container } = render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const memoryCards = container.querySelectorAll('[class*="ring-2"]');
      expect(memoryCards.length).toBeGreaterThan(0);
    });

    it('does not show pin icon for unpinned memories', () => {
      const unpinnedMemories = mockMemories.filter(m => !m.pinned);

      render(
        <MemoryList
          memories={unpinnedMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const pinIcons = screen.queryAllByTitle('Unpin');
      expect(pinIcons.length).toBe(0);
    });

    it('applies ring styling to pinned memories', () => {
      const pinnedOnly = mockMemories.filter(m => m.pinned);

      const { container } = render(
        <MemoryList
          memories={pinnedOnly}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const card = container.querySelector('[class*="ring-2"]');
      expect(card).toHaveClass('ring-primary-blue');
    });
  });

  describe('Action Buttons', () => {
    it('shows all action buttons for each memory', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      expect(screen.getByLabelText('Pin memory')).toBeInTheDocument();
      expect(screen.getByLabelText('Edit memory')).toBeInTheDocument();
      expect(screen.getByLabelText('Archive memory')).toBeInTheDocument();
      expect(screen.getByLabelText('Delete memory')).toBeInTheDocument();
    });

    it('calls onEdit when edit button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const editButton = screen.getByLabelText('Edit memory');
      await user.click(editButton);

      expect(mockOnEdit).toHaveBeenCalledWith(mockMemories[0]);
    });

    it('calls onDelete when delete button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const deleteButton = screen.getByLabelText('Delete memory');
      await user.click(deleteButton);

      expect(mockOnDelete).toHaveBeenCalledWith(mockMemories[0]);
    });

    it('calls onPin when pin button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const pinButton = screen.getByLabelText('Pin memory');
      await user.click(pinButton);

      expect(mockOnPin).toHaveBeenCalledWith(mockMemories[0]);
    });

    it('calls onArchive when archive button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const archiveButton = screen.getByLabelText('Archive memory');
      await user.click(archiveButton);

      expect(mockOnArchive).toHaveBeenCalledWith(mockMemories[0]);
    });

    it('shows "Unpin" label for pinned memories', () => {
      const pinnedMemory = mockMemories.filter(m => m.pinned);

      render(
        <MemoryList
          memories={pinnedMemory}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      expect(screen.getByLabelText('Unpin memory')).toBeInTheDocument();
    });
  });

  describe('Loading State', () => {
    it('disables action buttons when loading', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
          isLoading={true}
        />
      );

      expect(screen.getByLabelText('Pin memory')).toBeDisabled();
      expect(screen.getByLabelText('Edit memory')).toBeDisabled();
      expect(screen.getByLabelText('Archive memory')).toBeDisabled();
      expect(screen.getByLabelText('Delete memory')).toBeDisabled();
    });

    it('applies disabled opacity when loading', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
          isLoading={true}
        />
      );

      const pinButton = screen.getByLabelText('Pin memory');
      expect(pinButton).toHaveClass('disabled:opacity-50');
    });
  });

  describe('Timestamp Display', () => {
    it('shows creation time for new memories', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      expect(screen.getByText(/Created/)).toBeInTheDocument();
    });

    it('shows updated time when memory was updated', () => {
      const updatedMemory: Memory = {
        ...mockMemories[0],
        updatedAt: Date.now() - 1000,
      };

      render(
        <MemoryList
          memories={[updatedMemory]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      expect(screen.getByText(/Updated/)).toBeInTheDocument();
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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      mockMemories.forEach(memory => {
        expect(screen.getByText(memory.content)).toBeInTheDocument();
      });
    });

    it('truncates long content with line-clamp', () => {
      const longMemory: Memory = {
        ...mockMemories[0],
        content: 'A'.repeat(500),
      };

      const { container } = render(
        <MemoryList
          memories={[longMemory]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const contentElement = container.querySelector('.line-clamp-3');
      expect(contentElement).toBeInTheDocument();
    });
  });

  describe('Hover Effects', () => {
    it('applies hover shadow effect', () => {
      const { container } = render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const card = container.querySelector('[class*="hover:shadow-md"]');
      expect(card).toBeInTheDocument();
    });
  });

  describe('Multiple Memories', () => {
    it('renders multiple memory cards', () => {
      const { container } = render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const cards = container.querySelectorAll('[class*="border-surface-300"]');
      expect(cards.length).toBe(mockMemories.length);
    });

    it('each memory has its own action buttons', () => {
      render(
        <MemoryList
          memories={mockMemories}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
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
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      const badge = screen.getByText('instruction');
      expect(badge).toHaveClass('text-orange-700');
    });
  });

  describe('Accessibility', () => {
    it('has descriptive aria-labels for action buttons', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      expect(screen.getByLabelText('Pin memory')).toBeInTheDocument();
      expect(screen.getByLabelText('Edit memory')).toBeInTheDocument();
      expect(screen.getByLabelText('Archive memory')).toBeInTheDocument();
      expect(screen.getByLabelText('Delete memory')).toBeInTheDocument();
    });

    it('has title attributes for tooltips', () => {
      render(
        <MemoryList
          memories={[mockMemories[0]]}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
          onPin={mockOnPin}
          onArchive={mockOnArchive}
        />
      );

      expect(screen.getByTitle('Pin')).toBeInTheDocument();
      expect(screen.getByTitle('Edit')).toBeInTheDocument();
      expect(screen.getByTitle('Archive')).toBeInTheDocument();
      expect(screen.getByTitle('Delete')).toBeInTheDocument();
    });
  });
});

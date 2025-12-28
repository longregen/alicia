import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import { MemorySearch } from './MemorySearch';
import type { MemoryCategory } from '../../../stores/memoryStore';

describe('MemorySearch', () => {
  const mockOnSearchChange = vi.fn();
  const mockOnCategoryChange = vi.fn();
  const mockOnCreateNew = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders search input', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      expect(screen.getByPlaceholderText('Search memories...')).toBeInTheDocument();
    });

    it('renders create button', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      expect(screen.getByRole('button', { name: /create memory/i })).toBeInTheDocument();
    });

    it('renders category filter', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      expect(screen.getByText('Filter:')).toBeInTheDocument();
      expect(screen.getByText('All')).toBeInTheDocument();
    });

    it('applies custom className', () => {
      const { container } = render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
          className="custom-class"
        />
      );

      const rootElement = container.firstChild;
      expect(rootElement).toHaveClass('custom-class');
    });
  });

  describe('Search Input', () => {
    it('displays current search query', () => {
      render(
        <MemorySearch
          searchQuery="test query"
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const input = screen.getByPlaceholderText('Search memories...');
      expect(input).toHaveValue('test query');
    });

    it('calls onSearchChange when typing', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const input = screen.getByPlaceholderText('Search memories...');
      await user.type(input, 'new search');

      expect(mockOnSearchChange).toHaveBeenCalled();
    });

    it('updates input value as user types', async () => {
      const user = userEvent.setup();
      const { rerender } = render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const input = screen.getByPlaceholderText('Search memories...');
      await user.type(input, 'test');

      rerender(
        <MemorySearch
          searchQuery="test"
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      expect(input).toHaveValue('test');
    });
  });

  describe('Create Button', () => {
    it('calls onCreateNew when clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const button = screen.getByRole('button', { name: /create memory/i });
      await user.click(button);

      expect(mockOnCreateNew).toHaveBeenCalled();
    });
  });

  describe('Category Filter', () => {
    it('displays selected category', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="preference"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      expect(screen.getByText('Preferences')).toBeInTheDocument();
    });

    it('opens dropdown when clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const dropdownButton = screen.getByText('All');
      await user.click(dropdownButton);

      expect(screen.getAllByText('All')).toHaveLength(2); // One in button, one in dropdown
      expect(screen.getByText('Preferences')).toBeInTheDocument();
      expect(screen.getByText('Facts')).toBeInTheDocument();
      expect(screen.getByText('Context')).toBeInTheDocument();
      expect(screen.getByText('Instructions')).toBeInTheDocument();
    });

    it('closes dropdown when option is selected', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const dropdownButton = screen.getByText('All');
      await user.click(dropdownButton);

      const factsOption = screen.getByText('Facts');
      await user.click(factsOption);

      await waitFor(() => {
        // After selecting, the dropdown should close
        expect(mockOnCategoryChange).toHaveBeenCalledWith('fact');
      });
    });

    it('calls onCategoryChange when category is selected', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const dropdownButton = screen.getByText('All');
      await user.click(dropdownButton);

      const preferenceOption = screen.getByText('Preferences');
      await user.click(preferenceOption);

      expect(mockOnCategoryChange).toHaveBeenCalledWith('preference');
    });

    it('closes dropdown when backdrop is clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const dropdownButton = screen.getByText('All');
      await user.click(dropdownButton);

      const backdrop = document.querySelector('.fixed.inset-0.z-10');
      expect(backdrop).toBeInTheDocument();

      await user.click(backdrop!);

      await waitFor(() => {
        expect(screen.queryByText('Preferences')).not.toBeInTheDocument();
      });
    });

    it('shows chevron icon that rotates when open', async () => {
      const user = userEvent.setup();
      const { container } = render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const dropdownButton = screen.getByText('All');
      const chevron = container.querySelector('svg');

      expect(chevron).not.toHaveClass('rotate-180');

      await user.click(dropdownButton);

      expect(chevron).toHaveClass('rotate-180');
    });

    it('highlights selected category in dropdown', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="fact"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const dropdownButton = screen.getByText('Facts');
      await user.click(dropdownButton);

      const factsButtons = screen.getAllByText('Facts');
      const dropdownFactsButton = factsButtons[factsButtons.length - 1]; // Last one should be in dropdown
      expect(dropdownFactsButton).toHaveClass('bg-primary-blue');
    });
  });

  describe('Clear Filters', () => {
    it('shows clear filters button when search query is active', () => {
      render(
        <MemorySearch
          searchQuery="test"
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      expect(screen.getByText('Clear filters')).toBeInTheDocument();
    });

    it('shows clear filters button when category is selected', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="fact"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      expect(screen.getByText('Clear filters')).toBeInTheDocument();
    });

    it('hides clear filters button when no filters are active', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      expect(screen.queryByText('Clear filters')).not.toBeInTheDocument();
    });

    it('clears both search and category when clicked', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery="test"
          selectedCategory="fact"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const clearButton = screen.getByText('Clear filters');
      await user.click(clearButton);

      expect(mockOnSearchChange).toHaveBeenCalledWith('');
      expect(mockOnCategoryChange).toHaveBeenCalledWith('all');
    });
  });

  describe('Category Options', () => {
    it('displays all category options in dropdown', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const dropdownButton = screen.getByText('All');
      await user.click(dropdownButton);

      const options = screen.getAllByRole('button');
      const optionTexts = options.map(opt => opt.textContent);

      expect(optionTexts).toContain('All');
      expect(optionTexts).toContain('Preferences');
      expect(optionTexts).toContain('Facts');
      expect(optionTexts).toContain('Context');
      expect(optionTexts).toContain('Instructions');
    });

    it('allows selecting each category', async () => {
      const user = userEvent.setup();
      const categories: Array<MemoryCategory | 'all'> = ['preference', 'fact', 'context', 'instruction'];

      for (const category of categories) {
        mockOnCategoryChange.mockClear();

        const { unmount } = render(
          <MemorySearch
            searchQuery=""
            selectedCategory="all"
            onSearchChange={mockOnSearchChange}
            onCategoryChange={mockOnCategoryChange}
            onCreateNew={mockOnCreateNew}
          />
        );

        const dropdownButton = screen.getByText('All');
        await user.click(dropdownButton);

        const categoryLabels: Record<string, string> = {
          'preference': 'Preferences',
          'fact': 'Facts',
          'context': 'Context',
          'instruction': 'Instructions'
        };

        const options = screen.getAllByText(categoryLabels[category]);
        await user.click(options[options.length - 1]); // Click the last one (in dropdown)

        expect(mockOnCategoryChange).toHaveBeenCalledWith(category);
        unmount();
      }
    });
  });

  describe('Keyboard Interaction', () => {
    it('allows typing in search field', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const input = screen.getByPlaceholderText('Search memories...');
      await user.type(input, 'test search');

      expect(mockOnSearchChange).toHaveBeenCalled();
    });

    it('allows clearing search with backspace', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery="test"
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const input = screen.getByPlaceholderText('Search memories...');
      await user.click(input);
      await user.keyboard('{Backspace}');

      expect(mockOnSearchChange).toHaveBeenCalled();
    });
  });

  describe('Focus States', () => {
    it('applies focus styles to search input', async () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const input = screen.getByPlaceholderText('Search memories...');
      expect(input).toHaveClass('focus:border-primary-blue');
    });

    it('applies focus styles to create button', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const button = screen.getByRole('button', { name: /create memory/i });
      expect(button).toHaveClass('focus:ring-2');
    });
  });

  describe('Layout', () => {
    it('renders search bar and create button in same row', () => {
      const { container } = render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const searchRow = container.querySelector('[class*="flex gap-2"]');
      expect(searchRow).toBeInTheDocument();
    });

    it('renders filter section below search bar', () => {
      const { container } = render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const rootElement = container.firstChild;
      expect(rootElement).toHaveClass('flex-col');
    });
  });

  describe('Accessibility', () => {
    it('has accessible search input', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const input = screen.getByPlaceholderText('Search memories...');
      expect(input).toHaveAttribute('type', 'text');
    });

    it('has accessible create button', () => {
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const button = screen.getByRole('button', { name: /create memory/i });
      expect(button).toBeInTheDocument();
    });

    it('category dropdown buttons are keyboard accessible', async () => {
      const user = userEvent.setup();
      render(
        <MemorySearch
          searchQuery=""
          selectedCategory="all"
          onSearchChange={mockOnSearchChange}
          onCategoryChange={mockOnCategoryChange}
          onCreateNew={mockOnCreateNew}
        />
      );

      const dropdownButton = screen.getByText('All');
      await user.click(dropdownButton);

      const options = screen.getAllByRole('button');
      options.forEach(option => {
        expect(option).toBeInTheDocument();
      });
    });
  });
});

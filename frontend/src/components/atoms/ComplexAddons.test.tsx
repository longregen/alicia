import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import ComplexAddons from './ComplexAddons';
import type { MessageAddon } from '../../types/components';
import type { ToolDetail } from './ComplexAddons';

describe('ComplexAddons', () => {
  const mockTimestamp = new Date('2024-01-01T12:00:00');

  const mockAddons: MessageAddon[] = [
    {
      id: 'tool-1',
      type: 'tool',
      emoji: '🔧',
      tooltip: 'Search Tool'
    },
    {
      id: 'tool-2',
      type: 'icon',
      emoji: '📊',
      tooltip: 'Analytics'
    }
  ];

  const mockToolDetails: ToolDetail[] = [
    {
      id: 'tool-1',
      name: 'Search Tool',
      description: 'Searches for relevant information',
      status: 'completed',
      result: 'Found 5 results'
    },
    {
      id: 'tool-2',
      name: 'Analytics Tool',
      description: 'Analyzes data patterns',
      status: 'pending'
    }
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders addons with emojis', () => {
      render(
        <ComplexAddons
          addons={mockAddons}
          timestamp={mockTimestamp}
        />
      );

      expect(screen.getByText('🔧')).toBeInTheDocument();
      expect(screen.getByText('📊')).toBeInTheDocument();
    });

    it('renders timestamp correctly', () => {
      render(
        <ComplexAddons
          addons={mockAddons}
          timestamp={mockTimestamp}
        />
      );

      expect(screen.getByText('12:00 PM')).toBeInTheDocument();
    });

    it('applies custom className', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          timestamp={mockTimestamp}
          className="custom-class"
        />
      );

      const rootElement = container.firstChild;
      expect(rootElement).toHaveClass('custom-class');
    });

    it('handles empty addons array', () => {
      const { container } = render(
        <ComplexAddons
          addons={[]}
          timestamp={mockTimestamp}
        />
      );

      expect(container.querySelector('button')).not.toBeInTheDocument();
    });
  });

  describe('Tool Status Icon Rendering', () => {
    it('renders pending status icon', () => {
      const pendingTool: ToolDetail[] = [{
        id: 'tool-pending',
        name: 'Pending Tool',
        description: 'Tool in pending state',
        status: 'pending'
      }];

      const addons: MessageAddon[] = [{
        id: 'tool-pending',
        type: 'tool',
        emoji: '⏳',
        tooltip: 'Pending Tool'
      }];

      const { container } = render(
        <ComplexAddons
          addons={addons}
          toolDetails={pendingTool}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Pending Tool"]');
      expect(button).toHaveClass('opacity-70');
    });

    it('renders running status with pulse animation and indicator', () => {
      const runningTool: ToolDetail[] = [{
        id: 'tool-running',
        name: 'Running Tool',
        description: 'Tool currently executing',
        status: 'running'
      }];

      const addons: MessageAddon[] = [{
        id: 'tool-running',
        type: 'tool',
        emoji: '⚡',
        tooltip: 'Running Tool'
      }];

      const { container } = render(
        <ComplexAddons
          addons={addons}
          toolDetails={runningTool}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Running Tool"]');
      expect(button).toHaveClass('scale-110');

      // Check for status indicator dot
      const statusDot = container.querySelector('.bg-primary-blue.rounded-full');
      expect(statusDot).toBeInTheDocument();
    });

    it('renders completed status without special styling', () => {
      const completedTool: ToolDetail[] = [{
        id: 'tool-completed',
        name: 'Completed Tool',
        description: 'Tool finished successfully',
        status: 'completed'
      }];

      const addons: MessageAddon[] = [{
        id: 'tool-completed',
        type: 'tool',
        emoji: '✅',
        tooltip: 'Completed Tool'
      }];

      const { container } = render(
        <ComplexAddons
          addons={addons}
          toolDetails={completedTool}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Completed Tool"]');
      expect(button).toBeInTheDocument();
      expect(button).not.toHaveClass('opacity-70');
    });

    it('renders error status with red text', () => {
      const errorTool: ToolDetail[] = [{
        id: 'tool-error',
        name: 'Error Tool',
        description: 'Tool encountered an error',
        status: 'error'
      }];

      const addons: MessageAddon[] = [{
        id: 'tool-error',
        type: 'tool',
        emoji: '❌',
        tooltip: 'Error Tool'
      }];

      const { container } = render(
        <ComplexAddons
          addons={addons}
          toolDetails={errorTool}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Error Tool"]');
      expect(button).toHaveClass('text-red-500');
    });
  });

  describe('Tooltip Display', () => {
    it('shows tooltip on hover', async () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.mouseEnter(button!);

      await waitFor(() => {
        expect(screen.getByText('Search Tool')).toBeInTheDocument();
        expect(screen.getByText('Searches for relevant information')).toBeInTheDocument();
      });
    });

    it('hides tooltip on mouse leave', async () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.mouseEnter(button!);

      await waitFor(() => {
        expect(screen.getByText('Searches for relevant information')).toBeInTheDocument();
      });

      fireEvent.mouseLeave(button!);

      await waitFor(() => {
        expect(screen.queryByText('Searches for relevant information')).not.toBeInTheDocument();
      });
    });

    it('displays status in tooltip for running tools', async () => {
      const runningTool: ToolDetail[] = [{
        id: 'tool-1',
        name: 'Search Tool',
        description: 'Searches for information',
        status: 'running'
      }];

      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={runningTool}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.mouseEnter(button!);

      await waitFor(() => {
        expect(screen.getByText('⚡ Currently running...')).toBeInTheDocument();
      });
    });

    it('displays status in tooltip for completed tools', async () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.mouseEnter(button!);

      await waitFor(() => {
        expect(screen.getByText('✓ Completed')).toBeInTheDocument();
      });
    });

    it('displays status in tooltip for error tools', async () => {
      const errorTool: ToolDetail[] = [{
        id: 'tool-1',
        name: 'Search Tool',
        description: 'Searches for information',
        status: 'error'
      }];

      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={errorTool}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.mouseEnter(button!);

      await waitFor(() => {
        expect(screen.getByText('⚠️ Error occurred')).toBeInTheDocument();
      });
    });

    it('hides tooltip when tool is expanded', async () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();

      // Click to expand
      fireEvent.click(button!);

      // Hover should not show tooltip
      fireEvent.mouseEnter(button!);

      // Wait a bit to ensure tooltip doesn't appear
      await new Promise(resolve => setTimeout(resolve, 100));

      // Tooltip description should not be in a tooltip (it's in the expanded details)
      const tooltipDescriptions = screen.queryAllByText('Searches for relevant information');
      expect(tooltipDescriptions.length).toBe(1); // Only in expanded details, not tooltip
    });
  });

  describe('Expandable Details Behavior', () => {
    it('expands tool details on click', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      expect(screen.getByText('Searches for relevant information')).toBeInTheDocument();
      expect(screen.getByText('✓ Found 5 results')).toBeInTheDocument();
    });

    it('collapses tool details on second click', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();

      // First click - expand
      fireEvent.click(button!);
      expect(screen.getByText(/Found 5 results/)).toBeInTheDocument();

      // Second click - collapse
      fireEvent.click(button!);
      expect(screen.queryByText(/Found 5 results/)).not.toBeInTheDocument();
    });

    it('shows only one expanded tool at a time', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button1 = container.querySelector('button[title="Search Tool"]');
      const button2 = container.querySelector('button[title="Analytics"]');
      expect(button1).toBeInTheDocument();
      expect(button2).toBeInTheDocument();

      // Expand first tool
      fireEvent.click(button1!);
      expect(screen.getByText(/Found 5 results/)).toBeInTheDocument();

      // Expand second tool
      fireEvent.click(button2!);
      expect(screen.queryByText(/Found 5 results/)).not.toBeInTheDocument();
      expect(screen.getByText('Analyzes data patterns')).toBeInTheDocument();
    });

    it('does not expand audio addons', () => {
      const audioAddon: MessageAddon[] = [{
        id: 'audio-1',
        type: 'audio',
        emoji: '🎵',
        tooltip: 'Audio'
      }];

      const { container } = render(
        <ComplexAddons
          addons={audioAddon}
          timestamp={mockTimestamp}
        />
      );

      // Audio addon renders AudioAddon component, no button to click
      expect(container.querySelector('button[title="Audio"]')).not.toBeInTheDocument();
    });
  });

  describe('Tool Details Content', () => {
    it('displays tool name and description when expanded', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      expect(screen.getByText('Search Tool')).toBeInTheDocument();
      expect(screen.getByText('Searches for relevant information')).toBeInTheDocument();
    });

    it('displays result content when available', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      expect(screen.getByText('✓ Found 5 results')).toBeInTheDocument();
    });

    it('displays running status in expanded view', () => {
      const runningTool: ToolDetail[] = [{
        id: 'tool-1',
        name: 'Search Tool',
        description: 'Searching...',
        status: 'running'
      }];

      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={runningTool}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      expect(screen.getByText('Running...')).toBeInTheDocument();
    });

    it('displays pending status in expanded view', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Analytics"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      expect(screen.getByText('⏳ Pending...')).toBeInTheDocument();
    });

    it('displays error message in expanded view', () => {
      const errorTool: ToolDetail[] = [{
        id: 'tool-1',
        name: 'Search Tool',
        description: 'Failed to search',
        status: 'error'
      }];

      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={errorTool}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      expect(screen.getByText('❌ Error occurred')).toBeInTheDocument();
    });

    it('does not display result when not available', () => {
      const toolWithoutResult: ToolDetail[] = [{
        id: 'tool-1',
        name: 'Search Tool',
        description: 'Searches for information',
        status: 'completed'
      }];

      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={toolWithoutResult}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      expect(screen.queryByText(/✓/)).not.toBeInTheDocument();
    });
  });

  describe('Click Interactions', () => {
    it('handles click on tool addon', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();

      expect(screen.queryByText(/Found 5 results/)).not.toBeInTheDocument();

      fireEvent.click(button!);

      expect(screen.getByText(/Found 5 results/)).toBeInTheDocument();
    });

    it('handles click on icon addon', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Analytics"]');
      expect(button).toBeInTheDocument();

      fireEvent.click(button!);

      expect(screen.getByText('Analyzes data patterns')).toBeInTheDocument();
    });

    it('applies expanded styling when clicked', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();

      fireEvent.click(button!);

      expect(button).toHaveClass('scale-110');
      expect(button).toHaveClass('bg-primary-blue/20');
    });

    it('removes expanded styling when collapsed', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();

      fireEvent.click(button!);
      expect(button).toHaveClass('bg-primary-blue/20');

      fireEvent.click(button!);
      expect(button).not.toHaveClass('bg-primary-blue/20');
    });
  });

  describe('Empty State Handling', () => {
    it('renders without tool details', () => {
      render(
        <ComplexAddons
          addons={mockAddons}
          timestamp={mockTimestamp}
        />
      );

      expect(screen.getByText('🔧')).toBeInTheDocument();
      expect(screen.getByText('📊')).toBeInTheDocument();
    });

    it('handles addon without matching tool detail', () => {
      const addonWithoutDetail: MessageAddon[] = [{
        id: 'no-detail',
        type: 'tool',
        emoji: '🔍',
        tooltip: 'Unknown Tool'
      }];

      const { container } = render(
        <ComplexAddons
          addons={addonWithoutDetail}
          toolDetails={mockToolDetails}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Unknown Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      // Should not show expanded details when no matching tool detail exists
      const unknownButtons = container.querySelectorAll('button[title="Unknown Tool"]');
      expect(unknownButtons.length).toBeGreaterThan(0); // Button still exists
    });

    it('handles empty tool details array', () => {
      const { container } = render(
        <ComplexAddons
          addons={mockAddons}
          toolDetails={[]}
          timestamp={mockTimestamp}
        />
      );

      const button = container.querySelector('button[title="Search Tool"]');
      expect(button).toBeInTheDocument();
      fireEvent.click(button!);

      // Should not crash, just not show details
      expect(screen.queryByText('Found 5 results')).not.toBeInTheDocument();
    });
  });

  describe('Audio Addon Rendering', () => {
    it('renders AudioAddon component for audio type', () => {
      const audioAddon: MessageAddon[] = [{
        id: 'audio-1',
        type: 'audio',
        emoji: '🎵',
        tooltip: 'Audio Message'
      }];

      const { container } = render(
        <ComplexAddons
          addons={audioAddon}
          timestamp={mockTimestamp}
        />
      );

      // AudioAddon should be rendered (we can check for its presence)
      // Since AudioAddon is a complex component, we just verify it's being rendered
      expect(container.querySelector('button')).toBeInTheDocument();
    });

    it('renders both audio and non-audio addons together', () => {
      const mixedAddons: MessageAddon[] = [
        {
          id: 'audio-1',
          type: 'audio',
          emoji: '🎵',
          tooltip: 'Audio'
        },
        {
          id: 'tool-1',
          type: 'tool',
          emoji: '🔧',
          tooltip: 'Tool'
        }
      ];

      render(
        <ComplexAddons
          addons={mixedAddons}
          timestamp={mockTimestamp}
        />
      );

      expect(screen.getByText('🔧')).toBeInTheDocument();
    });
  });
});

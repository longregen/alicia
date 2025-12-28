import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ServerInfoPanel from './ServerInfoPanel';

// Mock API service
const mockGetServerInfo = vi.fn();
const mockGetGlobalStats = vi.fn();

vi.mock('../../../services/api', () => ({
  api: {
    getServerInfo: () => mockGetServerInfo(),
    getGlobalStats: () => mockGetGlobalStats(),
  },
}));

// Mock useServerInfo hook
vi.mock('../../../hooks/useServerInfo', () => ({
  useServerInfo: () => ({
    connectionStatus: 'connected',
    latency: 45,
    isConnected: true,
    isConnecting: false,
    connectionQuality: 'excellent' as const,
    modelInfo: {
      name: 'claude-opus-4-5',
      provider: 'anthropic',
    },
    mcpServers: [
      { name: 'filesystem', status: 'connected' },
      { name: 'database', status: 'disconnected' },
    ],
    sessionStats: {
      messageCount: 42,
      toolCallCount: 15,
      memoriesUsed: 8,
      sessionDuration: 3665,
    },
    mcpServerSummary: {
      total: 2,
      connected: 1,
      disconnected: 1,
    },
    formattedSessionDuration: '1h 1m',
  }),
}));

// Mock serverInfoStore
vi.mock('../../../stores/serverInfoStore', () => ({
  useServerInfoStore: {
    getState: () => ({
      setConnectionStatus: vi.fn(),
      setLatency: vi.fn(),
      setModelInfo: vi.fn(),
      setMCPServers: vi.fn(),
      setSessionStats: vi.fn(),
    }),
  },
}));

describe('ServerInfoPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockGetServerInfo.mockResolvedValue({
      connection: {
        status: 'connected',
        latency: 45,
      },
      model: {
        name: 'claude-opus-4-5',
        provider: 'anthropic',
      },
      mcpServers: [
        { name: 'filesystem', status: 'connected' },
        { name: 'database', status: 'disconnected' },
      ],
    });

    mockGetGlobalStats.mockResolvedValue({
      messageCount: 42,
      toolCallCount: 15,
      memoriesUsed: 8,
      sessionDuration: 3665,
    });
  });

  describe('Loading State', () => {
    it('shows loading spinner initially', () => {
      render(<ServerInfoPanel />);

      expect(screen.getByText('Loading server info...')).toBeInTheDocument();
    });
  });

  describe('Connection Status', () => {
    it('displays connection status', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('Connected')).toBeInTheDocument();
      });
    });

    it('displays latency and quality', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText(/45ms/)).toBeInTheDocument();
        expect(screen.getByText(/excellent/i)).toBeInTheDocument();
      });
    });

    it('applies correct color for connected status', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        const statusBadge = screen.getByText('Connected');
        expect(statusBadge).toHaveClass('text-green-600');
      });
    });
  });

  describe('Model Info', () => {
    it('displays model name', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('claude-opus-4-5')).toBeInTheDocument();
      });
    });

    it('displays provider', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText(/Provider: anthropic/)).toBeInTheDocument();
      });
    });
  });

  describe('MCP Servers', () => {
    it('displays MCP server section', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('MCP Servers')).toBeInTheDocument();
      });
    });

    it('shows connected/total count', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('1/2 connected')).toBeInTheDocument();
      });
    });

    it('lists all MCP servers', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('filesystem')).toBeInTheDocument();
        expect(screen.getByText('database')).toBeInTheDocument();
      });
    });

    it('shows server status badges', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        const statusElements = screen.getAllByText(/connected|disconnected/i);
        expect(statusElements.length).toBeGreaterThan(0);
      });
    });

    it('applies correct colors for server status', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        const connectedBadges = screen.getAllByText('connected');
        expect(connectedBadges[0]).toHaveClass('text-green-600');
      });
    });
  });

  describe('Session Statistics', () => {
    it('displays message count', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('42')).toBeInTheDocument();
        expect(screen.getByText('Messages')).toBeInTheDocument();
      });
    });

    it('displays tool call count', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('15')).toBeInTheDocument();
        expect(screen.getByText('Tool Calls')).toBeInTheDocument();
      });
    });

    it('displays memories used', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('8')).toBeInTheDocument();
        expect(screen.getByText('Memories Used')).toBeInTheDocument();
      });
    });

    it('displays formatted session duration', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('1h 1m')).toBeInTheDocument();
        expect(screen.getByText('Duration')).toBeInTheDocument();
      });
    });
  });

  describe('Error State', () => {
    it('shows error message when fetch fails', async () => {
      mockGetServerInfo.mockRejectedValue(new Error('Network error'));

      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('Network error')).toBeInTheDocument();
      });
    });

    it('shows retry button on error', async () => {
      mockGetServerInfo.mockRejectedValue(new Error('Network error'));

      render(<ServerInfoPanel />);

      await waitFor(() => {
        expect(screen.getByText('Retry')).toBeInTheDocument();
      });
    });
  });

  describe('Compact Mode', () => {
    it('applies compact spacing', () => {
      const { container } = render(<ServerInfoPanel compact={true} />);

      const sections = container.querySelectorAll('[class*="space-y"]');
      expect(sections.length).toBeGreaterThan(0);
    });
  });

  describe('Custom ClassName', () => {
    it('applies custom className', () => {
      const { container } = render(<ServerInfoPanel className="custom-class" />);

      const rootElement = container.firstChild;
      expect(rootElement).toHaveClass('custom-class');
    });
  });

  describe('Connection Quality Colors', () => {
    it('uses green for excellent quality', async () => {
      render(<ServerInfoPanel />);

      await waitFor(() => {
        const qualityText = screen.getByText(/excellent/i);
        expect(qualityText).toHaveClass('text-green-600');
      });
    });
  });

  describe('Responsive Grid Layout', () => {
    it('uses grid layout for session stats', async () => {
      const { container } = render(<ServerInfoPanel />);

      await waitFor(() => {
        const grid = container.querySelector('.grid-cols-2');
        expect(grid).toBeInTheDocument();
      });
    });
  });
});

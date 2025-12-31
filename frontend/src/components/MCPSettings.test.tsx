import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { MCPSettings } from './MCPSettings';
import { api } from '../services/api';
import type { MCPServer, MCPTool } from '../types/mcp';

vi.mock('../services/api');

describe('MCPSettings', () => {
  const mockServers: MCPServer[] = [
    {
      name: 'test-server',
      transport: 'stdio',
      command: 'npx',
      args: ['-y', 'test-mcp-server'],
      env: { API_KEY: 'test-key' },
      status: 'connected',
      tools: ['tool1', 'tool2'],
    },
    {
      name: 'error-server',
      transport: 'sse',
      command: 'broken-server',
      args: [],
      status: 'error',
      tools: [],
      error: 'Connection failed',
    },
  ];

  const mockTools: MCPTool[] = [
    {
      name: 'tool1',
      description: 'First test tool',
    },
    {
      name: 'tool2',
      description: 'Second test tool',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getMCPServers).mockResolvedValue(mockServers);
    vi.mocked(api.getMCPTools).mockResolvedValue(mockTools);
    vi.mocked(api.addMCPServer).mockResolvedValue(mockServers[0]);
    vi.mocked(api.removeMCPServer).mockResolvedValue(undefined);
    vi.spyOn(window, 'confirm').mockReturnValue(true);
  });

  describe('initial loading', () => {
    it('should display loading state initially', () => {
      vi.mocked(api.getMCPServers).mockReturnValue(new Promise(() => {}));
      vi.mocked(api.getMCPTools).mockReturnValue(new Promise(() => {}));

      render(<MCPSettings />);

      expect(screen.getByText('Loading servers...')).toBeInTheDocument();
    });

    it('should load and display servers', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      expect(screen.getByText('error-server')).toBeInTheDocument();
      expect(api.getMCPServers).toHaveBeenCalledOnce();
      expect(api.getMCPTools).toHaveBeenCalledOnce();
    });

    it('should display error message when loading fails', async () => {
      vi.mocked(api.getMCPServers).mockRejectedValue(new Error('Network error'));

      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('Network error')).toBeInTheDocument();
      });
    });

    it('should display empty state when no servers exist', async () => {
      vi.mocked(api.getMCPServers).mockResolvedValue([]);
      vi.mocked(api.getMCPTools).mockResolvedValue([]);

      render(<MCPSettings />);

      await waitFor(() => {
        expect(
          screen.getByText('No MCP servers configured. Click "Add Server" to get started.')
        ).toBeInTheDocument();
      });
    });
  });

  describe('server status display', () => {
    it('should display connected status badge', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const statusBadge = screen.getByText('Connected');
      expect(statusBadge).toHaveClass('badge', 'badge-success');
    });

    it('should display error status badge', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('error-server')).toBeInTheDocument();
      });

      const statusBadge = screen.getByText('Error');
      expect(statusBadge).toHaveClass('badge', 'badge-error');
    });

    it('should display error message for failed servers', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('Connection failed')).toBeInTheDocument();
      });
    });
  });

  describe('server details', () => {
    it('should display server transport', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('stdio')).toBeInTheDocument();
      });

      expect(screen.getByText('sse')).toBeInTheDocument();
    });

    it('should display server command', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('npx')).toBeInTheDocument();
      });

      expect(screen.getByText('broken-server')).toBeInTheDocument();
    });

    it('should display server arguments', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('-y, test-mcp-server')).toBeInTheDocument();
      });
    });

    it('should not display args section when no arguments exist', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('error-server')).toBeInTheDocument();
      });

      const errorServerCard = screen.getByText('error-server').closest('.card');
      expect(errorServerCard).toBeInTheDocument();

      const argsLabel = within(errorServerCard as HTMLElement).queryByText('Args:');
      expect(argsLabel).not.toBeInTheDocument();
    });
  });

  describe('tool listing', () => {
    it('should show tool count for each server', async () => {
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText(/Tools \(2\)/)).toBeInTheDocument();
      });
    });

    it('should expand and show tools when toggle is clicked', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const toggleButton = screen.getByText(/Tools \(2\)/);
      await user.click(toggleButton);

      expect(screen.getByText('tool1')).toBeInTheDocument();
      expect(screen.getByText('First test tool')).toBeInTheDocument();
      expect(screen.getByText('tool2')).toBeInTheDocument();
      expect(screen.getByText('Second test tool')).toBeInTheDocument();
    });

    it('should collapse tools when toggle is clicked again', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const toggleButton = screen.getByText(/Tools \(2\)/);

      await user.click(toggleButton);
      expect(screen.getByText('tool1')).toBeInTheDocument();

      await user.click(toggleButton);
      expect(screen.queryByText('tool1')).not.toBeInTheDocument();
    });

    it('should display tool without description', async () => {
      const user = userEvent.setup();
      vi.mocked(api.getMCPTools).mockResolvedValue([{ name: 'tool-no-desc' }]);
      vi.mocked(api.getMCPServers).mockResolvedValue([
        {
          ...mockServers[0],
          tools: ['tool-no-desc'],
        },
      ]);

      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const toggleButton = screen.getByText(/Tools \(1\)/);
      await user.click(toggleButton);

      expect(screen.getByText('tool-no-desc')).toBeInTheDocument();
    });
  });

  describe('add server form', () => {
    it('should show form when add button is clicked', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      expect(screen.getByText('Add MCP Server')).toBeInTheDocument();
      expect(screen.getByLabelText('Server Name *')).toBeInTheDocument();
      expect(screen.getByLabelText('Transport *')).toBeInTheDocument();
      expect(screen.getByLabelText('Command *')).toBeInTheDocument();
    });

    it('should hide form when cancel is clicked', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));
      expect(screen.getByText('Add MCP Server')).toBeInTheDocument();

      // Click the cancel button within the form actions (not the header button)
      const form = screen.getByText('Add MCP Server').closest('form');
      const cancelButton = within(form!).getByRole('button', { name: 'Cancel' });
      await user.click(cancelButton);
      expect(screen.queryByText('Add MCP Server')).not.toBeInTheDocument();
    });

    it('should change button text to "Cancel" when form is shown', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      // Check that the header button now says "Cancel"
      const headerButtons = screen.getAllByText('Cancel');
      expect(headerButtons.length).toBeGreaterThan(0);
      expect(screen.queryByText('+ Add Server')).not.toBeInTheDocument();
    });
  });

  describe('form validation', () => {
    it('should trim whitespace from name and command before validation', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      const nameInput = screen.getByLabelText('Server Name *');
      await user.type(nameInput, '   ');

      const commandInput = screen.getByLabelText('Command *');
      await user.type(commandInput, '   ');

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Server name is required')).toBeInTheDocument();
      });
    });
  });

  describe('adding servers', () => {
    it('should add server with required fields only', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'new-server');
      await user.type(screen.getByLabelText('Command *'), 'npx');

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(api.addMCPServer).toHaveBeenCalledWith({
          name: 'new-server',
          transport: 'stdio',
          command: 'npx',
          args: [],
          env: undefined,
        });
      });
    });

    it('should add server with all fields', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'full-server');
      await user.selectOptions(screen.getByLabelText('Transport *'), 'sse');
      await user.type(screen.getByLabelText('Command *'), 'test-cmd');
      await user.type(screen.getByLabelText('Arguments (comma-separated)'), 'arg1, arg2, arg3');
      await user.type(
        screen.getByLabelText('Environment Variables (KEY=value, one per line)'),
        'API_KEY=secret\nDEBUG=true'
      );

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(api.addMCPServer).toHaveBeenCalledWith({
          name: 'full-server',
          transport: 'sse',
          command: 'test-cmd',
          args: ['arg1', 'arg2', 'arg3'],
          env: {
            API_KEY: 'secret',
            DEBUG: 'true',
          },
        });
      });
    });

    it('should parse comma-separated args correctly', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'test');
      await user.type(screen.getByLabelText('Command *'), 'cmd');
      await user.type(screen.getByLabelText('Arguments (comma-separated)'), '  arg1  ,  arg2  ');

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(api.addMCPServer).toHaveBeenCalledWith(
          expect.objectContaining({
            args: ['arg1', 'arg2'],
          })
        );
      });
    });

    it('should parse environment variables with equals signs in values', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'test');
      await user.type(screen.getByLabelText('Command *'), 'cmd');
      await user.type(
        screen.getByLabelText('Environment Variables (KEY=value, one per line)'),
        'URL=http://example.com?foo=bar'
      );

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(api.addMCPServer).toHaveBeenCalledWith(
          expect.objectContaining({
            env: {
              URL: 'http://example.com?foo=bar',
            },
          })
        );
      });
    });

    it('should reset form after successful submission', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'test-server');
      await user.type(screen.getByLabelText('Command *'), 'test-cmd');
      await user.type(screen.getByLabelText('Arguments (comma-separated)'), 'arg1');

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.queryByText('Add MCP Server')).not.toBeInTheDocument();
      });
    });

    it('should show success toast after adding server', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'test');
      await user.type(screen.getByLabelText('Command *'), 'cmd');

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Server added successfully')).toBeInTheDocument();
      });
    });

    it('should reload servers and tools after adding', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      vi.clearAllMocks();

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'test');
      await user.type(screen.getByLabelText('Command *'), 'cmd');

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(api.getMCPServers).toHaveBeenCalled();
        expect(api.getMCPTools).toHaveBeenCalled();
      });
    });

    it('should show error toast when adding server fails', async () => {
      const user = userEvent.setup();
      vi.mocked(api.addMCPServer).mockRejectedValue(new Error('Server already exists'));

      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'test');
      await user.type(screen.getByLabelText('Command *'), 'cmd');

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Server already exists')).toBeInTheDocument();
      });
    });

    it('should disable submit button while submitting', async () => {
      const user = userEvent.setup();
      vi.mocked(api.addMCPServer).mockReturnValue(new Promise(() => {}));

      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('+ Add Server')).toBeInTheDocument();
      });

      await user.click(screen.getByText('+ Add Server'));

      await user.type(screen.getByLabelText('Server Name *'), 'test');
      await user.type(screen.getByLabelText('Command *'), 'cmd');

      const submitButton = screen.getByRole('button', { name: /Add Server/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Adding.../i })).toBeDisabled();
      });
    });
  });

  describe('removing servers', () => {
    it('should show confirmation dialog when remove button is clicked', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const removeButtons = screen.getAllByTitle('Remove server');
      await user.click(removeButtons[0]);

      expect(window.confirm).toHaveBeenCalledWith(
        'Are you sure you want to remove the server "test-server"?'
      );
    });

    it('should remove server when confirmed', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const removeButtons = screen.getAllByTitle('Remove server');
      await user.click(removeButtons[0]);

      await waitFor(() => {
        expect(api.removeMCPServer).toHaveBeenCalledWith('test-server');
      });
    });

    it('should not remove server when cancelled', async () => {
      const user = userEvent.setup();
      vi.spyOn(window, 'confirm').mockReturnValue(false);

      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const removeButtons = screen.getAllByTitle('Remove server');
      await user.click(removeButtons[0]);

      expect(api.removeMCPServer).not.toHaveBeenCalled();
    });

    it('should reload servers and tools after removing', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      vi.clearAllMocks();

      const removeButtons = screen.getAllByTitle('Remove server');
      await user.click(removeButtons[0]);

      await waitFor(() => {
        expect(api.getMCPServers).toHaveBeenCalled();
        expect(api.getMCPTools).toHaveBeenCalled();
      });
    });

    it('should show success toast after removing server', async () => {
      const user = userEvent.setup();
      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const removeButtons = screen.getAllByTitle('Remove server');
      await user.click(removeButtons[0]);

      await waitFor(() => {
        expect(screen.getByText('Server removed successfully')).toBeInTheDocument();
      });
    });

    it('should show error toast when removing fails', async () => {
      const user = userEvent.setup();
      vi.mocked(api.removeMCPServer).mockRejectedValue(new Error('Server not found'));

      render(<MCPSettings />);

      await waitFor(() => {
        expect(screen.getByText('test-server')).toBeInTheDocument();
      });

      const removeButtons = screen.getAllByTitle('Remove server');
      await user.click(removeButtons[0]);

      await waitFor(() => {
        expect(screen.getByText('Server not found')).toBeInTheDocument();
      });
    });
  });

});

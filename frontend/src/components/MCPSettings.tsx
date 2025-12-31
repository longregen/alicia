import { useState, useEffect } from 'react';
import { MCPServer, MCPServerConfig, MCPTool } from '../types/mcp';
import { api } from '../services/api';

export function MCPSettings() {
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);
  const [expandedServers, setExpandedServers] = useState<Set<string>>(new Set());

  // Form state
  const [formData, setFormData] = useState<MCPServerConfig>({
    name: '',
    transport: 'stdio',
    command: '',
    args: [],
    env: {},
  });
  const [argsInput, setArgsInput] = useState('');
  const [envInput, setEnvInput] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // Toast state
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  useEffect(() => {
    loadServers();
    loadTools();
  }, []);

  useEffect(() => {
    if (toast) {
      const timer = setTimeout(() => setToast(null), 3000);
      return () => clearTimeout(timer);
    }
  }, [toast]);

  const loadServers = async () => {
    try {
      setLoading(true);
      const data = await api.getMCPServers();
      setServers(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load servers');
    } finally {
      setLoading(false);
    }
  };

  const loadTools = async () => {
    try {
      const data = await api.getMCPTools();
      setTools(data);
    } catch (err) {
      console.error('Failed to load tools:', err);
    }
  };

  const handleAddServer = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validate form
    if (!formData.name.trim()) {
      setToast({ message: 'Server name is required', type: 'error' });
      return;
    }
    if (!formData.command.trim()) {
      setToast({ message: 'Command is required', type: 'error' });
      return;
    }

    try {
      setSubmitting(true);

      // Parse args from comma-separated input
      const args = argsInput
        .split(',')
        .map(arg => arg.trim())
        .filter(arg => arg.length > 0);

      // Parse env from key=value pairs (one per line)
      const env: Record<string, string> = {};
      if (envInput.trim()) {
        envInput.split('\n').forEach(line => {
          const [key, ...valueParts] = line.split('=');
          if (key && valueParts.length > 0) {
            env[key.trim()] = valueParts.join('=').trim();
          }
        });
      }

      const serverConfig: MCPServerConfig = {
        ...formData,
        args,
        env: Object.keys(env).length > 0 ? env : undefined,
      };

      await api.addMCPServer(serverConfig);
      await loadServers();
      await loadTools();

      // Reset form
      setFormData({
        name: '',
        transport: 'stdio',
        command: '',
        args: [],
        env: {},
      });
      setArgsInput('');
      setEnvInput('');
      setShowAddForm(false);

      setToast({ message: 'Server added successfully', type: 'success' });
    } catch (err) {
      setToast({
        message: err instanceof Error ? err.message : 'Failed to add server',
        type: 'error'
      });
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemoveServer = async (name: string) => {
    if (!confirm(`Are you sure you want to remove the server "${name}"?`)) {
      return;
    }

    try {
      await api.removeMCPServer(name);
      await loadServers();
      await loadTools();
      setToast({ message: 'Server removed successfully', type: 'success' });
    } catch (err) {
      setToast({
        message: err instanceof Error ? err.message : 'Failed to remove server',
        type: 'error'
      });
    }
  };

  const toggleServerExpanded = (name: string) => {
    const newExpanded = new Set(expandedServers);
    if (newExpanded.has(name)) {
      newExpanded.delete(name);
    } else {
      newExpanded.add(name);
    }
    setExpandedServers(newExpanded);
  };

  const getServerTools = (serverName: string): MCPTool[] => {
    const server = servers.find(s => s.name === serverName);
    if (!server) return [];

    return tools.filter(tool => server.tools.includes(tool.name));
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'connected':
        return <span className="status-badge badge badge-success">Connected</span>;
      case 'error':
        return <span className="status-badge badge badge-error">Error</span>;
      default:
        return <span className="status-badge badge badge-warning">Disconnected</span>;
    }
  };

  return (
    <div className="mcp-settings p-5 max-w-4xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-semibold text-default m-0">MCP Server Settings</h2>
        <button
          className="btn btn-primary"
          onClick={() => setShowAddForm(!showAddForm)}
        >
          {showAddForm ? 'Cancel' : '+ Add Server'}
        </button>
      </div>

      {toast && (
        <div className={`fixed top-5 right-5 px-5 py-3 rounded-md text-sm font-medium shadow-lg z-[1000] animate-[slideIn_0.3s_ease] ${
          toast.type === 'success' ? 'toast-success bg-green-500 text-white' : 'toast-error bg-red-500 text-white'
        }`}>
          {toast.message}
        </div>
      )}

      {showAddForm && (
        <form className="add-server-form bg-elevated border border-default rounded-lg p-5 mb-6" onSubmit={handleAddServer}>
          <h3 className="m-0 mb-4 text-lg font-semibold text-default">Add MCP Server</h3>

          <div className="mb-4">
            <label htmlFor="server-name" className="block mb-1.5 text-sm font-medium text-default">Server Name *</label>
            <input
              id="server-name"
              type="text"
              className="input"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="my-mcp-server"
              required
            />
          </div>

          <div className="mb-4">
            <label htmlFor="transport" className="block mb-1.5 text-sm font-medium text-default">Transport *</label>
            <select
              id="transport"
              className="input"
              value={formData.transport}
              onChange={(e) => setFormData({ ...formData, transport: e.target.value as 'stdio' | 'sse' })}
              required
            >
              <option value="stdio">stdio</option>
              <option value="sse">SSE</option>
            </select>
          </div>

          <div className="mb-4">
            <label htmlFor="command" className="block mb-1.5 text-sm font-medium text-default">Command *</label>
            <input
              id="command"
              type="text"
              className="input"
              value={formData.command}
              onChange={(e) => setFormData({ ...formData, command: e.target.value })}
              placeholder="/path/to/executable or npx package-name"
              required
            />
          </div>

          <div className="mb-4">
            <label htmlFor="args" className="block mb-1.5 text-sm font-medium text-default">Arguments (comma-separated)</label>
            <input
              id="args"
              type="text"
              className="input"
              value={argsInput}
              onChange={(e) => setArgsInput(e.target.value)}
              placeholder="arg1, arg2, arg3"
            />
          </div>

          <div className="mb-4">
            <label htmlFor="env" className="block mb-1.5 text-sm font-medium text-default">Environment Variables (KEY=value, one per line)</label>
            <textarea
              id="env"
              className="input resize-y min-h-[60px]"
              value={envInput}
              onChange={(e) => setEnvInput(e.target.value)}
              placeholder="API_KEY=your-key&#10;DEBUG=true"
              rows={3}
            />
          </div>

          <div className="flex gap-3 justify-end mt-5">
            <button
              type="button"
              className="cancel-btn btn btn-secondary"
              onClick={() => setShowAddForm(false)}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
              disabled={submitting}
            >
              {submitting ? 'Adding...' : 'Add Server'}
            </button>
          </div>
        </form>
      )}

      {loading && <div className="text-center text-muted py-10 px-5 text-sm">Loading servers...</div>}

      {error && <div className="bg-red-50 text-red-800 px-4 py-3 rounded-md mb-4 text-sm">{error}</div>}

      {!loading && !error && servers.length === 0 && (
        <div className="empty-state text-center text-muted py-10 px-5 text-sm">
          No MCP servers configured. Click "Add Server" to get started.
        </div>
      )}

      {!loading && !error && servers.length > 0 && (
        <div className="servers-list flex flex-col gap-4">
          {servers.map(server => (
            <div key={server.name} className="server-card card card-hover p-4">
              <div className="flex justify-between items-start mb-3">
                <div className="flex items-center gap-3 flex-1">
                  <h3 className="m-0 text-lg font-semibold text-default">{server.name}</h3>
                  {getStatusBadge(server.status)}
                </div>
                <button
                  className="remove-server-btn bg-transparent border-0 text-muted text-[28px] cursor-pointer p-0 w-8 h-8 flex items-center justify-center rounded transition-all hover:bg-red-50 hover:text-red-500"
                  onClick={() => handleRemoveServer(server.name)}
                  title="Remove server"
                >
                  ×
                </button>
              </div>

              <div className="flex flex-col gap-2 mb-3">
                <div className="flex gap-2 text-sm">
                  <span className="font-semibold text-muted min-w-[80px]">Transport:</span>
                  <span className="detail-value text-default break-all">{server.transport}</span>
                </div>
                <div className="flex gap-2 text-sm">
                  <span className="font-semibold text-muted min-w-[80px]">Command:</span>
                  <span className="detail-value text-default break-all">{server.command}</span>
                </div>
                {server.args.length > 0 && (
                  <div className="flex gap-2 text-sm">
                    <span className="font-semibold text-muted min-w-[80px]">Args:</span>
                    <span className="detail-value text-default break-all">{server.args.join(', ')}</span>
                  </div>
                )}
                {server.error && (
                  <div className="flex gap-2 text-sm bg-red-50 p-2 rounded">
                    <span className="font-semibold text-muted min-w-[80px]">Error:</span>
                    <span className="text-red-800">{server.error}</span>
                  </div>
                )}
              </div>

              {server.tools.length > 0 && (
                <div className="tools-section mt-3 pt-3 border-t border-default">
                  <button
                    className="tools-toggle bg-transparent border-0 text-accent text-sm font-medium cursor-pointer py-2 px-0 flex items-center gap-2 transition-colors hover:text-accent/80"
                    onClick={() => toggleServerExpanded(server.name)}
                  >
                    {expandedServers.has(server.name) ? '▼' : '▶'}
                    {' '}Tools ({server.tools.length})
                  </button>

                  {expandedServers.has(server.name) && (
                    <div className="tools-list mt-3 flex flex-col gap-2">
                      {getServerTools(server.name).map(tool => (
                        <div key={tool.name} className="tool-item bg-surface p-3 rounded-md border-l-[3px] border-accent">
                          <div className="tool-name font-semibold text-default text-sm mb-1">{tool.name}</div>
                          {tool.description && (
                            <div className="text-muted text-[13px] leading-[1.4]">{tool.description}</div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

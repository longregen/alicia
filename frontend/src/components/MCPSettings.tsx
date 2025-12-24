import { useState, useEffect } from 'react';
import { MCPServer, MCPServerConfig, MCPTool } from '../types/mcp';
import { api } from '../services/api';
import './MCPSettings.css';

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
        return <span className="status-badge status-connected">Connected</span>;
      case 'error':
        return <span className="status-badge status-error">Error</span>;
      default:
        return <span className="status-badge status-disconnected">Disconnected</span>;
    }
  };

  return (
    <div className="mcp-settings">
      <div className="mcp-header">
        <h2>MCP Server Settings</h2>
        <button
          className="add-server-btn"
          onClick={() => setShowAddForm(!showAddForm)}
        >
          {showAddForm ? 'Cancel' : '+ Add Server'}
        </button>
      </div>

      {toast && (
        <div className={`toast toast-${toast.type}`}>
          {toast.message}
        </div>
      )}

      {showAddForm && (
        <form className="add-server-form" onSubmit={handleAddServer}>
          <h3>Add MCP Server</h3>

          <div className="form-group">
            <label htmlFor="server-name">Server Name *</label>
            <input
              id="server-name"
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="my-mcp-server"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="transport">Transport *</label>
            <select
              id="transport"
              value={formData.transport}
              onChange={(e) => setFormData({ ...formData, transport: e.target.value as 'stdio' | 'sse' })}
              required
            >
              <option value="stdio">stdio</option>
              <option value="sse">SSE</option>
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="command">Command *</label>
            <input
              id="command"
              type="text"
              value={formData.command}
              onChange={(e) => setFormData({ ...formData, command: e.target.value })}
              placeholder="/path/to/executable or npx package-name"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="args">Arguments (comma-separated)</label>
            <input
              id="args"
              type="text"
              value={argsInput}
              onChange={(e) => setArgsInput(e.target.value)}
              placeholder="arg1, arg2, arg3"
            />
          </div>

          <div className="form-group">
            <label htmlFor="env">Environment Variables (KEY=value, one per line)</label>
            <textarea
              id="env"
              value={envInput}
              onChange={(e) => setEnvInput(e.target.value)}
              placeholder="API_KEY=your-key&#10;DEBUG=true"
              rows={3}
            />
          </div>

          <div className="form-actions">
            <button
              type="button"
              className="cancel-btn"
              onClick={() => setShowAddForm(false)}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="submit-btn"
              disabled={submitting}
            >
              {submitting ? 'Adding...' : 'Add Server'}
            </button>
          </div>
        </form>
      )}

      {loading && <div className="loading">Loading servers...</div>}

      {error && <div className="error-message">{error}</div>}

      {!loading && !error && servers.length === 0 && (
        <div className="empty-state">
          No MCP servers configured. Click "Add Server" to get started.
        </div>
      )}

      {!loading && !error && servers.length > 0 && (
        <div className="servers-list">
          {servers.map(server => (
            <div key={server.name} className="server-card">
              <div className="server-header">
                <div className="server-info">
                  <h3>{server.name}</h3>
                  {getStatusBadge(server.status)}
                </div>
                <button
                  className="remove-server-btn"
                  onClick={() => handleRemoveServer(server.name)}
                  title="Remove server"
                >
                  ×
                </button>
              </div>

              <div className="server-details">
                <div className="detail-row">
                  <span className="detail-label">Transport:</span>
                  <span className="detail-value">{server.transport}</span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">Command:</span>
                  <span className="detail-value">{server.command}</span>
                </div>
                {server.args.length > 0 && (
                  <div className="detail-row">
                    <span className="detail-label">Args:</span>
                    <span className="detail-value">{server.args.join(', ')}</span>
                  </div>
                )}
                {server.error && (
                  <div className="detail-row error-detail">
                    <span className="detail-label">Error:</span>
                    <span className="detail-value">{server.error}</span>
                  </div>
                )}
              </div>

              {server.tools.length > 0 && (
                <div className="tools-section">
                  <button
                    className="tools-toggle"
                    onClick={() => toggleServerExpanded(server.name)}
                  >
                    {expandedServers.has(server.name) ? '▼' : '▶'}
                    {' '}Tools ({server.tools.length})
                  </button>

                  {expandedServers.has(server.name) && (
                    <div className="tools-list">
                      {getServerTools(server.name).map(tool => (
                        <div key={tool.name} className="tool-item">
                          <div className="tool-name">{tool.name}</div>
                          {tool.description && (
                            <div className="tool-description">{tool.description}</div>
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

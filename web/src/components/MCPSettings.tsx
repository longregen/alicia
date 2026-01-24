import { useState, useEffect } from 'react';
import { MCPServer, MCPServerConfig, MCPTool } from '../types/mcp';
import { api } from '../services/api';

export function MCPSettings() {
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [tools, setTools] = useState<Record<string, MCPTool[]>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);
  const [expandedServers, setExpandedServers] = useState<Set<string>>(new Set());

  const [formData, setFormData] = useState<MCPServerConfig>({
    name: '',
    transport_type: 'stdio',
    command: '',
    args: [],
  });
  const [argsInput, setArgsInput] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // Inline validation state
  const [fieldErrors, setFieldErrors] = useState<{ name?: string; command?: string }>({});
  const [submitStatus, setSubmitStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    loadServers();
    loadTools();
  }, []);

  // Auto-clear success status after 3 seconds
  useEffect(() => {
    if (submitStatus === 'success') {
      const timer = setTimeout(() => setSubmitStatus('idle'), 3000);
      return () => clearTimeout(timer);
    }
  }, [submitStatus]);

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

    // Clear previous errors
    setFieldErrors({});
    setSubmitStatus('idle');
    setSubmitError(null);

    // Validate fields
    const errors: { name?: string; command?: string } = {};
    if (!formData.name.trim()) {
      errors.name = 'Server name is required';
    }
    if (!formData.command?.trim()) {
      errors.command = 'Command is required';
    }
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    try {
      setSubmitting(true);

      const args = argsInput
        .split(',')
        .map((arg) => arg.trim())
        .filter((arg) => arg.length > 0);

      const serverConfig: MCPServerConfig = {
        ...formData,
        args,
      };

      await api.addMCPServer(serverConfig);
      await loadServers();
      await loadTools();

      setFormData({
        name: '',
        transport_type: 'stdio',
        command: '',
        args: [],
      });
      setArgsInput('');
      setShowAddForm(false);
      setSubmitStatus('success');
    } catch (err) {
      setSubmitStatus('error');
      setSubmitError(err instanceof Error ? err.message : 'Failed to add server');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemoveServer = async (name: string) => {
    if (!confirm(`Are you sure you want to remove the server "${name}"?`)) {
      return;
    }

    setSubmitStatus('idle');
    setSubmitError(null);

    try {
      await api.removeMCPServer(name);
      await loadServers();
      await loadTools();
      setSubmitStatus('success');
    } catch (err) {
      setSubmitStatus('error');
      setSubmitError(err instanceof Error ? err.message : 'Failed to remove server');
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
    return tools[serverName] || [];
  };

  return (
    <div className="mcp-settings p-5 max-w-4xl mx-auto">
      <div className="layout-between mb-6">
        <h2 className="text-2xl font-semibold text-foreground m-0">MCP Server Settings</h2>
        <button className="btn btn-primary" onClick={() => setShowAddForm(!showAddForm)}>
          {showAddForm ? 'Cancel' : '+ Add Server'}
        </button>
      </div>

      {/* Success/error status banner */}
      {submitStatus === 'success' && (
        <div className="bg-success/10 text-success px-4 py-3 rounded-md mb-4 text-sm flex items-center gap-2 animate-[fadeIn_0.2s_ease]">
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
          </svg>
          Operation completed successfully
        </div>
      )}
      {submitStatus === 'error' && submitError && (
        <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-md mb-4 text-sm">
          {submitError}
        </div>
      )}

      {showAddForm && (
        <form
          className="add-server-form bg-elevated border border-border rounded-lg p-5 mb-6"
          onSubmit={handleAddServer}
        >
          <h3 className="m-0 mb-4 text-lg font-semibold text-foreground">Add MCP Server</h3>

          <div className="mb-4">
            <label htmlFor="server-name" className="block mb-1.5 text-sm font-medium text-foreground">
              Server Name *
            </label>
            <input
              id="server-name"
              type="text"
              className={`input ${fieldErrors.name ? 'border-destructive' : ''}`}
              value={formData.name}
              onChange={(e) => {
                setFormData({ ...formData, name: e.target.value });
                if (fieldErrors.name) setFieldErrors(prev => ({ ...prev, name: undefined }));
              }}
              placeholder="my-mcp-server"
            />
            {fieldErrors.name && (
              <span className="text-destructive text-sm mt-1 block">{fieldErrors.name}</span>
            )}
          </div>

          <div className="mb-4">
            <label htmlFor="transport" className="block mb-1.5 text-sm font-medium text-foreground">
              Transport *
            </label>
            <select
              id="transport"
              className="input"
              value={formData.transport_type}
              onChange={(e) =>
                setFormData({ ...formData, transport_type: e.target.value as 'stdio' | 'sse' })
              }
              required
            >
              <option value="stdio">stdio</option>
              <option value="sse">SSE</option>
            </select>
          </div>

          <div className="mb-4">
            <label htmlFor="command" className="block mb-1.5 text-sm font-medium text-foreground">
              Command *
            </label>
            <input
              id="command"
              type="text"
              className={`input ${fieldErrors.command ? 'border-destructive' : ''}`}
              value={formData.command}
              onChange={(e) => {
                setFormData({ ...formData, command: e.target.value });
                if (fieldErrors.command) setFieldErrors(prev => ({ ...prev, command: undefined }));
              }}
              placeholder="/path/to/executable or npx package-name"
            />
            {fieldErrors.command && (
              <span className="text-destructive text-sm mt-1 block">{fieldErrors.command}</span>
            )}
          </div>

          <div className="mb-4">
            <label htmlFor="args" className="block mb-1.5 text-sm font-medium text-foreground">
              Arguments (comma-separated)
            </label>
            <input
              id="args"
              type="text"
              className="input"
              value={argsInput}
              onChange={(e) => setArgsInput(e.target.value)}
              placeholder="arg1, arg2, arg3"
            />
          </div>

          <div className="flex gap-3 justify-end mt-5">
            <button type="button" className="cancel-btn btn btn-secondary" onClick={() => setShowAddForm(false)}>
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

      {error && <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-md mb-4 text-sm">{error}</div>}

      {!loading && !error && servers.length === 0 && (
        <div className="empty-state text-center text-muted py-10 px-5 text-sm">
          No MCP servers configured. Click "Add Server" to get started.
        </div>
      )}

      {!loading && !error && servers.length > 0 && (
        <div className="servers-list layout-stack-gap-4">
          {servers.map((server) => {
            const serverTools = getServerTools(server.name);
            return (
              <div key={server.name} className="server-card card card-hover p-4">
                <div className="flex justify-between items-start mb-3">
                  <div className="flex items-center gap-3 flex-1">
                    <h3 className="m-0 text-lg font-semibold text-foreground">{server.name}</h3>
                    <span
                      className={`badge ${server.enabled ? 'badge-success' : 'badge-warning'}`}
                    >
                      {server.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </div>
                  <button
                    className="remove-server-btn bg-transparent border-0 text-muted-foreground text-[28px] cursor-pointer p-0 w-8 h-8 flex items-center justify-center rounded transition-all hover:bg-destructive/10 hover:text-destructive"
                    onClick={() => handleRemoveServer(server.name)}
                    title="Remove server"
                  >
                    ×
                  </button>
                </div>

                <div className="layout-stack-gap mb-3">
                  <div className="flex gap-2 text-sm">
                    <span className="font-semibold text-muted-foreground min-w-[80px]">Transport:</span>
                    <span className="detail-value text-foreground break-all">{server.transport_type}</span>
                  </div>
                  {server.command && (
                    <div className="flex gap-2 text-sm">
                      <span className="font-semibold text-muted-foreground min-w-[80px]">Command:</span>
                      <span className="detail-value text-foreground break-all">{server.command}</span>
                    </div>
                  )}
                  {server.url && (
                    <div className="flex gap-2 text-sm">
                      <span className="font-semibold text-muted-foreground min-w-[80px]">URL:</span>
                      <span className="detail-value text-foreground break-all">{server.url}</span>
                    </div>
                  )}
                  {server.args && server.args.length > 0 && (
                    <div className="flex gap-2 text-sm">
                      <span className="font-semibold text-muted-foreground min-w-[80px]">Args:</span>
                      <span className="detail-value text-foreground break-all">{server.args.join(', ')}</span>
                    </div>
                  )}
                </div>

                {serverTools.length > 0 && (
                  <div className="tools-section mt-3 pt-3 border-t border-border">
                    <button
                      className="tools-toggle bg-transparent border-0 text-accent text-sm font-medium cursor-pointer py-2 px-0 layout-center-gap transition-colors hover:text-accent/80"
                      onClick={() => toggleServerExpanded(server.name)}
                    >
                      {expandedServers.has(server.name) ? '▼' : '▶'} Tools ({serverTools.length})
                    </button>

                    {expandedServers.has(server.name) && (
                      <div className="tools-list mt-3 layout-stack-gap">
                        {serverTools.map((tool) => (
                          <div
                            key={tool.name}
                            className="tool-item bg-card p-3 rounded-md border-l-[3px] border-accent"
                          >
                            <div className="tool-name font-semibold text-foreground text-sm mb-1">{tool.name}</div>
                            {tool.description && (
                              <div className="text-muted-foreground text-[13px] leading-[1.4]">
                                {tool.description}
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

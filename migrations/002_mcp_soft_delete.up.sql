-- Add soft delete support to MCP servers table
ALTER TABLE alicia_mcp_servers ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;

-- Index for efficient filtering of non-deleted servers
CREATE INDEX idx_mcp_servers_deleted_at ON alicia_mcp_servers(deleted_at) WHERE deleted_at IS NULL;

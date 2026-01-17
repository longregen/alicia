-- Remove soft delete support from MCP servers table
DROP INDEX IF EXISTS idx_mcp_servers_deleted_at;
ALTER TABLE alicia_mcp_servers DROP COLUMN IF EXISTS deleted_at;

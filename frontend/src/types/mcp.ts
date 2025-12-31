/**
 * Model Context Protocol (MCP) types.
 *
 * MCP enables Claude to connect with external data sources and tools.
 * See: https://modelcontextprotocol.io/
 */

export type MCPTransport = 'stdio' | 'sse';
export type MCPServerStatus = 'connected' | 'disconnected' | 'error';

export interface MCPServerConfig {
  name: string;
  transport: MCPTransport;
  command: string;
  args: string[];
  env?: Record<string, string>;
}

export interface MCPTool {
  name: string;
  description?: string;
  /** JSON Schema definition for tool input parameters */
  inputSchema?: Record<string, unknown>;
}

export interface MCPServer extends MCPServerConfig {
  status: MCPServerStatus;
  tools: string[];
  error?: string;
}

export interface MCPServersResponse {
  servers: MCPServer[];
}

export interface MCPToolsResponse {
  tools: Record<string, MCPTool[]>;
  total: number;
}

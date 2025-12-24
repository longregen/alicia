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
  inputSchema?: any;
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

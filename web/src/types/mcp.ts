export type MCPTransport = 'stdio' | 'sse';

export interface MCPServer {
  id: string;
  name: string;
  transport_type: MCPTransport;
  command?: string;
  args?: string[];
  url?: string;
  enabled: boolean;
  created_at: string;
}

export interface MCPServerConfig {
  name: string;
  transport_type: MCPTransport;
  command?: string;
  args?: string[];
  url?: string;
}

export interface MCPTool {
  name: string;
  description: string;
  schema: Record<string, unknown>;
}

export interface MCPServersResponse {
  servers: MCPServer[];
}

export interface MCPToolsResponse {
  tools: Record<string, MCPTool[]>;
}

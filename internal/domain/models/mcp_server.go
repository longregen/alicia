package models

import "time"

// MCPServer represents an MCP server configuration stored in the database
type MCPServer struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	TransportType  string    `json:"transport_type"`
	Command        string    `json:"command,omitempty"`
	Args           []string  `json:"args,omitempty"`
	Env            []string  `json:"env,omitempty"`
	URL            string    `json:"url,omitempty"`
	APIKey         string    `json:"api_key,omitempty"`
	AutoReconnect  bool      `json:"auto_reconnect"`
	ReconnectDelay int       `json:"reconnect_delay"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

package models

import "time"

// MCPServer represents an MCP server configuration stored in the database
type MCPServer struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	TransportType  string     `json:"transport_type"`
	Command        string     `json:"command,omitempty"`
	Args           []string   `json:"args,omitempty"`
	Env            []string   `json:"env,omitempty"`
	URL            string     `json:"url,omitempty"`
	APIKey         string     `json:"api_key,omitempty"`
	AutoReconnect  bool       `json:"auto_reconnect"`
	ReconnectDelay int        `json:"reconnect_delay"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// IsDeleted returns true if the server has been soft-deleted
func (s *MCPServer) IsDeleted() bool {
	return s.DeletedAt != nil
}

// MarkAsDeleted soft-deletes the server
func (s *MCPServer) MarkAsDeleted() {
	now := time.Now().UTC()
	s.DeletedAt = &now
	s.UpdatedAt = now
}

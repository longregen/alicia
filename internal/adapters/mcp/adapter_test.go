package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/domain/models"
)

// Mock transport for testing
type mockTransport struct {
	sendFn      func(ctx context.Context, message any) error
	receiveCh   chan Message
	isConnected bool
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		receiveCh:   make(chan Message, 10),
		isConnected: true,
	}
}

func (m *mockTransport) Send(ctx context.Context, message any) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, message)
	}
	return nil
}

func (m *mockTransport) Receive() <-chan Message {
	return m.receiveCh
}

func (m *mockTransport) Close() error {
	close(m.receiveCh)
	m.isConnected = false
	return nil
}

func (m *mockTransport) IsConnected() bool {
	return m.isConnected
}

// Mock tool service for testing
type mockToolService struct {
	tools     map[string]*models.Tool
	executors map[string]func(context.Context, map[string]any) (any, error)
}

func newMockToolService() *mockToolService {
	return &mockToolService{
		tools:     make(map[string]*models.Tool),
		executors: make(map[string]func(context.Context, map[string]any) (any, error)),
	}
}

func (m *mockToolService) RegisterTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error) {
	if _, exists := m.tools[name]; exists {
		return nil, errors.New("tool already exists")
	}
	tool := &models.Tool{
		ID:          "tool_" + name,
		Name:        name,
		Description: description,
		Schema:      schema,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.tools[name] = tool
	return tool, nil
}

func (m *mockToolService) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	for _, tool := range m.tools {
		if tool.ID == id {
			return tool, nil
		}
	}
	return nil, errors.New("tool not found")
}

func (m *mockToolService) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	tool, ok := m.tools[name]
	if !ok {
		return nil, errors.New("tool not found")
	}
	return tool, nil
}

func (m *mockToolService) Enable(ctx context.Context, id string) (*models.Tool, error) {
	tool, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	tool.Enabled = true
	return tool, nil
}

func (m *mockToolService) Disable(ctx context.Context, id string) (*models.Tool, error) {
	tool, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	tool.Enabled = false
	return tool, nil
}

func (m *mockToolService) Update(ctx context.Context, tool *models.Tool) error {
	m.tools[tool.Name] = tool
	return nil
}

func (m *mockToolService) Delete(ctx context.Context, id string) error {
	for name, tool := range m.tools {
		if tool.ID == id {
			delete(m.tools, name)
			return nil
		}
	}
	return errors.New("tool not found")
}

func (m *mockToolService) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	tools := make([]*models.Tool, 0)
	for _, tool := range m.tools {
		if tool.Enabled {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

func (m *mockToolService) ListAll(ctx context.Context) ([]*models.Tool, error) {
	tools := make([]*models.Tool, 0)
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools, nil
}

func (m *mockToolService) RegisterExecutor(name string, executor func(context.Context, map[string]any) (any, error)) error {
	if _, exists := m.executors[name]; exists {
		return errors.New("executor already registered")
	}
	m.executors[name] = executor
	return nil
}

func (m *mockToolService) ExecuteTool(ctx context.Context, name string, arguments map[string]any) (any, error) {
	executor, ok := m.executors[name]
	if !ok {
		return nil, errors.New("no executor for tool")
	}
	return executor(ctx, arguments)
}

func (m *mockToolService) CreateToolUse(ctx context.Context, messageID, toolName string, arguments map[string]any) (*models.ToolUse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockToolService) ExecuteToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockToolService) CancelToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockToolService) GetToolUseByID(ctx context.Context, id string) (*models.ToolUse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockToolService) GetToolUsesByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockToolService) GetPendingToolUses(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	return nil, errors.New("not implemented")
}

// Mock MCP server repository for testing
type mockMCPServerRepository struct {
	servers map[string]*models.MCPServer
}

func newMockMCPServerRepository() *mockMCPServerRepository {
	return &mockMCPServerRepository{
		servers: make(map[string]*models.MCPServer),
	}
}

func (m *mockMCPServerRepository) Create(ctx context.Context, server *models.MCPServer) error {
	if _, exists := m.servers[server.ID]; exists {
		return errors.New("server already exists")
	}
	m.servers[server.ID] = server
	return nil
}

func (m *mockMCPServerRepository) GetByID(ctx context.Context, id string) (*models.MCPServer, error) {
	server, ok := m.servers[id]
	if !ok {
		return nil, errors.New("server not found")
	}
	return server, nil
}

func (m *mockMCPServerRepository) GetByName(ctx context.Context, name string) (*models.MCPServer, error) {
	for _, server := range m.servers {
		if server.Name == name {
			return server, nil
		}
	}
	return nil, errors.New("server not found")
}

func (m *mockMCPServerRepository) Update(ctx context.Context, server *models.MCPServer) error {
	if _, exists := m.servers[server.ID]; !exists {
		return errors.New("server not found")
	}
	m.servers[server.ID] = server
	return nil
}

func (m *mockMCPServerRepository) Delete(ctx context.Context, id string) error {
	if _, exists := m.servers[id]; !exists {
		return errors.New("server not found")
	}
	delete(m.servers, id)
	return nil
}

func (m *mockMCPServerRepository) List(ctx context.Context) ([]*models.MCPServer, error) {
	servers := make([]*models.MCPServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	return servers, nil
}

// Mock ID generator for testing
type mockIDGenerator struct {
	counter int
}

func newMockIDGenerator() *mockIDGenerator {
	return &mockIDGenerator{counter: 0}
}

func (m *mockIDGenerator) GenerateConversationID() string {
	m.counter++
	return "ac_test"
}

func (m *mockIDGenerator) GenerateMessageID() string {
	m.counter++
	return "am_test"
}

func (m *mockIDGenerator) GenerateSentenceID() string {
	m.counter++
	return "ams_test"
}

func (m *mockIDGenerator) GenerateAudioID() string {
	m.counter++
	return "aa_test"
}

func (m *mockIDGenerator) GenerateMemoryID() string {
	m.counter++
	return "amem_test"
}

func (m *mockIDGenerator) GenerateMemoryUsageID() string {
	m.counter++
	return "amu_test"
}

func (m *mockIDGenerator) GenerateToolID() string {
	m.counter++
	return "at_test"
}

func (m *mockIDGenerator) GenerateToolUseID() string {
	m.counter++
	return "atu_test"
}

func (m *mockIDGenerator) GenerateReasoningStepID() string {
	m.counter++
	return "ar_test"
}

func (m *mockIDGenerator) GenerateCommentaryID() string {
	m.counter++
	return "aucc_test"
}

func (m *mockIDGenerator) GenerateMetaID() string {
	m.counter++
	return "amt_test"
}

func (m *mockIDGenerator) GenerateMCPServerID() string {
	m.counter++
	return "amcp_test"
}

// Helper to create a pre-initialized mock client
func createMockClient(tools []Tool, callResult *ToolsCallResult, callErr error) *Client {
	transport := newMockTransport()

	// Set up mock responses
	transport.sendFn = func(ctx context.Context, message any) error {
		req, ok := message.(*JSONRPCRequest)
		if !ok {
			return nil
		}

		// Handle different methods
		go func() {
			time.Sleep(5 * time.Millisecond)

			var result any
			switch req.Method {
			case MethodToolsList:
				result = ToolsListResult{
					Tools: tools,
				}
			case MethodToolsCall:
				if callResult != nil {
					result = callResult
				} else {
					result = ToolsCallResult{
						Content: []ContentItem{{Type: "text", Text: "default result"}},
					}
				}
			default:
				result = map[string]any{}
			}

			resp, _ := NewJSONRPCResponse(req.ID, result)
			data, _ := json.Marshal(resp)
			transport.receiveCh <- Message{Data: data}
		}()

		return nil
	}

	client := NewClient("test", transport)
	client.initialized = true
	client.serverInfo = &ServerInfo{Name: "test-server", Version: "1.0.0"}
	client.capabilities = &ServerCapabilities{}

	// Start the receive loop so the client can process responses
	go client.receiveLoop()

	return client
}

// Tests for Adapter

func TestAdapter_AddServer(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	// Replace manager with one we can control
	adapter.manager = &Manager{
		servers: make(map[string]*ManagedClient),
		ctx:     ctx,
	}

	cfg := config.MCPServerConfig{
		Name:      "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}

	// This will fail because we can't actually start a process
	// But we can verify the flow
	err := adapter.AddServer(ctx, cfg)
	if err == nil {
		t.Error("expected error when adding server with invalid command")
	}
}

func TestAdapter_RemoveServer(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	// Register some tools for a server
	toolService.RegisterTool(ctx, "mcp_test_tool1", "Tool 1", nil)
	toolService.RegisterTool(ctx, "mcp_test_tool2", "Tool 2", nil)
	adapter.serverTools["test"] = []string{"mcp_test_tool1", "mcp_test_tool2"}

	// Verify tools exist and are enabled
	if len(toolService.tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolService.tools))
	}

	// Remove server (will fail because server doesn't exist in manager)
	err := adapter.RemoveServer(ctx, "test")
	if err == nil {
		t.Error("expected error when removing non-existent server")
	}

	// But tools should still be disabled
	tool1, _ := toolService.GetByName(ctx, "mcp_test_tool1")
	if tool1 != nil && tool1.Enabled {
		t.Error("expected tool1 to be disabled")
	}
}

func TestAdapter_DiscoverAndRegisterTools(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	// Create a mock client with some tools
	tools := []Tool{
		{
			Name:        "search",
			Description: "Search for information",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:        "calculator",
			Description: "Perform calculations",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{"type": "string"},
				},
			},
		},
	}

	client := createMockClient(tools, nil, nil)
	defer client.Close()

	err := adapter.discoverAndRegisterTools(ctx, "test-server", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tools were registered
	if len(adapter.serverTools["test-server"]) != 2 {
		t.Errorf("expected 2 tools registered, got %d", len(adapter.serverTools["test-server"]))
	}

	// Verify tool naming convention
	tool, err := toolService.GetByName(ctx, "mcp_test-server_search")
	if err != nil {
		t.Errorf("expected tool to be registered: %v", err)
	}
	if tool != nil && !tool.Enabled {
		t.Error("expected tool to be enabled")
	}

	// Verify description format
	if tool != nil && tool.Description != "[MCP:test-server] Search for information" {
		t.Errorf("unexpected description: %s", tool.Description)
	}
}

func TestAdapter_DiscoverAndRegisterTools_EmptyDescription(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	tools := []Tool{
		{
			Name:        "tool",
			Description: "", // Empty description
			InputSchema: map[string]any{"type": "object"},
		},
	}

	client := createMockClient(tools, nil, nil)
	defer client.Close()

	err := adapter.discoverAndRegisterTools(ctx, "test-server", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tool, _ := toolService.GetByName(ctx, "mcp_test-server_tool")
	if tool == nil {
		t.Fatal("expected tool to be registered")
	}

	// Should have default description
	if tool.Description != "[MCP:test-server] Tool from MCP server test-server" {
		t.Errorf("unexpected description: %s", tool.Description)
	}
}

func TestAdapter_CreateExecutor(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	// Create a mock client
	callResult := &ToolsCallResult{
		Content: []ContentItem{
			{Type: "text", Text: "Result from MCP"},
		},
		IsError: false,
	}

	client := createMockClient(nil, callResult, nil)
	defer client.Close()

	// Add client to manager
	adapter.manager.servers = map[string]*ManagedClient{
		"test-server": {
			client:    client,
			connected: true,
		},
	}

	executor := adapter.createExecutor("test-server", "search")

	// Execute the tool
	result, err := executor(ctx, map[string]any{"query": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Result should be just the text since there's only one text content item
	if result != "Result from MCP" {
		t.Errorf("expected 'Result from MCP', got %v", result)
	}
}

func TestAdapter_CreateExecutor_MultipleContent(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	callResult := &ToolsCallResult{
		Content: []ContentItem{
			{Type: "text", Text: "Text result"},
			{Type: "image", MimeType: "image/png", Data: "base64data"},
		},
		IsError: false,
	}

	client := createMockClient(nil, callResult, nil)
	defer client.Close()

	adapter.manager.servers = map[string]*ManagedClient{
		"test-server": {
			client:    client,
			connected: true,
		},
	}

	executor := adapter.createExecutor("test-server", "tool")

	result, err := executor(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return structured data
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok {
		t.Fatal("expected content to be array")
	}

	if len(content) != 2 {
		t.Errorf("expected 2 content items, got %d", len(content))
	}
}

func TestAdapter_CreateExecutor_Error(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	callResult := &ToolsCallResult{
		Content: []ContentItem{
			{Type: "text", Text: "Error occurred"},
		},
		IsError: true,
	}

	client := createMockClient(nil, callResult, nil)
	defer client.Close()

	adapter.manager.servers = map[string]*ManagedClient{
		"test-server": {
			client:    client,
			connected: true,
		},
	}

	executor := adapter.createExecutor("test-server", "tool")

	_, err := executor(ctx, map[string]any{})
	if err == nil {
		t.Fatal("expected error when tool returns isError=true")
	}
}

func TestAdapter_CreateExecutor_ServerNotAvailable(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	executor := adapter.createExecutor("non-existent", "tool")

	_, err := executor(ctx, map[string]any{})
	if err == nil {
		t.Fatal("expected error when server not available")
	}
}

func TestAdapter_RefreshTools(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	// Register initial tools
	toolService.RegisterTool(ctx, "mcp_test_old_tool", "Old tool", nil)
	adapter.serverTools["test"] = []string{"mcp_test_old_tool"}

	// Create new client with different tools
	tools := []Tool{
		{Name: "new_tool", Description: "New tool", InputSchema: map[string]any{}},
	}

	client := createMockClient(tools, nil, nil)
	defer client.Close()

	adapter.manager.servers = map[string]*ManagedClient{
		"test": {
			client:    client,
			connected: true,
		},
	}

	err := adapter.RefreshTools(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Old tool should be deleted
	_, err = toolService.GetByName(ctx, "mcp_test_old_tool")
	if err == nil {
		t.Error("expected old tool to be deleted")
	}

	// New tool should be registered
	_, err = toolService.GetByName(ctx, "mcp_test_new_tool")
	if err != nil {
		t.Error("expected new tool to be registered")
	}
}

func TestAdapter_ListServers(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	adapter.manager.servers = map[string]*ManagedClient{
		"server1": {},
		"server2": {},
		"server3": {},
	}

	servers := adapter.ListServers()
	if len(servers) != 3 {
		t.Errorf("expected 3 servers, got %d", len(servers))
	}
}

func TestAdapter_GetServerStatus(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	adapter.manager.servers = map[string]*ManagedClient{
		"connected": {
			connected: true,
		},
		"disconnected": {
			connected: false,
		},
	}

	status := adapter.GetServerStatus()
	if len(status) != 2 {
		t.Errorf("expected 2 servers in status, got %d", len(status))
	}

	if !status["connected"] {
		t.Error("expected connected server to show as connected")
	}

	if status["disconnected"] {
		t.Error("expected disconnected server to show as disconnected")
	}
}

func TestAdapter_GetServerTools(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	adapter.serverTools["test"] = []string{"tool1", "tool2", "tool3"}

	tools := adapter.GetServerTools("test")
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestAdapter_ExportToolsAsJSON(t *testing.T) {
	ctx := context.Background()
	toolService := newMockToolService()
	mcpRepo := newMockMCPServerRepository()
	idGen := newMockIDGenerator()
	adapter := NewAdapter(ctx, toolService, mcpRepo, idGen)

	// Register tools
	toolService.RegisterTool(ctx, "tool1", "Tool 1", map[string]any{"type": "object"})
	toolService.RegisterTool(ctx, "tool2", "Tool 2", map[string]any{"type": "object"})
	adapter.serverTools["server1"] = []string{"tool1"}
	adapter.serverTools["server2"] = []string{"tool2"}

	jsonStr, err := adapter.ExportToolsAsJSON(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify structure
	if len(result) != 2 {
		t.Errorf("expected 2 servers in export, got %d", len(result))
	}
}

func TestFormatContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []ContentItem
		expected string
	}{
		{
			name: "single text",
			content: []ContentItem{
				{Type: "text", Text: "Hello world"},
			},
			expected: "Hello world",
		},
		{
			name: "multiple items",
			content: []ContentItem{
				{Type: "text", Text: "First"},
				{Type: "text", Text: "Second"},
			},
			expected: "First\nSecond",
		},
		{
			name: "image content",
			content: []ContentItem{
				{Type: "image", MimeType: "image/png"},
			},
			expected: "[Image: image/png]",
		},
		{
			name: "resource content",
			content: []ContentItem{
				{Type: "resource", Text: "file.txt"},
			},
			expected: "[Resource: file.txt]",
		},
		{
			name: "unknown type",
			content: []ContentItem{
				{Type: "unknown"},
			},
			expected: "[unknown]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatContent(tt.content)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatToolResult(t *testing.T) {
	tests := []struct {
		name     string
		result   *ToolsCallResult
		validate func(t *testing.T, result any)
	}{
		{
			name: "single text content",
			result: &ToolsCallResult{
				Content: []ContentItem{
					{Type: "text", Text: "Simple result"},
				},
			},
			validate: func(t *testing.T, result any) {
				str, ok := result.(string)
				if !ok {
					t.Fatal("expected string result")
				}
				if str != "Simple result" {
					t.Errorf("expected 'Simple result', got %q", str)
				}
			},
		},
		{
			name: "multiple content items",
			result: &ToolsCallResult{
				Content: []ContentItem{
					{Type: "text", Text: "Text"},
					{Type: "image", Data: "data", MimeType: "image/png"},
				},
			},
			validate: func(t *testing.T, result any) {
				m, ok := result.(map[string]any)
				if !ok {
					t.Fatal("expected map result")
				}
				content, ok := m["content"].([]map[string]any)
				if !ok {
					t.Fatal("expected content array")
				}
				if len(content) != 2 {
					t.Errorf("expected 2 items, got %d", len(content))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatToolResult(tt.result)
			tt.validate(t, result)
		})
	}
}

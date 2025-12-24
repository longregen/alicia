//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/mcp"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/config"
	"github.com/longregen/alicia/internal/domain/models"
)

// mockMCPServerRepository implements ports.MCPServerRepository for testing
type mockMCPServerRepository struct {
	servers map[string]*models.MCPServer
}

func newMockMCPServerRepository() *mockMCPServerRepository {
	return &mockMCPServerRepository{
		servers: make(map[string]*models.MCPServer),
	}
}

func (m *mockMCPServerRepository) Create(ctx context.Context, server *models.MCPServer) error {
	m.servers[server.ID] = server
	return nil
}

func (m *mockMCPServerRepository) GetByID(ctx context.Context, id string) (*models.MCPServer, error) {
	server, ok := m.servers[id]
	if !ok {
		return nil, nil
	}
	return server, nil
}

func (m *mockMCPServerRepository) GetByName(ctx context.Context, name string) (*models.MCPServer, error) {
	for _, server := range m.servers {
		if server.Name == name {
			return server, nil
		}
	}
	return nil, nil
}

func (m *mockMCPServerRepository) List(ctx context.Context) ([]*models.MCPServer, error) {
	servers := make([]*models.MCPServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	return servers, nil
}

func (m *mockMCPServerRepository) Update(ctx context.Context, server *models.MCPServer) error {
	m.servers[server.ID] = server
	return nil
}

func (m *mockMCPServerRepository) Delete(ctx context.Context, id string) error {
	delete(m.servers, id)
	return nil
}

// mockMCPTransport implements mcp.Transport for testing
type mockMCPTransport struct {
	receiveCh   chan mcp.Message
	isConnected bool
	toolCalled  bool
	toolArgs    map[string]any
}

func newMockMCPTransport() *mockMCPTransport {
	return &mockMCPTransport{
		receiveCh:   make(chan mcp.Message, 10),
		isConnected: true,
	}
}

func (m *mockMCPTransport) Send(ctx context.Context, message any) error {
	req, ok := message.(*mcp.JSONRPCRequest)
	if !ok {
		return nil
	}

	// Handle different methods
	go func() {
		time.Sleep(5 * time.Millisecond)

		var result any
		switch req.Method {
		case mcp.MethodInitialize:
			result = map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]any{
					"name":    "test-weather-server",
					"version": "1.0.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
			}

		case mcp.MethodToolsList:
			result = mcp.ToolsListResult{
				Tools: []mcp.Tool{
					{
						Name:        "get_weather",
						Description: "Get weather for a location",
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"location": map[string]any{
									"type":        "string",
									"description": "City name",
								},
							},
							"required": []string{"location"},
						},
					},
				},
			}

		case mcp.MethodToolsCall:
			// Extract arguments
			var params mcp.ToolsCallParams
			if paramsBytes, err := json.Marshal(req.Params); err == nil {
				json.Unmarshal(paramsBytes, &params)
			}

			// Track that tool was called
			m.toolCalled = true
			m.toolArgs = params.Arguments

			// Return weather data
			result = mcp.ToolsCallResult{
				Content: []mcp.ContentItem{
					{
						Type: "text",
						Text: "Weather in Paris: Sunny, 22°C",
					},
				},
				IsError: false,
			}

		default:
			result = map[string]any{}
		}

		resp, _ := mcp.NewJSONRPCResponse(req.ID, result)
		data, _ := json.Marshal(resp)
		m.receiveCh <- mcp.Message{Data: data}
	}()

	return nil
}

func (m *mockMCPTransport) Receive() <-chan mcp.Message {
	return m.receiveCh
}

func (m *mockMCPTransport) Close() error {
	close(m.receiveCh)
	m.isConnected = false
	return nil
}

func (m *mockMCPTransport) IsConnected() bool {
	return m.isConnected
}

// TestMCPFlow_EndToEnd tests the complete MCP server integration flow
func TestMCPFlow_EndToEnd(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	idGen := id.NewGenerator()

	// Setup repositories
	conversationRepo := postgres.NewConversationRepository(db.Pool)
	messageRepo := postgres.NewMessageRepository(db.Pool)
	toolRepo := postgres.NewToolRepository(db.Pool)
	toolUseRepo := postgres.NewToolUseRepository(db.Pool)

	// Setup services
	mockLiveKit := &mockLiveKitService{}
	conversationSvc := services.NewConversationService(conversationRepo, mockLiveKit, idGen)
	messageSvc := services.NewMessageService(messageRepo, idGen)

	// Setup tool service
	toolSvc := services.NewToolService(toolRepo, toolUseRepo, idGen)

	// Setup MCP adapter
	mcpRepo := newMockMCPServerRepository()
	mcpAdapter := mcp.NewAdapter(ctx, toolSvc, mcpRepo, idGen)

	// Create mock transport
	transport := newMockMCPTransport()

	// Manually create and configure MCP client
	client := mcp.NewClient("test-weather-server", transport)
	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize MCP client: %v", err)
	}

	// Discover and register tools from the mock server
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("failed to list tools from MCP server: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "get_weather" {
		t.Errorf("expected tool name 'get_weather', got '%s'", tools[0].Name)
	}

	// Register the MCP tool with Alicia's tool service
	toolName := "mcp_test-weather-server_get_weather"
	_, err = toolSvc.RegisterTool(ctx, toolName, "[MCP:test-weather-server] Get weather for a location", tools[0].InputSchema)
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Register executor that calls the MCP server
	executor := func(ctx context.Context, arguments map[string]any) (any, error) {
		result, err := client.CallTool(ctx, "get_weather", arguments)
		if err != nil {
			return nil, err
		}
		if result.IsError {
			return nil, err
		}
		// Return simple text response
		if len(result.Content) == 1 && result.Content[0].Type == "text" {
			return result.Content[0].Text, nil
		}
		return result, nil
	}

	if err := toolSvc.RegisterExecutor(toolName, executor); err != nil {
		t.Fatalf("failed to register executor: %v", err)
	}

	// Test: Create a conversation
	conversation, err := conversationSvc.Create(ctx, "test-user", "Weather Query Test")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	// Test: Add a user message asking about weather
	userMsg, err := messageSvc.Create(ctx, &services.CreateMessageInput{
		ConversationID: conversation.ID,
		Role:           models.MessageRoleUser,
		Contents:       "What's the weather in Paris?",
	})
	if err != nil {
		t.Fatalf("failed to create user message: %v", err)
	}

	if userMsg.SequenceNumber != 1 {
		t.Errorf("expected sequence number 1, got %d", userMsg.SequenceNumber)
	}

	// Test: Simulate assistant using the MCP tool
	assistantMsg, err := messageSvc.Create(ctx, &services.CreateMessageInput{
		ConversationID: conversation.ID,
		Role:           models.MessageRoleAssistant,
		Contents:       "Let me check the weather for you.",
	})
	if err != nil {
		t.Fatalf("failed to create assistant message: %v", err)
	}

	// Create a tool use record
	toolUse := models.NewToolUse(
		idGen.GenerateToolUseID(),
		assistantMsg.ID,
		toolName,
		1,
		map[string]any{"location": "Paris"},
	)

	err = toolUseRepo.Create(ctx, toolUse)
	if err != nil {
		t.Fatalf("failed to create tool use: %v", err)
	}

	// Test: Execute the tool
	result, err := toolSvc.ExecuteTool(ctx, toolName, map[string]any{"location": "Paris"})
	if err != nil {
		t.Fatalf("failed to execute tool: %v", err)
	}

	// Verify the tool was called
	if !transport.toolCalled {
		t.Error("expected MCP tool to be called")
	}

	if transport.toolArgs["location"] != "Paris" {
		t.Errorf("expected location argument 'Paris', got %v", transport.toolArgs["location"])
	}

	// Verify the result
	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", result)
	}

	if resultStr != "Weather in Paris: Sunny, 22°C" {
		t.Errorf("expected weather result, got: %s", resultStr)
	}

	// Update tool use with result
	toolUse.Start()
	toolUse.Complete(result)
	err = toolUseRepo.Update(ctx, toolUse)
	if err != nil {
		t.Fatalf("failed to update tool use: %v", err)
	}

	// Verify tool use status
	retrievedToolUse, err := toolUseRepo.GetByID(ctx, toolUse.ID)
	if err != nil {
		t.Fatalf("failed to retrieve tool use: %v", err)
	}

	if retrievedToolUse.Status != models.ToolStatusSuccess {
		t.Errorf("expected status 'success', got '%s'", retrievedToolUse.Status)
	}

	if retrievedToolUse.Result == nil {
		t.Error("expected result to be set")
	}

	// Test: Create final assistant message with the result
	finalMsg, err := messageSvc.Create(ctx, &services.CreateMessageInput{
		ConversationID: conversation.ID,
		Role:           models.MessageRoleAssistant,
		Contents:       resultStr,
	})
	if err != nil {
		t.Fatalf("failed to create final message: %v", err)
	}

	// Verify conversation has all messages
	messages, err := messageRepo.ListByConversation(ctx, conversation.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}

	// Verify message sequence
	if messages[0].ID != userMsg.ID || messages[1].ID != assistantMsg.ID || messages[2].ID != finalMsg.ID {
		t.Error("messages not in correct sequence")
	}

	// Verify tool uses for assistant message
	toolUses, err := toolUseRepo.ListByMessage(ctx, assistantMsg.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list tool uses: %v", err)
	}

	if len(toolUses) != 1 {
		t.Errorf("expected 1 tool use, got %d", len(toolUses))
	}

	if toolUses[0].ToolName != toolName {
		t.Errorf("expected tool name '%s', got '%s'", toolName, toolUses[0].ToolName)
	}

	// Cleanup
	client.Close()
	mcpAdapter.Close()
}

// TestMCPFlow_AddAndRemoveServer tests adding and removing MCP servers
func TestMCPFlow_AddAndRemoveServer(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	idGen := id.NewGenerator()

	// Setup repositories
	toolRepo := postgres.NewToolRepository(db.Pool)
	toolUseRepo := postgres.NewToolUseRepository(db.Pool)

	// Setup services
	toolSvc := services.NewToolService(toolRepo, toolUseRepo, idGen)

	// Setup MCP adapter with mock repository
	mcpRepo := newMockMCPServerRepository()
	mcpAdapter := mcp.NewAdapter(ctx, toolSvc, mcpRepo, idGen)

	// Test: Add a server configuration (this will fail to connect since we can't spawn a real process)
	// but we can verify the configuration flow
	cfg := config.MCPServerConfig{
		Name:           "test-server",
		Transport:      "stdio",
		Command:        "nonexistent",
		Args:           []string{},
		AutoReconnect:  false,
		ReconnectDelay: 5,
	}

	// We expect this to fail because the command doesn't exist
	err := mcpAdapter.AddServer(ctx, cfg)
	if err == nil {
		t.Error("expected error when adding server with invalid command")
	}

	// Verify server list is empty (server wasn't added due to connection failure)
	servers := mcpAdapter.ListServers()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after failed add, got %d", len(servers))
	}

	// Cleanup
	mcpAdapter.Close()
}

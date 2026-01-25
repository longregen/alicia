package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/longregen/alicia/pkg/langfuse"
	"github.com/longregen/alicia/pkg/otel"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	lfClient     *langfuse.Client
	lfClientOnce sync.Once
)

func getLangfuseClient() *langfuse.Client {
	lfClientOnce.Do(func() {
		host := os.Getenv("LANGFUSE_HOST")
		publicKey := os.Getenv("LANGFUSE_PUBLIC_KEY")
		secretKey := os.Getenv("LANGFUSE_SECRET_KEY")

		if host != "" && publicKey != "" && secretKey != "" {
			lfClient = langfuse.New(host, publicKey, secretKey)
			slog.Info("langfuse client initialized")
		} else {
			slog.Info("langfuse not configured, missing LANGFUSE_HOST, LANGFUSE_PUBLIC_KEY, or LANGFUSE_SECRET_KEY")
		}
	})
	return lfClient
}

// PromptResult holds compiled prompt text and its Langfuse metadata for tracing.
type PromptResult struct {
	Text    string
	Name    string // Langfuse prompt name (empty if fallback)
	Version int    // Langfuse prompt version (0 if fallback)
}

func getSystemPrompt(memories []Memory, notes []Note, tools []Tool, instructions string) PromptResult {
	client := getLangfuseClient()

	memoriesStr := ""
	if len(memories) > 0 {
		var sb strings.Builder
		sb.WriteString("Relevant memories:\n")
		for _, m := range memories {
			sb.WriteString("- ")
			sb.WriteString(m.Content)
			sb.WriteByte('\n')
		}
		memoriesStr = sb.String()
	}

	notesStr := ""
	if len(notes) > 0 {
		var nb strings.Builder
		for _, n := range notes {
			fmt.Fprintf(&nb, "[User Note: %s]\n%s\n[/User Note]\n\n", n.Title, n.Content)
		}
		notesStr = nb.String()
	}

	toolsStr := ""
	if len(tools) > 0 {
		var tb strings.Builder
		tb.WriteString("Available tools:\n")
		for _, t := range tools {
			fmt.Fprintf(&tb, "- %s: %s\n", t.Name, t.Description)
		}
		tb.WriteString("\nUse tools as needed to find the best answer.")
		toolsStr = tb.String()
	}

	if client != nil {
		prompt, err := client.GetPrompt("alicia/agent/system-prompt", langfuse.WithLabel("production"))
		if err != nil {
			slog.Warn("failed to fetch system prompt from langfuse, using default", "error", err)
		} else {
			vars := map[string]string{"memories": memoriesStr, "tools": toolsStr, "instructions": instructions}
			if notesStr != "" {
				vars["notes"] = notesStr
			}
			return PromptResult{
				Text:    prompt.Compile(vars),
				Name:    prompt.Name,
				Version: prompt.Version,
			}
		}
	}

	system := "You are Alicia, a helpful AI assistant.\n"
	if memoriesStr != "" {
		system += "\n" + memoriesStr
	}
	if notesStr != "" {
		system += "\n" + notesStr
	}
	if toolsStr != "" {
		system += "\n" + toolsStr
	}
	if instructions != "" {
		system += "\n" + instructions
	}
	return PromptResult{Text: system}
}

func getContinuePrompt() string {
	client := getLangfuseClient()

	if client != nil {
		prompt, err := client.GetPrompt("alicia/agent/continue-response", langfuse.WithLabel("production"))
		if err != nil {
			slog.Warn("failed to fetch continue prompt from langfuse, using default", "error", err)
		} else {
			return prompt.GetText()
		}
	}

	return "Please continue your previous response."
}

func getTitlePrompt(userMsg, assistantMsg string) PromptResult {
	client := getLangfuseClient()

	vars := map[string]string{"user_message": userMsg}
	if assistantMsg != "" {
		vars["assistant_message"] = assistantMsg
	}

	if client != nil {
		prompt, err := client.GetPrompt("alicia/agent/conversation-title", langfuse.WithLabel("production"))
		if err != nil {
			slog.Warn("failed to fetch title prompt from langfuse, using default", "error", err)
		} else {
			return PromptResult{
				Text:    prompt.Compile(vars),
				Name:    prompt.Name,
				Version: prompt.Version,
			}
		}
	}

	// Fallback prompt
	text := "Generate a short title (under 50 chars) for this conversation.\n\nUser message: " + userMsg
	if assistantMsg != "" {
		text += "\nAssistant response: " + assistantMsg
	}
	text += "\n\nRespond with ONLY the title, no quotes or explanation."
	return PromptResult{Text: text}
}

func generateTitle(ctx context.Context, llm *LLMClient, userMsg, assistantMsg string) (string, error) {
	prompt := getTitlePrompt(
		langfuse.TruncateString(userMsg, 500, "..."),
		langfuse.TruncateString(assistantMsg, 500, "..."),
	)

	msgs := []LLMMessage{{Role: "user", Content: prompt.Text}}
	resp, err := llm.ChatWithOptions(ctx, msgs, nil, ChatOptions{
		GenerationName: "agent.generate_title",
		PromptName:     prompt.Name,
		PromptVersion:  prompt.Version,
	})
	if err != nil {
		return "", err
	}

	title := strings.TrimSpace(resp.Content)
	title = strings.Trim(title, "\"'")
	if len(title) > 50 {
		title = title[:47] + "..."
	}
	return title, nil
}

func maybeUpdateTitle(ctx context.Context, deps AgentDeps, convID, userMsg, assistantMsg string) {
	currentTitle, err := GetConversationTitle(ctx, deps.DB, convID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get conversation title", "conversation_id", convID, "error", err)
		return
	}

	if currentTitle != "" && currentTitle != "New Chat" {
		return
	}

	title, err := generateTitle(ctx, deps.LLM, userMsg, assistantMsg)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate title", "conversation_id", convID, "error", err)
		return
	}

	if err := UpdateConversationTitle(ctx, deps.DB, convID, title); err != nil {
		slog.ErrorContext(ctx, "failed to update title", "conversation_id", convID, "error", err)
		return
	}

	deps.Notifier.SendTitleUpdate(ctx, title)
	slog.InfoContext(ctx, "updated conversation title", "conversation_id", convID, "title", title)
}

type AgentDeps struct {
	DB         *pgxpool.Pool
	LLM        *LLMClient
	MCP        *MCPManager
	Notifier   Notifier
	Prefs      *PreferencesStore
	ParetoMode bool
	UserID     string
}

func HandleSend(ctx context.Context, req ResponseGenerationRequest, deps AgentDeps) error {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "agent.handle_send",
		trace.WithAttributes(
			attribute.String("conversation_id", req.ConversationID),
			attribute.String("message_id", req.MessageID),
			attribute.String("request_type", req.RequestType),
			attribute.String(otel.AttrTraceName, "agent:send"),
			attribute.String(otel.AttrAliciaType, otel.AliciaResponseTag),
		))
	defer span.End()

	cfg := GenerateConfig{
		MaxToolIterations: 5,
		EnableTools:       req.EnableTools,
		ParetoMode:        deps.ParetoMode && req.UsePareto,
	}

	if cfg.ParetoMode {
		return generateParetoResponse(ctx, req.ConversationID, req.MessageID, req.MessageID, cfg, deps)
	}
	return generateResponse(ctx, req.ConversationID, req.MessageID, req.MessageID, cfg, deps)
}

func HandleRegenerate(ctx context.Context, req ResponseGenerationRequest, deps AgentDeps) error {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "agent.handle_regenerate",
		trace.WithAttributes(
			attribute.String("message_id", req.MessageID),
			attribute.String("request_type", req.RequestType),
			attribute.String(otel.AttrTraceName, "agent:regenerate"),
			attribute.String(otel.AttrAliciaType, otel.AliciaResponseTag),
		))
	defer span.End()

	convID, err := GetConversationIDForMessage(ctx, deps.DB, req.MessageID)
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}
	span.SetAttributes(attribute.String("conversation_id", convID))

	userMsg, err := GetPreviousUserMessage(ctx, deps.DB, req.MessageID)
	if err != nil {
		return fmt.Errorf("get previous user message: %w", err)
	}
	if userMsg == nil {
		return fmt.Errorf("previous user message not found")
	}

	newMsgID := NewMessageID()
	if err := CreateMessage(ctx, deps.DB, newMsgID, convID, "assistant", "", "", &userMsg.ID); err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	if err := UpdateConversationTip(ctx, deps.DB, convID, newMsgID); err != nil {
		slog.ErrorContext(ctx, "failed to update conversation tip", "error", err)
	}

	cfg := GenerateConfig{
		MaxToolIterations: 5,
		EnableTools:       req.EnableTools,
		ParetoMode:        deps.ParetoMode && req.UsePareto,
	}

	if cfg.ParetoMode {
		paretoCfg := GetParetoConfig(deps.Prefs, deps.UserID)
		return runParetoExploration(ctx, convID, newMsgID, userMsg.ID, userMsg.Content, cfg, paretoCfg, deps)
	}
	return runToolLoop(ctx, convID, newMsgID, userMsg.ID, userMsg.Content, cfg, deps)
}

func HandleContinue(ctx context.Context, req ResponseGenerationRequest, deps AgentDeps) error {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "agent.handle_continue",
		trace.WithAttributes(
			attribute.String("message_id", req.MessageID),
			attribute.String("request_type", req.RequestType),
			attribute.String(otel.AttrTraceName, "agent:continue"),
			attribute.String(otel.AttrAliciaType, otel.AliciaResponseTag),
		))
	defer span.End()

	convID, err := GetConversationIDForMessage(ctx, deps.DB, req.MessageID)
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}
	span.SetAttributes(attribute.String("conversation_id", convID))

	msg, err := GetMessage(ctx, deps.DB, req.MessageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}

	cfg := GenerateConfig{
		MaxToolIterations: 5,
		EnableTools:       req.EnableTools,
	}
	return continueResponse(ctx, convID, msg, cfg, deps)
}

func HandleEdit(ctx context.Context, req ResponseGenerationRequest, deps AgentDeps) error {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "agent.handle_edit",
		trace.WithAttributes(
			attribute.String("conversation_id", req.ConversationID),
			attribute.String("message_id", req.MessageID),
			attribute.String("request_type", req.RequestType),
			attribute.String(otel.AttrTraceName, "agent:edit"),
			attribute.String(otel.AttrAliciaType, otel.AliciaResponseTag),
		))
	defer span.End()

	cfg := GenerateConfig{
		MaxToolIterations: 5,
		EnableTools:       req.EnableTools,
		ParetoMode:        deps.ParetoMode && req.UsePareto,
	}

	if cfg.ParetoMode {
		return generateParetoResponse(ctx, req.ConversationID, req.MessageID, req.MessageID, cfg, deps)
	}
	return generateResponse(ctx, req.ConversationID, req.MessageID, req.MessageID, cfg, deps)
}

func setupResponseMessage(ctx context.Context, convID, userMsgID, previousID string, deps AgentDeps) (msgID string, userContent string, err error) {
	userMsg, err := GetMessage(ctx, deps.DB, userMsgID)
	if err != nil {
		deps.Notifier.SendError(ctx, "", fmt.Errorf("get user message: %w", err))
		return "", "", err
	}

	msgID = NewMessageID()
	var prevPtr *string
	if previousID != "" {
		prevPtr = &previousID
	}
	if err := CreateMessage(ctx, deps.DB, msgID, convID, "assistant", "", "", prevPtr); err != nil {
		deps.Notifier.SendError(ctx, "", fmt.Errorf("create message: %w", err))
		return "", "", err
	}
	if err := UpdateConversationTip(ctx, deps.DB, convID, msgID); err != nil {
		slog.ErrorContext(ctx, "failed to update conversation tip", "error", err)
	}
	return msgID, userMsg.Content, nil
}

func generateResponse(ctx context.Context, convID, userMsgID, previousID string, cfg GenerateConfig, deps AgentDeps) error {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "response.generation",
		trace.WithAttributes(
			attribute.String("conversation_id", convID),
			attribute.String("message_id", userMsgID),
		))
	defer span.End()

	msgID, userContent, err := setupResponseMessage(ctx, convID, userMsgID, previousID, deps)
	if err != nil {
		return err
	}
	return runToolLoop(ctx, convID, msgID, previousID, userContent, cfg, deps)
}

func generateParetoResponse(ctx context.Context, convID, userMsgID, previousID string, cfg GenerateConfig, deps AgentDeps) error {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "pareto.response_generation",
		trace.WithAttributes(
			attribute.String("conversation_id", convID),
			attribute.String("message_id", userMsgID),
		))
	defer span.End()

	msgID, userContent, err := setupResponseMessage(ctx, convID, userMsgID, previousID, deps)
	if err != nil {
		return err
	}
	paretoCfg := GetParetoConfig(deps.Prefs, deps.UserID)
	return runParetoExploration(ctx, convID, msgID, previousID, userContent, cfg, paretoCfg, deps)
}

func continueResponse(ctx context.Context, convID string, msg *Message, cfg GenerateConfig, deps AgentDeps) error {
	deps.Notifier.SendThinking(ctx, msg.ID, "Continuing response...")

	messages, err := LoadConversationFull(ctx, deps.DB, convID)
	if err != nil {
		deps.Notifier.SendError(ctx, msg.ID, fmt.Errorf("load conversation: %w", err))
		return err
	}

	var tools []Tool
	if cfg.EnableTools {
		var toolsErr error
		tools, toolsErr = LoadTools(ctx, deps.DB)
		if toolsErr != nil {
			slog.ErrorContext(ctx, "failed to load tools", "error", toolsErr)
		}
		if deps.MCP != nil {
			tools = append(tools, deps.MCP.Tools()...)
		}
	}

	llmMsgs, systemPrompt := buildLLMMessages(messages, nil, nil, tools)
	llmMsgs = append(llmMsgs, LLMMessage{Role: "user", Content: getContinuePrompt()})

	if systemPrompt.Name != "" {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(
			attribute.String(otel.AttrPromptName, systemPrompt.Name),
			attribute.Int(otel.AttrPromptVersion, systemPrompt.Version),
		)
	}

	userPrefs := deps.Prefs.Get(deps.UserID)
	resp, err := deps.LLM.ChatWithOptions(ctx, llmMsgs, tools, ChatOptions{
		Temperature:    float32Ptr(userPrefs.Temperature),
		GenerationName: "agent.continue_response",
		PromptName:     systemPrompt.Name,
		PromptVersion:  systemPrompt.Version,
	})
	if err != nil {
		deps.Notifier.SendError(ctx, msg.ID, err)
		return err
	}

	fullContent := msg.Content + resp.Content
	reasoning := msg.Reasoning
	if resp.Reasoning != "" {
		if reasoning != "" {
			reasoning += "\n\n"
		}
		reasoning += resp.Reasoning
	}
	if err := UpdateMessage(ctx, deps.DB, msg.ID, fullContent, reasoning, "completed"); err != nil {
		deps.Notifier.SendError(ctx, msg.ID, err)
		return err
	}

	deps.Notifier.SendComplete(ctx, msg.ID, fullContent)
	slog.InfoContext(ctx, "response complete", "message_id", msg.ID, "content_length", len(fullContent))
	return nil
}

func runToolLoop(ctx context.Context, convID, msgID, previousID, userQuery string, cfg GenerateConfig, deps AgentDeps) error {
	ctx, span := otel.Tracer("alicia-agent").Start(ctx, "agent.tool_loop",
		trace.WithAttributes(
			attribute.String("conversation_id", convID),
			attribute.String("message_id", msgID),
			attribute.Int("max_iterations", cfg.MaxToolIterations),
			attribute.String(otel.AttrTraceName, "agent:tool_loop"),
			attribute.String(otel.AttrAliciaType, otel.AliciaResponseTag),
		))
	defer span.End()

	deps.Notifier.SetMessageID(msgID)
	deps.Notifier.SetPreviousID(previousID)
	deps.Notifier.SendThinking(ctx, msgID, "Processing request...")

	setupCtx, setupSpan := otel.Tracer("alicia-agent").Start(ctx, "response.setup",
		trace.WithAttributes(attribute.String("conversation_id", convID)))

	messages, err := LoadConversationFull(setupCtx, deps.DB, convID)
	if err != nil {
		slog.ErrorContext(setupCtx, "failed to load conversation", "conversation_id", convID, "error", err)
	}

	var memories []Memory
	embedding, err := deps.LLM.Embed(setupCtx, userQuery)
	if err != nil {
		slog.ErrorContext(setupCtx, "failed to generate embedding for memory search", "error", err)
	} else if embedding != nil {
		userPrefs := deps.Prefs.Get(deps.UserID)
		memories, err = SearchMemories(setupCtx, deps.DB, embedding, 0.7, userPrefs.MemoryRetrievalCount)
		if err != nil {
			slog.ErrorContext(setupCtx, "failed to search memories", "error", err)
		} else {
			for _, m := range memories {
				RecordMemoryUse(setupCtx, deps.DB, NewMemoryUseID(), m.ID, msgID, convID, m.Similarity)
				deps.Notifier.SendMemoryTrace(setupCtx, msgID, m.ID, m.Content, m.Similarity)
			}
		}
	}

	var notes []Note
	if embedding != nil && deps.UserID != "" {
		userPrefs := deps.Prefs.Get(deps.UserID)
		notes, err = SearchNotes(setupCtx, deps.DB, deps.UserID, embedding, userPrefs.NotesSimilarityThreshold, userPrefs.NotesMaxCount)
		if err != nil {
			slog.ErrorContext(setupCtx, "failed to search notes", "error", err)
		}
	}

	var tools []Tool
	if cfg.EnableTools {
		var toolsErr error
		tools, toolsErr = LoadTools(setupCtx, deps.DB)
		if toolsErr != nil {
			slog.ErrorContext(setupCtx, "failed to load tools", "error", toolsErr)
		}
		if deps.MCP != nil {
			tools = append(tools, deps.MCP.Tools()...)
		}
	}

	llmMsgs, systemPrompt := buildLLMMessages(messages, memories, notes, tools)

	setupSpan.SetAttributes(
		attribute.Int("memory_count", len(memories)),
		attribute.Int("tool_count", len(tools)),
	)
	if systemPrompt.Name != "" {
		setupSpan.SetAttributes(
			attribute.String(otel.AttrPromptName, systemPrompt.Name),
			attribute.Int(otel.AttrPromptVersion, systemPrompt.Version),
		)
	}
	setupSpan.End()

	span.SetAttributes(
		attribute.Int("memory_count", len(memories)),
		attribute.Int("tool_count", len(tools)),
	)
	if systemPrompt.Name != "" {
		span.SetAttributes(
			attribute.String(otel.AttrPromptName, systemPrompt.Name),
			attribute.Int(otel.AttrPromptVersion, systemPrompt.Version),
		)
	}

	var finalContent string
	var totalToolCalls int
	var reasoningParts []string

	userPrefs := deps.Prefs.Get(deps.UserID)
	temperature := float32Ptr(userPrefs.Temperature)

	for i := 0; i < cfg.MaxToolIterations; i++ {
		if i > 0 {
			deps.Notifier.SendThinking(ctx, msgID, fmt.Sprintf("Analyzing results (step %d)...", i+1))
		}

		llmAttrs := []attribute.KeyValue{
			attribute.Int("iteration", i+1),
			attribute.Int("message_count", len(llmMsgs)),
		}
		if systemPrompt.Name != "" {
			llmAttrs = append(llmAttrs,
				attribute.String(otel.AttrPromptName, systemPrompt.Name),
				attribute.Int(otel.AttrPromptVersion, systemPrompt.Version),
			)
		}
		llmCtx, llmSpan := otel.Tracer("alicia-agent").Start(ctx, "llm.chat",
			trace.WithAttributes(llmAttrs...))

		llmStart := time.Now()
		resp, err := deps.LLM.ChatWithOptions(llmCtx, llmMsgs, tools, ChatOptions{
			Temperature:    temperature,
			GenerationName: "agent.tool_loop",
			PromptName:     systemPrompt.Name,
			PromptVersion:  systemPrompt.Version,
		})
		if err != nil {
			llmSpan.RecordError(err)
			llmSpan.End()
			deps.Notifier.SendError(ctx, msgID, err)
			return err
		}
		llmEnd := time.Now()

		llmSpan.SetAttributes(
			attribute.Int("response_length", len(resp.Content)),
			attribute.Int("tool_calls", len(resp.ToolCalls)),
		)
		llmSpan.End()

		if resp.Reasoning != "" {
			reasoningParts = append(reasoningParts, resp.Reasoning)
		}

		if systemPrompt.Name != "" {
			traceID := span.SpanContext().TraceID().String()
			genID := llmSpan.SpanContext().SpanID().String()
			go sendGenerationToLangfuse(traceID, genID, convID, deps.UserID, deps.LLM.model, "agent:tool_loop", "llm.chat", systemPrompt, resp.Content, llmStart, llmEnd)
		}

		if len(resp.ToolCalls) == 0 {
			finalContent = resp.Content
			break
		}

		llmMsgs = append(llmMsgs, LLMMessage{Role: "assistant", Content: resp.Content, ToolCalls: resp.ToolCalls})

		toolNames := make([]string, len(resp.ToolCalls))
		for j, tc := range resp.ToolCalls {
			toolNames[j] = tc.Name
		}
		deps.Notifier.SendThinking(ctx, msgID, fmt.Sprintf("Using %s...", strings.Join(toolNames, ", ")))

		for _, tc := range resp.ToolCalls {
			deps.Notifier.SendToolStart(ctx, tc.ID, tc.Name, tc.Arguments)

			mcpName := tc.Name
			if strings.HasPrefix(mcpName, "mcp_garden_") {
				mcpName = strings.TrimPrefix(mcpName, "mcp_garden_")
			}

			// Create span for tool execution
			toolCtx, toolSpan := otel.Tracer("alicia-agent").Start(ctx, "tool.execute",
				trace.WithAttributes(
					attribute.String("tool.name", tc.Name),
					attribute.String("tool.id", tc.ID),
				))

			if deps.MCP == nil {
				toolSpan.End()
				deps.Notifier.SendError(ctx, msgID, fmt.Errorf("MCP not available"))
				return fmt.Errorf("MCP not available for tool call: %s", tc.Name)
			}

			result, execErr := deps.MCP.Call(toolCtx, mcpName, tc.Arguments)
			totalToolCalls++

			tu := ToolUse{ID: tc.ID, ToolName: tc.Name, Arguments: tc.Arguments}
			var toolMsg LLMMessage

			if execErr != nil {
				tu.Success = false
				tu.Error = execErr.Error()
				toolSpan.RecordError(execErr)
				toolSpan.SetAttributes(attribute.Bool("tool.success", false))
				deps.Notifier.SendToolComplete(ctx, tc.ID, false, nil, execErr.Error())
				toolMsg = LLMMessage{Role: "tool", Content: "Error: " + execErr.Error(), ToolCallID: tc.ID}
			} else {
				tu.Success = true
				tu.Result = result
				toolSpan.SetAttributes(attribute.Bool("tool.success", true))
				deps.Notifier.SendToolComplete(ctx, tc.ID, true, result, "")
				toolMsg = LLMMessage{Role: "tool", Content: fmt.Sprintf("%v", result), ToolCallID: tc.ID}
			}
			toolSpan.End()

			llmMsgs = append(llmMsgs, toolMsg)
			SaveToolUse(ctx, deps.DB, msgID, tu)
		}

		if i == cfg.MaxToolIterations-1 {
			finalContent = resp.Content
			if finalContent == "" {
				finalContent = "Max tool iterations reached."
			}
		}
	}

	span.SetAttributes(attribute.Int("total_tool_calls", totalToolCalls))

	finalContent = strings.TrimSpace(finalContent)
	reasoning := strings.Join(reasoningParts, "\n\n")
	if err := UpdateMessage(ctx, deps.DB, msgID, finalContent, reasoning, "completed"); err != nil {
		deps.Notifier.SendError(ctx, msgID, err)
		return err
	}

	deps.Notifier.SendComplete(ctx, msgID, finalContent)
	slog.InfoContext(ctx, "response complete", "message_id", msgID, "content_length", len(finalContent))

	// Detached context with timeout: title update must complete even if client disconnects
	titleCtx, titleCancel := context.WithTimeout(context.Background(), 45*time.Second)
	go func() {
		defer titleCancel()
		maybeUpdateTitle(titleCtx, deps, convID, userQuery, finalContent)
	}()

	// Extract and save memories asynchronously (detached context to survive client disconnect)
	// Carry span context for Langfuse score ingestion
	memCtx := trace.ContextWithSpanContext(context.Background(), trace.SpanFromContext(ctx).SpanContext())
	go ExtractAndSaveMemories(memCtx, convID, deps)

	return nil
}

func buildLLMMessages(history []Message, newMemories []Memory, notes []Note, tools []Tool) ([]LLMMessage, PromptResult) {
	var msgs []LLMMessage

	memorySet := make(map[string]Memory)
	for _, m := range history {
		for _, mem := range m.Memories {
			memorySet[mem.ID] = mem
		}
	}
	for _, mem := range newMemories {
		memorySet[mem.ID] = mem
	}

	memories := make([]Memory, 0, len(memorySet))
	for _, m := range memorySet {
		memories = append(memories, m)
	}

	systemPrompt := getSystemPrompt(memories, notes, tools, "")
	msgs = append(msgs, LLMMessage{Role: "system", Content: systemPrompt.Text})

	for _, m := range history {
		if m.Role == "system" {
			continue
		}

		if m.Role == "assistant" && len(m.ToolUses) > 0 {
			toolCalls := make([]LLMToolCall, len(m.ToolUses))
			for i, tu := range m.ToolUses {
				toolCalls[i] = LLMToolCall{
					ID:        tu.ID,
					Name:      tu.ToolName,
					Arguments: tu.Arguments,
				}
			}
			msgs = append(msgs, LLMMessage{Role: "assistant", Content: m.Content, ToolCalls: toolCalls})
			for _, tu := range m.ToolUses {
				content := fmt.Sprintf("%v", tu.Result)
				if !tu.Success {
					content = "Error: " + tu.Error
				}
				msgs = append(msgs, LLMMessage{Role: "tool", Content: content, ToolCallID: tu.ID})
			}
		} else {
			msgs = append(msgs, LLMMessage{Role: m.Role, Content: m.Content})
		}
	}
	return msgs, systemPrompt
}

func sendGenerationToLangfuse(traceID, genID, convID, userID, model, traceName, generationName string, prompt PromptResult, output string, start, end time.Time) {
	client := getLangfuseClient()
	if client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.CreateTrace(ctx, langfuse.TraceParams{
		ID:        traceID,
		Name:      traceName,
		SessionID: convID,
		UserID:    userID,
	}); err != nil {
		slog.Warn("langfuse: failed to create trace for generation", "error", err)
	}

	if err := client.CreateGeneration(ctx, langfuse.GenerationParams{
		TraceID:       traceID,
		ID:            genID,
		Name:          generationName,
		Model:         model,
		PromptName:    prompt.Name,
		PromptVersion: prompt.Version,
		Output:        langfuse.TruncateString(output, 10000, "..."),
		StartTime:     start,
		EndTime:       end,
	}); err != nil {
		slog.Warn("langfuse: failed to create generation", "error", err)
	}
}

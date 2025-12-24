package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/llm"
	"github.com/spf13/cobra"
)

// chatCmd creates the chat command for interactive conversations
func chatCmd() *cobra.Command {
	var title string

	cmd := &cobra.Command{
		Use:   "chat [conversation-id]",
		Short: "Interactive chat with Alicia",
		Long: `Start an interactive chat session with Alicia.
Provide a conversation ID to continue an existing conversation, or omit it to create a new one.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			conversationRepo := postgres.NewConversationRepository(pool)
			messageRepo := postgres.NewMessageRepository(pool)
			idGen := id.New()

			var conversation *models.Conversation

			// Get or create conversation
			if len(args) > 0 {
				// Continue existing conversation
				conversationID := args[0]
				conversation, err = conversationRepo.GetByID(ctx, conversationID)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return fmt.Errorf("conversation not found: %s", conversationID)
					}
					return fmt.Errorf("failed to get conversation: %w", err)
				}
				fmt.Printf("Continuing conversation: %s\n", conversation.Title)
			} else {
				// Create new conversation
				if title == "" {
					title = fmt.Sprintf("Chat %s", time.Now().Format("2006-01-02 15:04"))
				}
				conversation = models.NewConversation(idGen.GenerateConversationID(), "default-user", title)
				if err := conversationRepo.Create(ctx, conversation); err != nil {
					return fmt.Errorf("failed to create conversation: %w", err)
				}
				fmt.Printf("Started new conversation: %s\n", conversation.Title)
				fmt.Printf("ID: %s\n", conversation.ID)
			}

			fmt.Println("\nType your message and press Enter. Type 'exit' or 'quit' to end the conversation.")
			fmt.Println(strings.Repeat("-", 80))
			fmt.Println()

			scanner := bufio.NewScanner(os.Stdin)

			for {
				fmt.Print("You: ")
				if !scanner.Scan() {
					break
				}

				userInput := strings.TrimSpace(scanner.Text())
				if userInput == "" {
					continue
				}

				if strings.ToLower(userInput) == "exit" || strings.ToLower(userInput) == "quit" {
					fmt.Println("\nGoodbye!")
					break
				}

				// Get next sequence number
				seqNum, err := messageRepo.GetNextSequenceNumber(ctx, conversation.ID)
				if err != nil {
					return fmt.Errorf("failed to get sequence number: %w", err)
				}

				// Create and save user message
				userMessage := models.NewUserMessage(idGen.GenerateMessageID(), conversation.ID, seqNum, userInput)
				if err := messageRepo.Create(ctx, userMessage); err != nil {
					return fmt.Errorf("failed to save user message: %w", err)
				}

				// Get conversation history for LLM
				messages, err := messageRepo.GetByConversation(ctx, conversation.ID)
				if err != nil {
					return fmt.Errorf("failed to get message history: %w", err)
				}

				// Convert to LLM format
				llmMessages := make([]llm.ChatMessage, 0, len(messages))
				for _, msg := range messages {
					llmMessages = append(llmMessages, llm.ChatMessage{
						Role:    string(msg.Role),
						Content: msg.Contents,
					})
				}

				// Stream response from LLM
				fmt.Print("Alicia: ")
				responseChunks, err := llmClient.ChatStream(ctx, llmMessages)
				if err != nil {
					return fmt.Errorf("failed to get LLM response: %w", err)
				}

				var fullResponse strings.Builder
				for chunk := range responseChunks {
					if chunk.Error != nil {
						return fmt.Errorf("error during streaming: %w", chunk.Error)
					}
					if chunk.Content != "" {
						fmt.Print(chunk.Content)
						fullResponse.WriteString(chunk.Content)
					}
				}
				fmt.Println()

				// Get next sequence number for assistant message
				seqNum, err = messageRepo.GetNextSequenceNumber(ctx, conversation.ID)
				if err != nil {
					return fmt.Errorf("failed to get sequence number: %w", err)
				}

				// Save assistant response
				assistantMessage := models.NewAssistantMessage(
					idGen.GenerateMessageID(),
					conversation.ID,
					seqNum,
					fullResponse.String(),
				)
				if err := messageRepo.Create(ctx, assistantMessage); err != nil {
					return fmt.Errorf("failed to save assistant message: %w", err)
				}

				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Title for the new conversation (only used when creating)")

	return cmd
}

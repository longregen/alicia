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
	"github.com/longregen/alicia/internal/application/usecases"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
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

			// Initialize repositories
			conversationRepo := postgres.NewConversationRepository(pool)
			messageRepo := postgres.NewMessageRepository(pool)
			idGen := id.New()
			txManager := postgres.NewTransactionManager(pool)

			// Create use cases using the same path as serve/agent
			simpleGenerateResponse := usecases.NewSimpleGenerateResponse(
				messageRepo,
				conversationRepo,
				llmClient,
				idGen,
			)

			processUserMessage := usecases.NewProcessUserMessage(
				messageRepo,
				nil, // audioRepo - not needed for text chat
				conversationRepo,
				nil, // asrService - not needed for text chat
				nil, // memoryService - optional
				idGen,
				txManager,
			)

			sendMessage := usecases.NewSendMessage(
				conversationRepo,
				messageRepo,
				processUserMessage,
				simpleGenerateResponse,
				txManager,
			)

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

				// Use SendMessage use case - same path as HTTP API and agent
				input := &ports.SendMessageInput{
					ConversationID:  conversation.ID,
					TextContent:     userInput,
					EnableStreaming: true,
				}

				output, err := sendMessage.Execute(ctx, input)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					continue
				}

				// Stream the response
				fmt.Print("Alicia: ")
				if output.StreamChannel != nil {
					for chunk := range output.StreamChannel {
						if chunk.Error != nil {
							fmt.Printf("\nError: %v\n", chunk.Error)
							break
						}
						if chunk.Text != "" {
							fmt.Print(chunk.Text)
						}
					}
				} else if output.AssistantMessage != nil {
					fmt.Print(output.AssistantMessage.Contents)
				}
				fmt.Println()
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Title for the new conversation (only used when creating)")

	return cmd
}

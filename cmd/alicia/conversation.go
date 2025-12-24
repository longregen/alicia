package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/spf13/cobra"
)

// newCmd creates a new conversation
func newCmd() *cobra.Command {
	var title string

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new conversation",
		Long:  `Create a new conversation with an optional title.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			conversationRepo := postgres.NewConversationRepository(pool)
			idGen := id.New()

			// Generate a title if not provided
			if title == "" {
				title = fmt.Sprintf("Conversation %s", time.Now().Format("2006-01-02 15:04"))
			}

			// Create new conversation
			conversation := models.NewConversation(idGen.GenerateConversationID(), "default-user", title)

			if err := conversationRepo.Create(ctx, conversation); err != nil {
				return fmt.Errorf("failed to create conversation: %w", err)
			}

			fmt.Printf("Created conversation: %s\n", conversation.ID)
			fmt.Printf("Title: %s\n", conversation.Title)

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Title for the conversation")

	return cmd
}

// listCmd lists conversations
func listCmd() *cobra.Command {
	var all bool
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List conversations",
		Long:  `List all conversations with their ID, title, and creation date.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			conversationRepo := postgres.NewConversationRepository(pool)

			var conversations []*models.Conversation
			if all {
				conversations, err = conversationRepo.List(ctx, limit, 0)
			} else {
				conversations, err = conversationRepo.ListActive(ctx, limit, 0)
			}

			if err != nil {
				return fmt.Errorf("failed to list conversations: %w", err)
			}

			if len(conversations) == 0 {
				fmt.Println("No conversations found.")
				return nil
			}

			// Print header
			fmt.Printf("%-30s %-40s %-10s %s\n", "ID", "Title", "Status", "Created")
			fmt.Println(strings.Repeat("-", 100))

			// Print conversations
			for _, conv := range conversations {
				createdAt := conv.CreatedAt.Format("2006-01-02 15:04")
				fmt.Printf("%-30s %-40s %-10s %s\n", conv.ID, conv.Title, conv.Status, createdAt)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include archived conversations")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of conversations to list")

	return cmd
}

// showCmd shows messages in a conversation
func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <conversation-id>",
		Short: "Show messages in a conversation",
		Long:  `Display all messages in the specified conversation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			conversationID := args[0]

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			conversationRepo := postgres.NewConversationRepository(pool)
			messageRepo := postgres.NewMessageRepository(pool)

			// Get conversation details
			conversation, err := conversationRepo.GetByID(ctx, conversationID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("conversation not found: %s", conversationID)
				}
				return fmt.Errorf("failed to get conversation: %w", err)
			}

			// Get messages
			messages, err := messageRepo.GetByConversation(ctx, conversationID)
			if err != nil {
				return fmt.Errorf("failed to get messages: %w", err)
			}

			// Display conversation header
			fmt.Printf("Conversation: %s\n", conversation.Title)
			fmt.Printf("ID: %s\n", conversation.ID)
			fmt.Printf("Status: %s\n", conversation.Status)
			fmt.Printf("Created: %s\n\n", conversation.CreatedAt.Format("2006-01-02 15:04:05"))

			if len(messages) == 0 {
				fmt.Println("No messages in this conversation.")
				return nil
			}

			// Display messages
			for i, msg := range messages {
				if i > 0 {
					fmt.Println()
				}
				fmt.Printf("[%s] %s:\n", msg.CreatedAt.Format("15:04:05"), msg.Role)
				fmt.Println(msg.Contents)
				fmt.Println(strings.Repeat("-", 80))
			}

			return nil
		},
	}
}

// deleteCmd deletes a conversation
func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <conversation-id>",
		Short: "Delete a conversation",
		Long:  `Delete the specified conversation (soft delete).`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			conversationID := args[0]

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			conversationRepo := postgres.NewConversationRepository(pool)

			// Verify conversation exists
			conversation, err := conversationRepo.GetByID(ctx, conversationID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("conversation not found: %s", conversationID)
				}
				return fmt.Errorf("failed to get conversation: %w", err)
			}

			// Delete conversation
			if err := conversationRepo.Delete(ctx, conversationID); err != nil {
				return fmt.Errorf("failed to delete conversation: %w", err)
			}

			fmt.Printf("Deleted conversation: %s\n", conversation.Title)
			fmt.Printf("ID: %s\n", conversationID)

			return nil
		},
	}
}

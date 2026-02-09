package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// inflightResult holds the result of an in-progress conversation creation.
type inflightResult struct {
	done chan struct{}
	id   string
	err  error
}

type Bridge struct {
	cfg      *Config
	archive  *Archive
	client   *http.Client
	convs    map[string]string
	inflight map[string]*inflightResult
	convMu   sync.Mutex
}

func NewBridge(cfg *Config, archive *Archive) *Bridge {
	return &Bridge{
		cfg:      cfg,
		archive:  archive,
		convs:    make(map[string]string),
		inflight: make(map[string]*inflightResult),
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (b *Bridge) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", b.cfg.AliciaUserID)
	if b.cfg.AgentSecret != "" {
		req.Header.Set("Authorization", "Bearer "+b.cfg.AgentSecret)
	}
}

func (b *Bridge) EnsureConversationForContact(ctx context.Context, contactJID, contactName string) (string, error) {
	b.convMu.Lock()

	// Fast path: already cached in memory.
	if id, ok := b.convs[contactJID]; ok {
		b.convMu.Unlock()
		return id, nil
	}

	// Check archive state under lock.
	stateKey := "conv:" + contactJID
	stored, err := b.archive.GetState(stateKey)
	if err != nil {
		b.convMu.Unlock()
		return "", fmt.Errorf("get stored conversation for %s: %w", contactJID, err)
	}
	if stored != "" {
		b.convs[contactJID] = stored
		b.convMu.Unlock()
		slog.Info("bridge: using stored conversation", "contact", contactJID, "conversation_id", stored)
		return stored, nil
	}

	// Another goroutine is already creating a conversation for this contact.
	// Wait for it to finish and use its result.
	if flight, ok := b.inflight[contactJID]; ok {
		b.convMu.Unlock()
		select {
		case <-flight.done:
			return flight.id, flight.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// Register ourselves as the inflight creator for this contact.
	flight := &inflightResult{done: make(chan struct{})}
	b.inflight[contactJID] = flight
	b.convMu.Unlock()

	// Perform the HTTP call WITHOUT holding the lock.
	id, err := b.createConversation(ctx, contactJID, contactName)

	// Store result and clean up inflight entry.
	b.convMu.Lock()
	delete(b.inflight, contactJID)
	if err == nil {
		// Double-check: another path may have populated the map (e.g. archive
		// state changed). Prefer the already-stored value if present.
		if existing, ok := b.convs[contactJID]; ok {
			id = existing
		} else {
			b.convs[contactJID] = id
			if persistErr := b.archive.SetState(stateKey, id); persistErr != nil {
				slog.Error("bridge: failed to persist conversation id", "contact", contactJID, "error", persistErr)
			}
			slog.Info("bridge: created conversation", "contact", contactJID, "contact_name", contactName, "conversation_id", id)
		}
	}
	b.convMu.Unlock()

	// Signal all waiters.
	flight.id = id
	flight.err = err
	close(flight.done)

	return id, err
}

// createConversation makes the HTTP POST to create a new conversation. It must
// be called WITHOUT holding convMu.
func (b *Bridge) createConversation(ctx context.Context, contactJID, contactName string) (string, error) {
	title := "WhatsApp: " + contactName
	if contactName == "" {
		title = "WhatsApp: " + contactJID
	}

	body, _ := json.Marshal(map[string]string{
		"title": title,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", b.cfg.AliciaAPIURL+"/conversations", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create conversation request: %w", err)
	}
	b.setHeaders(req)

	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create conversation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create conversation: status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode conversation response: %w", err)
	}

	return result.ID, nil
}

func (b *Bridge) SendMessageForContact(ctx context.Context, contactJID, contactName, text string) (string, error) {
	convID, err := b.EnsureConversationForContact(ctx, contactJID, contactName)
	if err != nil {
		return "", err
	}

	body, _ := json.Marshal(map[string]string{
		"content": text,
	})

	url := fmt.Sprintf("%s/conversations/%s/messages?sync=true", b.cfg.AliciaAPIURL, convID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create message request: %w", err)
	}
	b.setHeaders(req)

	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("send message: status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		AssistantMessage struct {
			Content string `json:"content"`
		} `json:"assistant_message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode message response: %w", err)
	}

	return result.AssistantMessage.Content, nil
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type contactMsg struct {
	chatJID     types.JID
	contactJID  string
	contactName string
	text        string
}

type WhatsAppClient struct {
	cfg         *Config
	role        string
	dbPath      string
	allowedJIDs map[string]bool
	ws          *WSClient
	bridge      *Bridge
	archive     *Archive
	client      *whatsmeow.Client
	clientMu    sync.RWMutex
	container   *sqlstore.Container

	contactChans   map[string]chan contactMsg
	contactChansMu sync.Mutex

	pairing      bool
	pairingMu    sync.Mutex
	reconnecting bool
	reconnMu     sync.Mutex
}

func NewWhatsAppClient(cfg *Config, role, dbPath string, ws *WSClient, bridge *Bridge, archive *Archive) *WhatsAppClient {
	w := &WhatsAppClient{
		cfg:          cfg,
		role:         role,
		dbPath:       dbPath,
		ws:           ws,
		bridge:       bridge,
		archive:      archive,
		contactChans: make(map[string]chan contactMsg),
	}

	if role == "alicia" && len(cfg.AllowedJIDs) > 0 {
		w.allowedJIDs = make(map[string]bool, len(cfg.AllowedJIDs))
		for _, jid := range cfg.AllowedJIDs {
			w.allowedJIDs[jid] = true
		}
	}

	return w
}

func (w *WhatsAppClient) Init(ctx context.Context) error {
	dbLog := waLog.Stdout("whatsmeow-db-"+w.role, "WARN", true)
	container, err := sqlstore.New(ctx, "sqlite", fmt.Sprintf("file:%s?_foreign_keys=on", w.dbPath), dbLog)
	if err != nil {
		return fmt.Errorf("[%s] open whatsmeow store: %w", w.role, err)
	}
	w.container = container

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("[%s] get device store: %w", w.role, err)
	}

	clientLog := waLog.Stdout("whatsmeow-"+w.role, "WARN", true)
	w.client = whatsmeow.NewClient(deviceStore, clientLog)
	w.client.AddEventHandler(w.handleEvent)

	if w.client.Store.ID != nil {
		slog.Info("whatsapp: existing session found, connecting", "role", w.role)
		if err := w.client.Connect(); err != nil {
			slog.Error("whatsapp: auto-connect failed", "role", w.role, "error", err)
			if err2 := w.ws.SendWhatsAppStatus(false, "", err.Error(), w.role); err2 != nil {
				slog.Debug("whatsapp: failed to send status", "role", w.role, "error", err2)
			}
			return nil // Don't fail startup, user can re-pair
		}
		phone := w.client.Store.ID.User
		slog.Info("whatsapp: connected", "role", w.role, "phone", phone)
		if err := w.ws.SendWhatsAppStatus(true, phone, "", w.role); err != nil {
			slog.Debug("whatsapp: failed to send status", "role", w.role, "error", err)
		}
	} else {
		slog.Info("whatsapp: no existing session, waiting for pair request", "role", w.role)
		if err := w.ws.SendWhatsAppStatus(false, "", "", w.role); err != nil {
			slog.Debug("whatsapp: failed to send status", "role", w.role, "error", err)
		}
	}

	return nil
}

func (w *WhatsAppClient) StartPairing(ctx context.Context) {
	w.pairingMu.Lock()
	if w.pairing {
		w.pairingMu.Unlock()
		slog.Warn("whatsapp: pairing already in progress", "role", w.role)
		return
	}
	w.pairing = true
	w.pairingMu.Unlock()

	defer func() {
		w.pairingMu.Lock()
		w.pairing = false
		w.pairingMu.Unlock()
	}()

	w.clientMu.Lock()
	if w.client != nil && w.client.IsConnected() {
		w.client.Disconnect()
	}
	w.clientMu.Unlock()

	deviceStore, err := w.container.GetFirstDevice(ctx)
	if err != nil {
		slog.Error("whatsapp: get device store for pairing", "role", w.role, "error", err)
		if err2 := w.ws.SendWhatsAppQR("", "error", w.role); err2 != nil {
			slog.Debug("whatsapp: failed to send QR error", "role", w.role, "error", err2)
		}
		return
	}

	clientLog := waLog.Stdout("whatsmeow-"+w.role, "WARN", true)
	newClient := whatsmeow.NewClient(deviceStore, clientLog)
	newClient.AddEventHandler(w.handleEvent)

	w.clientMu.Lock()
	w.client = newClient
	w.clientMu.Unlock()

	qrChan, err := newClient.GetQRChannel(ctx)
	if err != nil {
		slog.Error("whatsapp: get QR channel", "role", w.role, "error", err)
		if err2 := w.ws.SendWhatsAppQR("", "error", w.role); err2 != nil {
			slog.Debug("whatsapp: failed to send QR error", "role", w.role, "error", err2)
		}
		return
	}

	if err := newClient.Connect(); err != nil {
		slog.Error("whatsapp: connect for pairing", "role", w.role, "error", err)
		if err2 := w.ws.SendWhatsAppQR("", "error", w.role); err2 != nil {
			slog.Debug("whatsapp: failed to send QR error", "role", w.role, "error", err2)
		}
		return
	}

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			slog.Info("whatsapp: QR code received", "role", w.role)
			if err := w.ws.SendWhatsAppQR(evt.Code, "code", w.role); err != nil {
				slog.Debug("whatsapp: failed to send QR", "role", w.role, "error", err)
			}
		case "login":
			slog.Info("whatsapp: login successful", "role", w.role)
			if err := w.ws.SendWhatsAppQR("", "login", w.role); err != nil {
				slog.Debug("whatsapp: failed to send QR login", "role", w.role, "error", err)
			}
			w.clientMu.RLock()
			phone := ""
			if w.client.Store.ID != nil {
				phone = w.client.Store.ID.User
			}
			w.clientMu.RUnlock()
			if err := w.ws.SendWhatsAppStatus(true, phone, "", w.role); err != nil {
				slog.Debug("whatsapp: failed to send status", "role", w.role, "error", err)
			}
			return
		case "timeout":
			slog.Warn("whatsapp: QR timeout", "role", w.role)
			if err := w.ws.SendWhatsAppQR("", "timeout", w.role); err != nil {
				slog.Debug("whatsapp: failed to send QR timeout", "role", w.role, "error", err)
			}
			return
		default:
			slog.Warn("whatsapp: QR event", "role", w.role, "event", evt.Event)
			if err := w.ws.SendWhatsAppQR("", evt.Event, w.role); err != nil {
				slog.Debug("whatsapp: failed to send QR event", "role", w.role, "error", err)
			}
		}
	}
}

func (w *WhatsAppClient) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		w.handleMessage(v)
	case *events.HistorySync:
		w.handleHistorySync(v)
	case *events.Connected:
		slog.Info("whatsapp: connected event", "role", w.role)
	case *events.Disconnected:
		slog.Warn("whatsapp: disconnected", "role", w.role)
		if err := w.ws.SendWhatsAppStatus(false, "", "", w.role); err != nil {
			slog.Debug("whatsapp: failed to send disconnect status", "role", w.role, "error", err)
		}
		w.clientMu.RLock()
		client := w.client
		w.clientMu.RUnlock()
		if client != nil && client.Store.ID != nil {
			w.reconnMu.Lock()
			shouldReconnect := !w.reconnecting
			w.reconnecting = true
			w.reconnMu.Unlock()
			if shouldReconnect {
				go w.reconnectWithBackoff()
			}
		}
	case *events.LoggedOut:
		slog.Warn("whatsapp: logged out, clearing session", "role", w.role)
		w.clientMu.RLock()
		client := w.client
		w.clientMu.RUnlock()
		if client != nil {
			client.Disconnect()
		}
		if err := w.ws.SendWhatsAppStatus(false, "", "logged out", w.role); err != nil {
			slog.Debug("whatsapp: failed to send logout status", "role", w.role, "error", err)
		}
	}
}

func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if s := msg.GetConversation(); s != "" {
		return s
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}

func extractMediaType(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	switch {
	case msg.GetImageMessage() != nil:
		return "image"
	case msg.GetVideoMessage() != nil:
		return "video"
	case msg.GetAudioMessage() != nil:
		return "audio"
	case msg.GetDocumentMessage() != nil:
		return "document"
	default:
		return ""
	}
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func (w *WhatsAppClient) handleMessage(msg *events.Message) {
	archived := &ArchivedMessage{
		ID:         msg.Info.ID,
		ChatJID:    msg.Info.Chat.String(),
		SenderJID:  msg.Info.Sender.String(),
		SenderName: msg.Info.PushName,
		Content:    extractText(msg.Message),
		MediaType:  extractMediaType(msg.Message),
		Timestamp:  msg.Info.Timestamp,
		IsFromMe:   msg.Info.IsFromMe,
		IsGroup:    msg.Info.IsGroup,
	}

	contentPreview := truncateRunes(archived.Content, 80)
	msgDesc := fmt.Sprintf("from=%s name=%q chat=%s group=%v me=%v media=%s text=%q",
		archived.SenderJID, archived.SenderName, archived.ChatJID,
		archived.IsGroup, archived.IsFromMe, archived.MediaType, contentPreview)

	w.ws.SendWhatsAppDebug(w.role, "msg_recv", msgDesc)

	if err := w.archive.Store(archived); err != nil {
		slog.Error("whatsapp: archive message error", "role", w.role, "error", err)
		w.ws.SendWhatsAppDebug(w.role, "archive_err", err.Error())
	}

	// Reader role: archive only, never respond
	if w.role == "reader" {
		return
	}

	// Alicia role: respond to allowlisted contacts
	if msg.Info.IsFromMe {
		w.ws.SendWhatsAppDebug(w.role, "skip", "from_me")
		return
	}
	if msg.Info.IsGroup {
		w.ws.SendWhatsAppDebug(w.role, "skip", "group message")
		return
	}
	if archived.Content == "" {
		w.ws.SendWhatsAppDebug(w.role, "skip", fmt.Sprintf("empty content (media=%s)", archived.MediaType))
		return
	}

	// Normalize sender JID for allowlist lookup
	senderJID := msg.Info.Sender.ToNonAD().String()
	if w.allowedJIDs != nil && !w.allowedJIDs[senderJID] {
		slog.Debug("whatsapp: message from non-allowlisted contact", "role", w.role, "sender", senderJID)
		w.ws.SendWhatsAppDebug(w.role, "skip", fmt.Sprintf("not allowlisted: %s", senderJID))
		return
	}

	slog.Info("whatsapp: incoming message from contact", "role", w.role, "sender", senderJID, "name", msg.Info.PushName, "content_len", len(archived.Content))
	w.ws.SendWhatsAppDebug(w.role, "queued", fmt.Sprintf("contact=%s name=%q len=%d", senderJID, msg.Info.PushName, len(archived.Content)))

	w.contactChansMu.Lock()
	ch, ok := w.contactChans[senderJID]
	if !ok {
		ch = make(chan contactMsg, 32)
		w.contactChans[senderJID] = ch
		go w.processContactQueue(senderJID, ch)
	}
	w.contactChansMu.Unlock()

	ch <- contactMsg{
		chatJID:     msg.Info.Chat,
		contactJID:  senderJID,
		contactName: msg.Info.PushName,
		text:        archived.Content,
	}
}

func (w *WhatsAppClient) processContactQueue(senderJID string, ch chan contactMsg) {
	for m := range ch {
		w.respondToMessage(m.chatJID, m.contactJID, m.contactName, m.text)
	}
}

func (w *WhatsAppClient) respondToMessage(chatJID types.JID, contactJID, contactName, text string) {
	if w.bridge == nil {
		slog.Error("whatsapp: bridge is nil, cannot respond", "role", w.role, "contact", contactJID)
		w.ws.SendWhatsAppDebug(w.role, "error", "bridge is nil")
		return
	}

	w.ws.SendWhatsAppDebug(w.role, "bridge_call", fmt.Sprintf("contact=%s name=%q", contactJID, contactName))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	response, err := w.bridge.SendMessageForContact(ctx, contactJID, contactName, text)
	if err != nil {
		slog.Error("whatsapp: bridge send error", "role", w.role, "contact", contactJID, "error", err)
		w.ws.SendWhatsAppDebug(w.role, "bridge_err", fmt.Sprintf("contact=%s err=%s", contactJID, err))
		return
	}

	if response == "" {
		slog.Warn("whatsapp: empty response from alicia", "role", w.role, "contact", contactJID)
		w.ws.SendWhatsAppDebug(w.role, "bridge_empty", fmt.Sprintf("contact=%s", contactJID))
		return
	}

	if w.cfg.ResponsePrefix != "" {
		response = w.cfg.ResponsePrefix + response
	}

	// Re-acquire client pointer after the bridge call to avoid using a stale
	// pointer that may have been replaced by StartPairing during the wait.
	w.clientMu.RLock()
	client := w.client
	w.clientMu.RUnlock()
	if client == nil {
		slog.Error("whatsapp: client is nil after bridge call, cannot send", "role", w.role, "contact", contactJID)
		w.ws.SendWhatsAppDebug(w.role, "error", "client nil after bridge call")
		return
	}

	_, err = client.SendMessage(ctx, chatJID, &waE2E.Message{
		Conversation: proto.String(response),
	})
	if err != nil {
		slog.Error("whatsapp: send response error", "role", w.role, "contact", contactJID, "error", err)
		w.ws.SendWhatsAppDebug(w.role, "send_err", fmt.Sprintf("contact=%s err=%s", contactJID, err))
		return
	}

	responsePreview := truncateRunes(response, 80)
	slog.Info("whatsapp: response sent", "role", w.role, "contact", contactJID, "response_len", len(response))
	w.ws.SendWhatsAppDebug(w.role, "sent", fmt.Sprintf("contact=%s len=%d text=%q", contactJID, len(response), responsePreview))
}

func (w *WhatsAppClient) handleHistorySync(evt *events.HistorySync) {
	if evt.Data == nil {
		return
	}

	conversations := evt.Data.GetConversations()
	slog.Info("whatsapp: history sync", "role", w.role, "conversations", len(conversations))
	w.ws.SendWhatsAppDebug(w.role, "history_sync", fmt.Sprintf("conversations=%d", len(conversations)))

	for _, conv := range conversations {
		chatJID := conv.GetID()
		isGroup := strings.HasSuffix(chatJID, "@g.us")

		for _, histMsg := range conv.GetMessages() {
			wMsg := histMsg.GetMessage()
			if wMsg == nil || wMsg.GetMessage() == nil {
				continue
			}

			protoMsg := wMsg.GetMessage()
			msgKey := wMsg.GetKey()

			archived := &ArchivedMessage{
				ID:       msgKey.GetID(),
				ChatJID:  chatJID,
				IsFromMe: msgKey.GetFromMe(),
				IsGroup:  isGroup,
			}

			// Participant (actual sender in groups) takes precedence over RemoteJID (chat JID)
			if p := msgKey.GetParticipant(); p != "" {
				archived.SenderJID = p
			} else if r := msgKey.GetRemoteJID(); r != "" {
				archived.SenderJID = r
			}

			if name := wMsg.GetPushName(); name != "" {
				archived.SenderName = name
			}

			ts := wMsg.GetMessageTimestamp()
			if ts > 0 {
				archived.Timestamp = time.Unix(int64(ts), 0)
			}

			archived.Content = extractText(protoMsg)
			archived.MediaType = extractMediaType(protoMsg)

			if archived.Content != "" || archived.MediaType != "" {
				if err := w.archive.Store(archived); err != nil {
					slog.Debug("whatsapp: archive history message error", "role", w.role, "error", err)
				}
			}
		}
	}
}

func (w *WhatsAppClient) reconnectWithBackoff() {
	defer func() {
		w.reconnMu.Lock()
		w.reconnecting = false
		w.reconnMu.Unlock()
	}()

	delays := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second}
	for i, delay := range delays {
		time.Sleep(delay)

		w.pairingMu.Lock()
		isPairing := w.pairing
		w.pairingMu.Unlock()
		if isPairing {
			slog.Info("whatsapp: reconnect aborted, pairing in progress", "role", w.role)
			return
		}

		w.clientMu.RLock()
		client := w.client
		w.clientMu.RUnlock()

		if client == nil || client.Store.ID == nil {
			slog.Info("whatsapp: reconnect aborted, no stored session", "role", w.role)
			return
		}

		if client.IsConnected() {
			slog.Info("whatsapp: already reconnected", "role", w.role)
			return
		}

		slog.Info("whatsapp: reconnect attempt", "role", w.role, "attempt", i+1)
		if err := client.Connect(); err != nil {
			slog.Warn("whatsapp: reconnect failed", "role", w.role, "attempt", i+1, "error", err)
			continue
		}

		phone := client.Store.ID.User
		slog.Info("whatsapp: reconnected", "role", w.role, "phone", phone)
		if err := w.ws.SendWhatsAppStatus(true, phone, "", w.role); err != nil {
			slog.Debug("whatsapp: failed to send reconnect status", "role", w.role, "error", err)
		}
		return
	}
	slog.Error("whatsapp: all reconnect attempts failed", "role", w.role)
	if err := w.ws.SendWhatsAppStatus(false, "", "reconnection failed", w.role); err != nil {
		slog.Debug("whatsapp: failed to send reconnect failure status", "role", w.role, "error", err)
	}
}

func (w *WhatsAppClient) Close() {
	w.clientMu.Lock()
	if w.client != nil && w.client.IsConnected() {
		w.client.Disconnect()
	}
	w.clientMu.Unlock()

	if w.container != nil {
		if err := w.container.Close(); err != nil {
			slog.Error("whatsapp: failed to close container", "role", w.role, "error", err)
		}
	}
}

package livekit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/livekit/protocol/auth"
	lkproto "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/longregen/alicia/internal/ports"
)

type ServiceConfig struct {
	URL                   string
	APIKey                string
	APISecret             string
	TokenValidityDuration time.Duration
}

func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		URL:                   "ws://localhost:7880",
		APIKey:                "",
		APISecret:             "",
		TokenValidityDuration: 6 * time.Hour,
	}
}

type Service struct {
	config     *ServiceConfig
	roomClient *lksdk.RoomServiceClient
}

func NewService(config *ServiceConfig) (*Service, error) {
	if config == nil {
		config = DefaultServiceConfig()
	}

	if config.URL == "" {
		return nil, fmt.Errorf("LiveKit URL is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("LiveKit API key is required")
	}

	if config.APISecret == "" {
		return nil, fmt.Errorf("LiveKit API secret is required")
	}

	if config.TokenValidityDuration == 0 {
		config.TokenValidityDuration = 6 * time.Hour
	}

	roomClient := lksdk.NewRoomServiceClient(config.URL, config.APIKey, config.APISecret)

	return &Service{
		config:     config,
		roomClient: roomClient,
	}, nil
}

func (s *Service) CreateRoom(ctx context.Context, name string) (*ports.LiveKitRoom, error) {
	if name == "" {
		return nil, fmt.Errorf("room name is required")
	}

	metadataMap := map[string]string{
		"conversation_id": name,
		"created_at":      time.Now().UTC().Format(time.RFC3339),
	}
	metadata, err := json.Marshal(metadataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal room metadata: %w", err)
	}

	req := &lkproto.CreateRoomRequest{
		Name:            name,
		EmptyTimeout:    300,
		MaxParticipants: 2,
		Metadata:        string(metadata),
	}

	room, err := s.roomClient.CreateRoom(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	return &ports.LiveKitRoom{
		Name:         room.Name,
		SID:          room.Sid,
		Participants: []*ports.LiveKitParticipant{},
	}, nil
}

func (s *Service) GetRoom(ctx context.Context, name string) (*ports.LiveKitRoom, error) {
	if name == "" {
		return nil, fmt.Errorf("room name is required")
	}

	rooms, err := s.roomClient.ListRooms(ctx, &lkproto.ListRoomsRequest{
		Names: []string{name},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}

	if len(rooms.GetRooms()) == 0 {
		return nil, fmt.Errorf("room not found: %s", name)
	}

	// Safe to access rooms.GetRooms()[0] due to bounds check above
	room := rooms.GetRooms()[0]

	participants, err := s.ListParticipants(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	return &ports.LiveKitRoom{
		Name:         room.Name,
		SID:          room.Sid,
		Participants: participants,
	}, nil
}

func (s *Service) DeleteRoom(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("room name is required")
	}

	_, err := s.roomClient.DeleteRoom(ctx, &lkproto.DeleteRoomRequest{
		Room: name,
	})
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}

	return nil
}

func (s *Service) GenerateToken(ctx context.Context, roomName, participantID, participantName string) (*ports.LiveKitToken, error) {
	if roomName == "" {
		return nil, fmt.Errorf("room name is required")
	}

	if participantID == "" {
		return nil, fmt.Errorf("participant ID is required")
	}

	if participantName == "" {
		participantName = participantID
	}

	at := auth.NewAccessToken(s.config.APIKey, s.config.APISecret)
	canPublish := true
	canSubscribe := true
	canPublishData := true
	grant := &auth.VideoGrant{
		RoomJoin:       true,
		Room:           roomName,
		CanPublish:     &canPublish,
		CanSubscribe:   &canSubscribe,
		CanPublishData: &canPublishData,
	}

	at.SetVideoGrant(grant).
		SetIdentity(participantID).
		SetName(participantName).
		SetValidFor(s.config.TokenValidityDuration)

	token, err := at.ToJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	expiresAt := time.Now().Add(s.config.TokenValidityDuration).Unix()

	return &ports.LiveKitToken{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) ListParticipants(ctx context.Context, roomName string) ([]*ports.LiveKitParticipant, error) {
	if roomName == "" {
		return nil, fmt.Errorf("room name is required")
	}

	participants, err := s.roomClient.ListParticipants(ctx, &lkproto.ListParticipantsRequest{
		Room: roomName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}

	participantList := participants.GetParticipants()
	result := make([]*ports.LiveKitParticipant, 0, len(participantList))
	for _, p := range participantList {
		result = append(result, &ports.LiveKitParticipant{
			ID:       p.Sid,
			Identity: p.Identity,
			Name:     p.Name,
		})
	}

	return result, nil
}

func (s *Service) SendData(ctx context.Context, roomName string, data []byte, participantIDs []string) error {
	if roomName == "" {
		return fmt.Errorf("room name is required")
	}

	if len(data) == 0 {
		return fmt.Errorf("data is required")
	}

	destinationIdentities := participantIDs
	if len(participantIDs) == 0 {
		destinationIdentities = nil
	}

	req := &lkproto.SendDataRequest{
		Room:                  roomName,
		Data:                  data,
		Kind:                  lkproto.DataPacket_RELIABLE,
		DestinationIdentities: destinationIdentities,
	}

	_, err := s.roomClient.SendData(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	return nil
}

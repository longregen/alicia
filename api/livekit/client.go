package livekit

import (
	"context"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// Service handles LiveKit room operations.
type Service struct {
	url       string
	apiKey    string
	apiSecret string
	client    *lksdk.RoomServiceClient
}

// NewService creates a new LiveKit service.
func NewService(url, apiKey, apiSecret string) (*Service, error) {
	client := lksdk.NewRoomServiceClient(url, apiKey, apiSecret)
	return &Service{
		url:       url,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client:    client,
	}, nil
}

// Room represents a LiveKit room.
type Room struct {
	Name string `json:"name"`
	SID  string `json:"sid"`
}

// CreateRoom creates a new LiveKit room.
func (s *Service) CreateRoom(ctx context.Context, name string) (*Room, error) {
	room, err := s.client.CreateRoom(ctx, &livekit.CreateRoomRequest{
		Name:            name,
		EmptyTimeout:    300, // 5 minutes
		MaxParticipants: 10,
	})
	if err != nil {
		return nil, err
	}
	return &Room{
		Name: room.Name,
		SID:  room.Sid,
	}, nil
}

// GetRoom gets a room by name.
func (s *Service) GetRoom(ctx context.Context, name string) (*Room, error) {
	rooms, err := s.client.ListRooms(ctx, &livekit.ListRoomsRequest{
		Names: []string{name},
	})
	if err != nil {
		return nil, err
	}
	if len(rooms.Rooms) == 0 {
		return nil, nil
	}
	room := rooms.Rooms[0]
	return &Room{
		Name: room.Name,
		SID:  room.Sid,
	}, nil
}

// DeleteRoom deletes a room.
func (s *Service) DeleteRoom(ctx context.Context, name string) error {
	_, err := s.client.DeleteRoom(ctx, &livekit.DeleteRoomRequest{
		Room: name,
	})
	return err
}

// GenerateToken generates an access token for a participant.
func (s *Service) GenerateToken(roomName, participantID, participantName string) (string, int64, error) {
	at := auth.NewAccessToken(s.apiKey, s.apiSecret)

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

	validity := 6 * time.Hour
	expiresAt := time.Now().Add(validity).Unix()

	at.AddGrant(grant).
		SetIdentity(participantID).
		SetName(participantName).
		SetValidFor(validity)

	token, err := at.ToJWT()
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

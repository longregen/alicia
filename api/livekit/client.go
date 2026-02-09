package livekit

import (
	"context"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

type Service struct {
	url       string
	apiKey    string
	apiSecret string
	client    *lksdk.RoomServiceClient
}

func NewService(url, apiKey, apiSecret string) (*Service, error) {
	client := lksdk.NewRoomServiceClient(url, apiKey, apiSecret)
	return &Service{
		url:       url,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client:    client,
	}, nil
}

type Room struct {
	Name string `json:"name"`
	SID  string `json:"sid"`
}

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

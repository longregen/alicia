package dto

type GenerateTokenRequest struct {
	ParticipantID   string `json:"participant_id"`
	ParticipantName string `json:"participant_name,omitempty"`
}

type GenerateTokenResponse struct {
	Token         string `json:"token"`
	ExpiresAt     int64  `json:"expires_at"`
	RoomName      string `json:"room_name"`
	ParticipantID string `json:"participant_id"`
}

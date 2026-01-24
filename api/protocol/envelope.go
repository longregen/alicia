package protocol

import "github.com/longregen/alicia/shared/protocol"

// Re-export Envelope type and functions
type Envelope = protocol.Envelope

var (
	NewEnvelope    = protocol.NewEnvelope
	DecodeEnvelope = protocol.DecodeEnvelope
)

// DecodeBody decodes the envelope body into the given type.
// This wrapper is needed because Go doesn't support re-exporting generic functions directly.
func DecodeBody[T any](e *Envelope) (*T, error) {
	return protocol.DecodeBody[T](e)
}

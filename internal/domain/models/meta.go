package models

import (
	"time"
)

// Meta represents arbitrary key-value metadata for any entity
type Meta struct {
	ID        string     `json:"id"`
	Ref       string     `json:"ref"`
	Key       string     `json:"key"`
	Value     string     `json:"value"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func NewMeta(id, ref, key, value string) *Meta {
	now := time.Now()
	return &Meta{
		ID:        id,
		Ref:       ref,
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

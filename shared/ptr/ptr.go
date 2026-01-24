// Package ptr provides generic pointer helper functions used across services.
package ptr

// To returns a pointer to the given value.
func To[T any](v T) *T { return &v }

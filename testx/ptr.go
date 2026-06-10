// Package testx provides small helpers shared by service test suites.
package testx

// Ptr returns a pointer to v.
func Ptr[T any](v T) *T { return &v }

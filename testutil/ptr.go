// Package testutil provides small helpers shared by service test suites.
package testutil

// Ptr returns a pointer to v.
func Ptr[T any](v T) *T { return &v }

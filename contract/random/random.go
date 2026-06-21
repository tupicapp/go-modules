// Package random defines the contract for generating random values. The
// crypto/rand-backed implementation lives in adapter/random.
package random

// Random generates random values, such as random strings.
type Random interface {
	String(size int) (string, error)
}

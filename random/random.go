// Package random provides the Random contract plus a crypto/rand-backed
// implementation.
package random

import (
	"crypto/rand"
	"math/big"
	"strings"

	"github.com/cockroachdb/errors"
)

// Random defines an interface for generating random values, such as random
// strings.
type Random interface {
	String(size int) (string, error)
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

type Secure struct{}

func NewSecure() *Secure { return &Secure{} }

func (s *Secure) String(size int) (string, error) {
	if size < 0 {
		return "", errors.New("size must be greater than zero")
	}
	if size == 0 {
		return "", nil
	}
	var builder strings.Builder
	builder.Grow(size)
	for i := 0; i < size; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", errors.WithStack(err)
		}
		builder.WriteByte(charset[idx.Int64()])
	}
	return builder.String(), nil
}

var _ Random = (*Secure)(nil)

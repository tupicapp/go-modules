// Package random provides a crypto/rand-backed implementation of the random.Random contract.
package random

import (
	"crypto/rand"
	"math/big"
	"strings"

	"github.com/cockroachdb/errors"
	contract "github.com/tupicapp/go-modules/contract/random"
)

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

var _ contract.Random = (*Secure)(nil)

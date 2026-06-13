// Package utility provides general-purpose helper functions shared across Tupic services.
package utility

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

// StringOrNil returns nil if value is empty after trimming.
func StringOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

// StringDereference returns the string value of p, or "" if p is nil.
func StringDereference(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ParseUUID parses a UUID string and wraps any parse error with context.
func ParseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "invalid uuid")
	}
	return id, nil
}

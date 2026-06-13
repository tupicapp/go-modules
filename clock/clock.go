// Package clock provides the Clock contract to provide time with testability.
package clock

import "time"

// Clock defines an interface for getting the current time, allowing for easier testing and time manipulation.
type Clock interface {
	Now() time.Time
}

type System struct{}

func NewSystem() *System         { return &System{} }
func (s *System) Now() time.Time { return time.Now() }

type Fixed struct{ T time.Time }

func NewFixed(t time.Time) *Fixed { return &Fixed{T: t} }
func (f *Fixed) Now() time.Time   { return f.T }

var (
	_ Clock = (*System)(nil)
	_ Clock = (*Fixed)(nil)
)

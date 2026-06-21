package system_clock

import (
	"time"

	clock2 "github.com/tupicapp/go-modules/contract/clock"
)

type System struct{}

func NewSystem() *System         { return &System{} }
func (s *System) Now() time.Time { return time.Now() }

var _ clock2.Clock = (*System)(nil)

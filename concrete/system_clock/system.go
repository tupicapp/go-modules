package system_clock

import (
	"time"

	"github.com/tupicapp/go-modules/contract/clock"
)

type System struct{}

func NewSystem() *System         { return &System{} }
func (s *System) Now() time.Time { return time.Now() }

var _ clock.Clock = (*System)(nil)

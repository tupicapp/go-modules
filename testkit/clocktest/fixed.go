package clocktest

import (
	"time"

	clock2 "github.com/tupicapp/go-modules/contract/clock"
)

type Fixed struct{ T time.Time }

func NewFixed(t time.Time) *Fixed { return &Fixed{T: t} }
func (f *Fixed) Now() time.Time   { return f.T }

var _ clock2.Clock = (*Fixed)(nil)

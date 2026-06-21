// Package clock provides the Clock contract to provide time with testability.
package clock

import "time"

// Clock defines an interface for getting the current time, allowing for easier testing and time manipulation.
type Clock interface {
	Now() time.Time
}

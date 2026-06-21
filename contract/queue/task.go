package queue

// Task is the constraint for queueable point-to-point work units.
// A task names itself logically (e.g. "log-created-asset") and declares its schema version.
// The transport channel is an infrastructure concern composed at enqueue time and not tied to task itself.
type Task interface {
	Name() string
	Version() string
}

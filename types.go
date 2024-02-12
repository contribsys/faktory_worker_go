package faktory_worker

import "context"

const (
	Version = "1.7.0"
)

// Perform actually executes the job.
// It must be thread-safe.
type Perform func(ctx context.Context, args ...interface{}) error

type LifecycleEventHandler func(*Manager) error

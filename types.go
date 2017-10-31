package faktory_worker

import (
	"context"
)

const (
        // Version is the current version
	Version = "0.5.0"
)


// Context provides Go's standard Context pattern
// along with a select few additions for job processing.
//
// We're pretty strict about what's exposed in the Context
// because execution should be orthogonal to
// most of the Job payload contents.
type Context interface {
	context.Context

	Jid() string
}


// Perform actually executes the job.
// It must be thread-safe.
type Perform func(ctx Context, args ...interface{}) error

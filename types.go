package faktory_worker

import (
	"context"

	faktory "github.com/contribsys/faktory/client"
)

const (
	Version = "0.7.0"
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
	Bid() string
	JobType() string
}

// Perform actually executes the job.
// It must be thread-safe.
type Perform func(ctx Context, args ...interface{}) error

type Handler func(ctx Context, job *faktory.Job) error

// MiddlewareFunc defines a function to process middleware.
type MiddlewareFunc func(Handler) Handler

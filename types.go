package faktory_worker

import (
	faktory "github.com/contribsys/faktory/client"
)

const (
	Version = "1.1.0"
)

// Perform actually executes the job.
// It must be thread-safe.
type Perform func(ctx Context, args ...interface{}) error

type Handler func(ctx Context, job *faktory.Job) error
type LifecycleEventHandler func(*Manager) error

// MiddlewareFunc defines a function to process middleware.
type MiddlewareFunc func(Handler) Handler

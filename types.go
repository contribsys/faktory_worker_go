package faktory_worker

import (
	"context"
	"time"

	faktory "github.com/contribsys/faktory/client"
)

const (
	Version = "1.1.0"
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
	JobType() string

	// Faktory Enterprise:
	// this method integrates with Faktory Enterprise's Job Tracking feature.
	// `reserveUntil` is optional, only needed for long jobs which have more dynamic
	// lifetimes.
	//
	//     ctx.TrackProgress(10, "Updating code...", nil)
	//     ctx.TrackProgress(20, "Cleaning caches...", &time.Now().Add(1 * time.Hour)))
	//
	TrackProgress(percent int, desc string, reserveUntil *time.Time) error

	// Faktory Enterprise:
	// the BID of the Batch associated with this job
	Bid() string

	// open the batch associated with this job so we can add more jobs to it.
	Batch(func(*faktory.Batch) error) error

	// allows direct access to the Faktory server from the job
	With(func(*faktory.Client) error) error
}

// Perform actually executes the job.
// It must be thread-safe.
type Perform func(ctx Context, args ...interface{}) error

type Handler func(ctx Context, job *faktory.Job) error

// MiddlewareFunc defines a function to process middleware.
type MiddlewareFunc func(Handler) Handler

package faktory_worker

import (
	"context"
	"fmt"
	"time"

	faktory "github.com/contribsys/faktory/client"
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

// DefaultContext embeds Go's standard context and associates it with a job ID.
type defaultContext struct {
	context.Context

	JID  string
	BID  string
	Type string

	mgr *Manager
}

// Jid returns the job ID for the default context
func (c *defaultContext) Jid() string {
	return c.JID
}

func (c *defaultContext) Bid() string {
	return c.BID
}

// JobType returns the job type for the default context
func (c *defaultContext) JobType() string {
	return c.Type
}

// requires Faktory Enterprise
func (c *defaultContext) TrackProgress(percent int, desc string, reserveUntil *time.Time) error {
	return c.mgr.with(func(cl *faktory.Client) error {
		return cl.TrackSet(c.JID, percent, desc, reserveUntil)
	})
}

// Provides a Faktory server connection to the given func
func (c *defaultContext) With(fn func(*faktory.Client) error) error {
	return c.mgr.with(fn)
}

// requires Faktory Enterprise
// Open the current batch so we can add more jobs to it.
func (c *defaultContext) Batch(fn func(*faktory.Batch) error) error {
	if c.BID == "" {
		return fmt.Errorf("No batch associated with this job")
	}

	var b *faktory.Batch
	var err error

	err = c.mgr.with(func(cl *faktory.Client) error {
		b, err = cl.BatchOpen(c.BID)
		if err != nil {
			return err
		}
		return fn(b)
	})
	if err != nil {
		return err
	}

	return nil
}

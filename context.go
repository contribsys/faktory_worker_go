package faktory_worker

import (
	"context"
	"fmt"
	"log"
	"time"

	faktory "github.com/contribsys/faktory/client"
)

// internal keys for context value storage
type valueKey int

const (
	helperKey valueKey = 1
)

var (
	NoAssociatedBatchError = fmt.Errorf("No batch associated with this job")
)

// The Helper provides access to valuable data and APIs
// within an executing job.
//
// We're pretty strict about what's exposed in the Helper
// because execution should be orthogonal to
// most of the Job payload contents.
//
//   func myJob(ctx context.Context, args ...interface{}) error {
//     helper := worker.HelperFor(ctx)
//     jid := helper.Jid()
//
//     helper.With(func(cl *faktory.Client) error {
//       cl.Push("anotherJob", 4, "arg")
//		 })
//
type Helper interface {
	Jid() string
	JobType() string

	// Faktory Enterprise:
	// the BID of the Batch associated with this job
	Bid() string

	// Faktory Enterprise:
	// open the batch associated with this job so we can add more jobs to it.
	//
	//   func myJob(ctx context.Context, args ...interface{}) error {
	//     helper := worker.HelperFor(ctx)
	//     helper.Batch(func(b *faktory.Batch) error {
	//       return b.Push(faktory.NewJob("sometype", 1, 2, 3))
	//     })
	Batch(func(*faktory.Batch) error) error

	// allows direct access to the Faktory server from the job
	With(func(*faktory.Client) error) error

	// Push the given job to Faktory. In test mode, this will
	// save the job in an array so you can assert it.
	Push(*faktory.Job) error

	// Faktory Enterprise:
	// this method integrates with Faktory Enterprise's Job Tracking feature.
	// `reserveUntil` is optional, only needed for long jobs which have more dynamic
	// lifetimes.
	//
	//     helper.TrackProgress(10, "Updating code...", nil)
	//     helper.TrackProgress(20, "Cleaning caches...", &time.Now().Add(1 * time.Hour)))
	//
	TrackProgress(percent int, desc string, reserveUntil *time.Time) error
}

// The test helper is a mode of operation which tries to avoid network calls.
// This allows you to unit test your job code with the Helper API without
// Faktory running.
type testHelper struct {
	job        *faktory.Job
	pushedJobs []*faktory.Job
}

func (h *testHelper) Jid() string {
	return h.job.Jid
}
func (h *testHelper) Bid() string {
	if b, ok := h.job.GetCustom("_bid"); ok {
		return b.(string)
	}
	return ""
}
func (h *testHelper) JobType() string {
	return h.job.Type
}

func (h *testHelper) TrackProgress(percent int, desc string, reserveUntil *time.Time) error {
	return nil
}
func (h *testHelper) Batch(fn func(*faktory.Batch) error) error {
	return nil
}
func (h *testHelper) With(fn func(*faktory.Client) error) error {
	return fmt.Errorf("With() does not work in test mode, you will need to abstract out this call")
}

func (h *testHelper) Push(job *faktory.Job) error {
	h.pushedJobs = append(h.pushedJobs, job)
	return nil
}

func (h *testHelper) PushedJobs() []*faktory.Job {
	return h.pushedJobs
}

// This "live" helper impl requires Faktory to be running.
type jobHelper struct {
	job  *faktory.Job
	pool *faktory.Pool
}

func (h *jobHelper) Jid() string {
	return h.job.Jid
}
func (h *jobHelper) Bid() string {
	if b, ok := h.job.GetCustom("_bid"); ok {
		return b.(string)
	}
	return ""
}
func (h *jobHelper) JobType() string {
	return h.job.Type
}

// Caution: this method must only be called within the
// context of an executing job. It will panic if it cannot
// create a Helper due to missing context values.
func HelperFor(ctx context.Context) Helper {
	if j := ctx.Value(helperKey); j != nil {
		helper := j.(Helper)
		return helper
	}
	log.Panic("Invalid job context, cannot create faktory_worker_go job helper")
	return nil
}

func jobContext(testing bool, pool *faktory.Pool, job *faktory.Job) context.Context {
	var helper Helper
	if testing {
		helper = &testHelper{
			job: job,
		}
	} else {
		helper = &jobHelper{
			job:  job,
			pool: pool,
		}
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, helperKey, helper)
	return ctx
}

// requires Faktory Enterprise
func (h *jobHelper) TrackProgress(percent int, desc string, reserveUntil *time.Time) error {
	return h.With(func(cl *faktory.Client) error {
		return cl.TrackSet(h.Jid(), percent, desc, reserveUntil)
	})
}

// requires Faktory Enterprise
// Open the current batch so we can add more jobs to it.
func (h *jobHelper) Batch(fn func(*faktory.Batch) error) error {
	bid := h.Bid()
	if bid == "" {
		return NoAssociatedBatchError
	}

	var b *faktory.Batch
	var err error

	err = h.pool.With(func(cl *faktory.Client) error {
		b, err = cl.BatchOpen(bid)
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

func (h *jobHelper) With(fn func(*faktory.Client) error) error {
	return h.pool.With(fn)
}

func (h *jobHelper) Push(job *faktory.Job) error {
	return h.pool.With(func(cl *faktory.Client) error {
		return cl.Push(job)
	})
}

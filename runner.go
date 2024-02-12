package faktory_worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	faktory "github.com/contribsys/faktory/client"
)

type lifecycleEventType int

const (
	Startup  lifecycleEventType = 1
	Quiet    lifecycleEventType = 2
	Shutdown lifecycleEventType = 3
)

type NoHandlerError struct {
	JobType string
}

func (s *NoHandlerError) Error() string {
	return fmt.Sprintf("No handler registered for job type %s", s.JobType)
}

func heartbeat(mgr *Manager) {
	mgr.shutdownWaiter.Add(1)
	defer mgr.shutdownWaiter.Done()

	timer := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-timer.C:
			// we don't care about errors, assume any network
			// errors will heal eventually
			err := mgr.with(func(c *faktory.Client) error {
				data, err := c.Beat(mgr.state)
				if err != nil && strings.Contains(err.Error(), "Unknown worker") {
					// If our heartbeat expires, we must restart and re-authenticate.
					// Use a signal so we can unwind and shutdown cleanly.
					mgr.Logger.Warn("Faktory heartbeat has expired, shutting down...")
					if process, err := os.FindProcess(os.Getpid()); err != nil {
						mgr.Logger.Errorf("Could not find worker process %d: %v", os.Getpid(), err)
					} else {
						_ = process.Signal(syscall.SIGTERM)
					}
				}
				if err != nil || data == "" {
					return err
				}
				var hash map[string]string
				err = json.Unmarshal([]byte(data), &hash)
				if err != nil {
					return err
				}

				if state, ok := hash["state"]; ok && state != "" {
					mgr.handleEvent(state)
				}
				return nil
			})
			if err != nil {
				mgr.Logger.Error(fmt.Sprintf("heartbeat error: %v", err))
			}
		case <-mgr.done:
			timer.Stop()
			return
		}
	}
}

func process(ctx context.Context, mgr *Manager, idx int) {
	mgr.shutdownWaiter.Add(1)
	defer mgr.shutdownWaiter.Done()

	// delay initial fetch randomly to prevent thundering herd.
	// this will pause between 0 and 2B nanoseconds, i.e. 0-2 seconds
	time.Sleep(time.Duration(rand.Int31()))
	sleep := 1.0
	for {
		if mgr.state != "" {
			return
		}

		// check for shutdown
		select {
		case <-mgr.done:
			return
		default:
		}

		err := processOne(ctx, mgr)
		if err != nil {
			mgr.Logger.Debug(err)
			if _, ok := err.(*NoHandlerError); !ok {
				// if we don't know how to process this jobtype,
				// we Fail it and sleep for a bit so we don't get
				// caught in an infinite loop "processing" a queue full
				// of jobs we don't understand.
				time.Sleep(50 * time.Millisecond)
			} else {
				// if we have an unknown error processing a job, use
				// exponential backoff so we don't constantly slam the
				// log with "connection refused" errors or similar.
				select {
				case <-mgr.done:
				case <-time.After(time.Duration(sleep) * time.Second):
					sleep = math.Max(sleep*2, 30)
				}
			}
		} else {
			// success, reset sleep timer
			sleep = 1.0
		}
	}
}

func processOne(ctx context.Context, mgr *Manager) error {
	var job *faktory.Job

	// explicit scopes to limit variable visibility
	{
		var e error
		err := mgr.with(func(c *faktory.Client) error {
			job, e = c.Fetch(mgr.queueList()...)
			if e != nil {
				return e
			}
			return nil
		})
		if err != nil {
			return err
		}
		if job == nil {
			return nil
		}
	}

	if !mgr.isRegistered(job.Type) {
		je := &NoHandlerError{JobType: job.Type}
		err := mgr.with(func(c *faktory.Client) error {
			return c.Fail(job.Jid, je, nil)
		})
		if err != nil {
			return err
		}
		return je
	}

	joberr := mgr.dispatch(ctx, job)
	if joberr != nil {
		// job errors are normal and expected, we don't return early from them
		mgr.Logger.Errorf("Error running %s job %s: %v", job.Type, job.Jid, joberr)
	}

	until := time.After(30 * time.Second)
	sleep := 1.0
	for {
		// we want to report the result back to Faktory.
		// we stay in this loop until we successfully report.
		err := mgr.with(func(c *faktory.Client) error {
			if joberr != nil {
				return c.Fail(job.Jid, joberr, nil)
			} else {
				return c.Ack(job.Jid)
			}
		})
		if err == nil {
			return nil
		}
		select {
		case <-until:
			mgr.Logger.Error(fmt.Errorf("Failed to report JID %v result to Faktory: %w", job.Jid, err))
			return nil
		case <-mgr.done:
			mgr.Logger.Error(fmt.Errorf("Unable to report JID %v result to Faktory: %w", job.Jid, err))
			return nil
		case <-time.After(time.Duration(sleep) * time.Second):
			sleep = math.Max(sleep*2, 30)
			mgr.Logger.Debug(fmt.Errorf("Unable to report JID %v result to Faktory: %w", job.Jid, err))
		}
	}
}

// expandWeightedQueues builds a slice of queues represented the number of times equal to their weights.
func expandWeightedQueues(queueWeights map[string]int) []string {
	weightsTotal := 0
	for _, queueWeight := range queueWeights {
		weightsTotal += queueWeight
	}

	weightedQueues := make([]string, weightsTotal)
	fillIndex := 0

	for queue, nTimes := range queueWeights {
		// Fill weightedQueues with queue n times
		for idx := 0; idx < nTimes; idx++ {
			weightedQueues[fillIndex] = queue
			fillIndex++
		}
	}

	// weightedQueues has to be stable so we can write tests
	sort.Strings(weightedQueues)
	return weightedQueues
}

func queueKeys(queues map[string]int) []string {
	keys := make([]string, len(queues))
	i := 0
	for k := range queues {
		keys[i] = k
		i++
	}
	// queues has to be stable so we can write tests
	sort.Strings(keys)
	return keys
}

// shuffleQueues returns a copy of the slice with the elements shuffled.
func shuffleQueues(queues []string) []string {
	wq := make([]string, len(queues))
	copy(wq, queues)

	rand.Shuffle(len(wq), func(i, j int) {
		wq[i], wq[j] = wq[j], wq[i]
	})

	return wq
}

// uniqQueues returns a slice of length len, of the unique elements while maintaining order.
// The underlying array is modified to avoid allocating another one.
func uniqQueues(length int, queues []string) []string {
	// Record the unique values and position.
	pos := 0
	uniqMap := make(map[string]int)
	for idx := range queues {
		if _, ok := uniqMap[queues[idx]]; !ok {
			uniqMap[queues[idx]] = pos
			pos++
		}
	}

	// Reuse the copied array, by updating the values.
	for queue, position := range uniqMap {
		queues[position] = queue
	}

	// Slice only what we need.
	return queues[:length]
}

func dumpThreads(logg Logger) {
	buf := make([]byte, 64*1024)
	_ = runtime.Stack(buf, true)
	logg.Info("FULL PROCESS THREAD DUMP:")
	logg.Info(string(buf))
}

package faktory_worker

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"sort"
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
	timer := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-timer.C:
			// we don't care about errors, assume any network
			// errors will heal eventually
			err := mgr.with(func(c *faktory.Client) error {
				data, err := c.Beat(mgr.state)
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
			mgr.shutdownWaiter.Done()
			return
		}
	}
}

func process(mgr *Manager, idx int) {
	mgr.shutdownWaiter.Add(1)
	// delay initial fetch randomly to prevent thundering herd.
	// this will pause between 0 and 2B nanoseconds, i.e. 0-2 seconds
	time.Sleep(time.Duration(rand.Int31()))
	defer mgr.shutdownWaiter.Done()

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

		err := processOne(mgr)
		if err != nil {
			mgr.Logger.Error(err)
			if _, ok := err.(*NoHandlerError); !ok {
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func processOne(mgr *Manager) error {
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

	perform := mgr.jobHandlers[job.Type]

	if perform == nil {
		je := &NoHandlerError{JobType: job.Type}
		err := mgr.with(func(c *faktory.Client) error {
			return c.Fail(job.Jid, je, nil)
		})
		if err != nil {
			return err
		}
		return je
	}

	joberr := dispatch(mgr.middleware, jobContext(false, mgr.Pool, job), job, perform)
	if joberr != nil {
		// job errors are normal and expected, we don't return early from them
		mgr.Logger.Errorf("Error running %s job %s: %v", job.Type, job.Jid, joberr)
	}

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
		mgr.Logger.Error(err)
		time.Sleep(1 * time.Second)
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
func uniqQueues(len int, queues []string) []string {
	// Record the unique values and position.
	pos := 0
	uniqMap := make(map[string]int)
	for _, v := range queues {
		if _, ok := uniqMap[v]; !ok {
			uniqMap[v] = pos
			pos++
		}
	}

	// Reuse the copied array, by updating the values.
	for queue, position := range uniqMap {
		queues[position] = queue
	}

	// Slice only what we need.
	return queues[:len]
}

func dumpThreads(logg Logger) {
	buf := make([]byte, 64*1024)
	_ = runtime.Stack(buf, true)
	logg.Info("FULL PROCESS THREAD DUMP:")
	logg.Info(string(buf))
}

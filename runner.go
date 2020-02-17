package faktory_worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	faktory "github.com/contribsys/faktory/client"
)

type eventType int

const (
	Startup  eventType = 1
	Quiet    eventType = 2
	Shutdown eventType = 3
)

// Register registers a handler for the given jobtype.  It is expected that all jobtypes
// are registered upon process startup.
//
// faktory_worker.Register("ImportantJob", ImportantFunc)
func (mgr *Manager) Register(name string, fn Perform) {
	mgr.jobHandlers[name] = func(ctx Context, job *faktory.Job) error {
		return fn(ctx, job.Args...)
	}
}

// Manager coordinates the processes for the worker.  It is responsible for
// starting and stopping goroutines to perform work at the desired concurrency level
type Manager struct {
	Concurrency int
	Logger      Logger
	ProcessWID  string
	Labels      []string
	Pool        *faktory.Pool

	queues     []string
	middleware []MiddlewareFunc
	state      string // "", "quiet" or "terminate"
	// The done channel will always block unless
	// the system is shutting down.
	done           chan interface{}
	shutdownWaiter *sync.WaitGroup
	jobHandlers    map[string]Handler
	eventHandlers  map[eventType][]func()

	// This only needs to be computed once. Store it here to keep things fast.
	weightedPriorityQueuesEnabled bool
	weightedQueues                []string
}

// Register a callback to be fired when a process lifecycle event occurs.
// These are useful for hooking into process startup or shutdown.
func (mgr *Manager) On(event eventType, fn func()) {
	mgr.eventHandlers[event] = append(mgr.eventHandlers[event], fn)
}

// After calling Quiet(), no more jobs will be pulled
// from Faktory by this process.
func (mgr *Manager) Quiet() {
	mgr.Logger.Info("Quieting...")
	mgr.state = "quiet"
	mgr.fireEvent(Quiet)
}

// Terminate signals that the various components should shutdown.
// Blocks on the shutdownWaiter until all components have finished.
func (mgr *Manager) Terminate() {
	mgr.Logger.Info("Shutting down...")
	mgr.state = "terminate"
	close(mgr.done)
	mgr.fireEvent(Shutdown)
	mgr.shutdownWaiter.Wait()
	mgr.Pool.Close()
	mgr.Logger.Info("Goodbye")
	os.Exit(0)
}

// Use adds middleware to the chain.
func (mgr *Manager) Use(middleware ...MiddlewareFunc) {
	mgr.middleware = append(mgr.middleware, middleware...)
}

// NewManager returns a new manager with default values.
func NewManager() *Manager {
	return &Manager{
		Concurrency: 20,
		Logger:      NewStdLogger(),
		Labels:      []string{"golang"},
		Pool:        nil,

		state:          "",
		queues:         []string{"default"},
		done:           make(chan interface{}),
		shutdownWaiter: &sync.WaitGroup{},
		jobHandlers:    map[string]Handler{},
		eventHandlers: map[eventType][]func(){
			Startup:  []func(){},
			Quiet:    []func(){},
			Shutdown: []func(){},
		},
		weightedPriorityQueuesEnabled: false,
		weightedQueues:                []string{},
	}
}

// Run starts processing jobs.
// This method does not return.
func (mgr *Manager) Run() {
	// This will signal to Faktory that all connections from this process
	// are worker connections.
	if len(mgr.ProcessWID) == 0 {
		rand.Seed(time.Now().UnixNano())
		faktory.RandomProcessWid = strconv.FormatInt(rand.Int63(), 32)
	} else {
		faktory.RandomProcessWid = mgr.ProcessWID
	}

	// Set labels to be displayed in the UI
	faktory.Labels = mgr.Labels

	if mgr.Pool == nil {
		pool, err := faktory.NewPool(mgr.Concurrency)
		if err != nil {
			log.Panicf("Couldn't create Faktory connection pool: %v", err)
		}
		mgr.Pool = pool
	}

	mgr.fireEvent(Startup)

	go heartbeat(mgr)

	mgr.Logger.Infof("faktory_worker_go PID %d now ready to process jobs", os.Getpid())
	for i := 0; i < mgr.Concurrency; i++ {
		go process(mgr, i)
	}

	sigchan := hookSignals()

	for {
		sig := <-sigchan
		handleEvent(signalMap[sig], mgr)
	}
}

func (mgr *Manager) with(fn func(cl *faktory.Client) error) error {
	if mgr.Pool == nil {
		panic("No Pool set on Manager, have you called manager.Run() yet?")
	}
	return mgr.Pool.With(fn)
}

// One of the Process*Queues methods should be called once before Run()
func (mgr *Manager) ProcessStrictPriorityQueues(queues ...string) {
	mgr.queues = queues
	mgr.weightedPriorityQueuesEnabled = false
}

func (mgr *Manager) ProcessWeightedPriorityQueues(queues map[string]int) {
	uniqueQueues := queueKeys(queues)
	weightedQueues := expandWeightedQueues(queues)

	mgr.queues = uniqueQueues
	mgr.weightedQueues = weightedQueues
	mgr.weightedPriorityQueuesEnabled = true
}

func (mgr *Manager) queueList() []string {
	if mgr.weightedPriorityQueuesEnabled {
		sq := shuffleQueues(mgr.weightedQueues)
		return uniqQueues(len(mgr.queues), sq)
	}
	return mgr.queues
}

func (mgr *Manager) CurrentState() string {
	return mgr.state
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
				//data, err := c.Beat(mgr.CurrentState())
				data, err := c.Beat()
				if err != nil || data == "" {
					return err
				}
				var hash map[string]string
				err = json.Unmarshal([]byte(data), &hash)
				if err != nil {
					return err
				}

				if state, ok := hash["state"]; ok && state != "" {
					handleEvent(state, mgr)
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

func handleEvent(sig string, mgr *Manager) {
	if sig == mgr.CurrentState() {
		return
	}
	if sig == "quiet" && mgr.CurrentState() == "terminate" {
		// this is a no-op, a terminating process is quiet already
		return
	}

	switch sig {
	case "terminate":
		go func() {
			mgr.Terminate()
		}()
	case "quiet":
		go func() {
			mgr.Quiet()
		}()
	case "dump":
		dumpThreads(mgr.Logger)
	}
}

func process(mgr *Manager, idx int) {
	mgr.shutdownWaiter.Add(1)
	// delay initial fetch randomly to prevent thundering herd.
	// this will pause between 0 and 2B nanoseconds, i.e. 0-2 seconds
	time.Sleep(time.Duration(rand.Int31()))
	defer mgr.shutdownWaiter.Done()

	for {
		if mgr.CurrentState() != "" {
			return
		}

		// check for shutdown
		select {
		case <-mgr.done:
			return
		default:
		}

		// fetch job
		var job *faktory.Job
		var err error

		err = mgr.with(func(c *faktory.Client) error {
			job, err = c.Fetch(mgr.queueList()...)
			if err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			mgr.Logger.Error(err)
			time.Sleep(1 * time.Second)
			continue
		}

		// execute
		if job != nil {
			perform := mgr.jobHandlers[job.Type]
			if perform == nil {
				mgr.with(func(c *faktory.Client) error {
					return c.Fail(job.Jid, fmt.Errorf("No handler for %s", job.Type), nil)
				})
			} else {
				h := perform
				for i := len(mgr.middleware) - 1; i >= 0; i-- {
					h = mgr.middleware[i](h)
				}

				err := h(ctxFor(mgr, job), job)
				mgr.with(func(c *faktory.Client) error {
					if err != nil {
						return c.Fail(job.Jid, err, nil)
					} else {
						return c.Ack(job.Jid)
					}
				})
			}
		} else {
			// if there are no jobs, Faktory will block us on
			// the first queue, so no need to poll or sleep
		}
	}
}

func (mgr *Manager) fireEvent(event eventType) {
	for _, fn := range mgr.eventHandlers[event] {
		fn()
	}
}

// DefaultContext embeds Go's standard context and associates it with a job ID.
type DefaultContext struct {
	context.Context

	JID  string
	BID  string
	Type string
	mgr  *Manager
}

// Jid returns the job ID for the default context
func (c *DefaultContext) Jid() string {
	return c.JID
}

func (c *DefaultContext) Bid() string {
	return c.BID
}

// JobType returns the job type for the default context
func (c *DefaultContext) JobType() string {
	return c.Type
}

// requires Faktory Enterprise
func (c *DefaultContext) JobProgress(percent int, desc string) error {
	return c.mgr.with(func(cl *faktory.Client) error {
		return cl.TrackSet(c.JID, percent, desc, nil)
	})
}

// requires Faktory Enterprise
// Open the current batch so we can add more jobs to it.
func (c *DefaultContext) Batch(fn func(*faktory.Batch) error) error {
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

func ctxFor(m *Manager, job *faktory.Job) Context {
	c := &DefaultContext{
		Context: context.Background(),
		JID:     job.Jid,
		Type:    job.Type,
		mgr:     m,
	}
	s, _ := job.GetCustom("bid")
	if s != nil {
		c.BID = s.(string)
	}
	return c
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

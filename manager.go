package faktory_worker

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	faktory "github.com/contribsys/faktory/client"
)

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
	eventHandlers  map[lifecycleEventType][]LifecycleEventHandler

	// This only needs to be computed once. Store it here to keep things fast.
	weightedPriorityQueuesEnabled bool
	weightedQueues                []string
}

// Register a handler for the given jobtype.  It is expected that all jobtypes
// are registered upon process startup.
//
//    mgr.Register("ImportantJob", ImportantFunc)
func (mgr *Manager) Register(name string, fn Perform) {
	mgr.jobHandlers[name] = func(ctx context.Context, job *faktory.Job) error {
		return fn(ctx, job.Args...)
	}
}

// Register a callback to be fired when a process lifecycle event occurs.
// These are useful for hooking into process startup or shutdown.
func (mgr *Manager) On(event lifecycleEventType, fn LifecycleEventHandler) {
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
func (mgr *Manager) Terminate(reallydie bool) {
	mgr.Logger.Info("Shutting down...")
	mgr.state = "terminate"
	close(mgr.done)
	mgr.fireEvent(Shutdown)
	mgr.shutdownWaiter.Wait()
	mgr.Pool.Close()
	mgr.Logger.Info("Goodbye")
	if reallydie {
		os.Exit(0)
	}
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
		eventHandlers: map[lifecycleEventType][]LifecycleEventHandler{
			Startup:  []LifecycleEventHandler{},
			Quiet:    []LifecycleEventHandler{},
			Shutdown: []LifecycleEventHandler{},
		},
		weightedPriorityQueuesEnabled: false,
		weightedQueues:                []string{},
	}
}

func (mgr *Manager) setUpWorkerProcess() {
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
		pool, err := faktory.NewPool(mgr.Concurrency + 2)
		if err != nil {
			log.Panicf("Couldn't create Faktory connection pool: %v", err)
		}
		mgr.Pool = pool
	}
}

// Run starts processing jobs.
// This method does not return.
func (mgr *Manager) Run() {
	mgr.setUpWorkerProcess()
	mgr.fireEvent(Startup)

	go heartbeat(mgr)

	mgr.Logger.Infof("faktory_worker_go %s PID %d now ready to process jobs", Version, os.Getpid())
	for i := 0; i < mgr.Concurrency; i++ {
		go process(mgr, i)
	}

	sigchan := hookSignals()

	for {
		sig := <-sigchan
		mgr.handleEvent(signalMap[sig])
	}
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

func (mgr *Manager) fireEvent(event lifecycleEventType) {
	for _, fn := range mgr.eventHandlers[event] {
		err := fn(mgr)
		if err != nil {
			mgr.Logger.Errorf("Error running lifecycle event handler: %v", err)
		}
	}
}

func (mgr *Manager) with(fn func(cl *faktory.Client) error) error {
	if mgr.Pool == nil {
		panic("No Pool set on Manager, have you called manager.Run() yet?")
	}
	return mgr.Pool.With(fn)
}

func (mgr *Manager) handleEvent(sig string) string {
	if sig == mgr.state {
		return mgr.state
	}
	if sig == "quiet" && mgr.state == "terminate" {
		// this is a no-op, a terminating process is quiet already
		return mgr.state
	}

	switch sig {
	case "terminate":
		go func() {
			mgr.Terminate(true)
		}()
	case "quiet":
		go func() {
			mgr.Quiet()
		}()
	case "dump":
		dumpThreads(mgr.Logger)
	}

	return ""
}

package faktory_worker

import (
	"context"
	"fmt"
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
	mut sync.Mutex

	Concurrency     int
	Logger          Logger
	ProcessWID      string
	Labels          []string
	Pool            *faktory.Pool
	ShutdownTimeout time.Duration

	queues     []string
	middleware []MiddlewareFunc
	state      string // "", "quiet" or "terminate"
	// The done channel will always block unless
	// the system is shutting down.
	done           chan interface{}
	shutdownWaiter *sync.WaitGroup
	jobHandlers    map[string]Handler
	eventHandlers  map[lifecycleEventType][]LifecycleEventHandler
	cancelFunc     context.CancelFunc

	// This only needs to be computed once. Store it here to keep things fast.
	weightedPriorityQueuesEnabled bool
	weightedQueues                []string
}

// Register a handler for the given jobtype.  It is expected that all jobtypes
// are registered upon process startup.
//
//	mgr.Register("ImportantJob", ImportantFunc)
func (mgr *Manager) Register(name string, fn Perform) {
	mgr.jobHandlers[name] = func(ctx context.Context, job *faktory.Job) error {
		return fn(ctx, job.Args...)
	}
}

// isRegistered checks if a given job name is registered with the manager.
//
//	mgr.isRegistered("SomeJob")
func (mgr *Manager) isRegistered(name string) bool {
	_, ok := mgr.jobHandlers[name]

	return ok
}

// dispatch immediately executes a job, including all middleware.
func (mgr *Manager) dispatch(ctx context.Context, job *faktory.Job) error {
	perform := mgr.jobHandlers[job.Type]

	return dispatch(jobContext(ctx, mgr.Pool, job), mgr.middleware, job, perform)
}

// InlineDispatch is designed for testing. It immediate executes a job, including all middleware,
// as well as performs manager setup if needed.
func (mgr *Manager) InlineDispatch(job *faktory.Job) error {
	if !mgr.isRegistered(job.Type) {
		return fmt.Errorf("failed to dispatch inline for job type %s; job not registered", job.Type)
	}

	err := mgr.setUpWorkerProcess()
	if err != nil {
		return fmt.Errorf("couldn't set up worker process for inline dispatch - %w", err)
	}

	err = mgr.dispatch(context.Background(), job)
	if err != nil {
		return fmt.Errorf("job was dispatched inline but failed. Job type %s, with args %+v - %w", job.Type, job.Args, err)
	}

	return nil
}

// Register a callback to be fired when a process lifecycle event occurs.
// These are useful for hooking into process startup or shutdown.
func (mgr *Manager) On(event lifecycleEventType, fn LifecycleEventHandler) {
	mgr.eventHandlers[event] = append(mgr.eventHandlers[event], fn)
}

// After calling Quiet(), no more jobs will be pulled
// from Faktory by this process.
func (mgr *Manager) Quiet() {
	mgr.mut.Lock()
	defer mgr.mut.Unlock()

	if mgr.state == "quiet" {
		return
	}

	mgr.Logger.Info("Quieting...")
	mgr.state = "quiet"
	mgr.fireEvent(Quiet)
}

// Terminate signals that the various components should shutdown.
// Blocks on the shutdownWaiter until all components have finished.
func (mgr *Manager) Terminate(reallydie bool) {
	mgr.mut.Lock()
	defer mgr.mut.Unlock()

	if mgr.state == "terminate" {
		return
	}

	mgr.Logger.Info("Shutting down...")
	mgr.state = "terminate"
	close(mgr.done)

	if mgr.cancelFunc != nil {
		// cancel any jobs which are lingering
		time.AfterFunc(mgr.ShutdownTimeout, mgr.cancelFunc)
	}
	mgr.fireEvent(Shutdown)
	mgr.shutdownWaiter.Wait() // can't pass this point until all jobs are done

	mgr.Pool.Close()
	mgr.Logger.Info("Goodbye")
	if reallydie {
		os.Exit(0) // nolint:gocritic
	}
}

// NewManager returns a new manager with default values.
func NewManager() *Manager {
	return &Manager{
		Concurrency: 20,
		Logger:      NewStdLogger(),
		Labels:      []string{"golang-" + Version},
		Pool:        nil,

		// best practice is to give jobs 25 seconds to finish their work
		// and then use the last 5 seconds to force any lingering jobs to
		// stop by closing their Context. Many cloud services default to a
		// hard 30 second timeout beforing KILLing the process.
		ShutdownTimeout: 25 * time.Second,

		state:          "",
		queues:         []string{"default"},
		done:           make(chan interface{}),
		shutdownWaiter: &sync.WaitGroup{},
		jobHandlers:    map[string]Handler{},
		eventHandlers: map[lifecycleEventType][]LifecycleEventHandler{
			Startup:  {},
			Quiet:    {},
			Shutdown: {},
		},
		weightedPriorityQueuesEnabled: false,
		weightedQueues:                []string{},
	}
}

func (mgr *Manager) setUpWorkerProcess() error {
	mgr.mut.Lock()
	defer mgr.mut.Unlock()

	if mgr.state != "" {
		return fmt.Errorf("cannot start worker process for the mananger in %v state", mgr.state)
	}

	// This will signal to Faktory that all connections from this process
	// are worker connections.
	if len(mgr.ProcessWID) == 0 {
		faktory.RandomProcessWid = strconv.FormatInt(rand.Int63(), 32)
	} else {
		faktory.RandomProcessWid = mgr.ProcessWID
	}
	// Set labels to be displayed in the UI
	faktory.Labels = mgr.Labels

	if mgr.Pool == nil {
		pool, err := faktory.NewPool(mgr.Concurrency + 2)
		if err != nil {
			return fmt.Errorf("couldn't create Faktory connection pool: %w", err)
		}
		mgr.Pool = pool
	}

	return nil
}

// RunWithContext starts processing jobs. The method will return if an error is encountered while starting.
// If the context is present then os signals will be ignored, the context must be canceled for the method to return
// after running.
func (mgr *Manager) RunWithContext(ctx context.Context) error {
	err := mgr.boot(ctx)
	if err != nil {
		return err
	}
	<-ctx.Done()
	mgr.Terminate(false)
	return nil
}

func (mgr *Manager) boot(ctx context.Context) error {
	err := mgr.setUpWorkerProcess()
	if err != nil {
		return err
	}

	mgr.fireEvent(Startup)
	go heartbeat(mgr)

	mgr.Logger.Infof("faktory_worker_go %s PID %d now ready to process jobs", Version, os.Getpid())
	mgr.Logger.Debugf("Using Faktory Client API %s", faktory.Version)
	for i := 0; i < mgr.Concurrency; i++ {
		go process(ctx, mgr, i)
	}

	return nil
}

// Run starts processing jobs.
// This method does not return unless an error is encountered while starting.
func (mgr *Manager) Run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	mgr.cancelFunc = cancel
	err := mgr.boot(ctx)
	if err != nil {
		return err
	}
	for {
		sig := <-hookSignals()
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

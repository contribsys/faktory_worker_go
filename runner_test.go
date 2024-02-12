package faktory_worker

import (
	"context"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

func sometask(ctx context.Context, args ...interface{}) error {
	return nil
}

func TestRegistration(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	mgr.Register("somejob", sometask)
}

func TestWeightedQueues(t *testing.T) {
	rand.Seed(42)

	mgr := NewManager()
	mgr.ProcessWeightedPriorityQueues(map[string]int{"critical": 3, "default": 2, "bulk": 1})
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.queueList())
	assert.Equal(t, []string{"bulk", "default", "critical"}, mgr.queueList())

	mgr.ProcessWeightedPriorityQueues(map[string]int{"critical": 1, "default": 100, "bulk": 1000})
	assert.Equal(t, []string{"bulk", "default", "critical"}, mgr.queueList())

	mgr.ProcessWeightedPriorityQueues(map[string]int{"critical": 1, "default": 1000, "bulk": 100})
	assert.Equal(t, []string{"default", "bulk", "critical"}, mgr.queueList())

	mgr.ProcessWeightedPriorityQueues(map[string]int{"critical": 1, "default": 1, "bulk": 1})
	assert.Equal(t, []string{"critical", "bulk", "default"}, mgr.queueList())
}

func TestStrictQueues(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	mgr.ProcessStrictPriorityQueues("critical", "default", "bulk")
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.queueList())
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.queueList())
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.queueList())
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.queueList())

	mgr.ProcessStrictPriorityQueues("default", "critical", "bulk")
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.queueList())
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.queueList())
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.queueList())
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.queueList())
}

func TestLiveServer(t *testing.T) {
	mgr := NewManager()
	mgr.ProcessStrictPriorityQueues("fwgtest")
	mgr.Concurrency = 1
	err := mgr.setUpWorkerProcess()
	assert.NoError(t, err)

	mgr.Register("aworker", func(ctx context.Context, args ...interface{}) error {
		//fmt.Println("doing work", args)
		return nil
	})

	withServer(t, "oss", mgr, func(cl *faktory.Client) error {
		cl.Flush()

		j := faktory.NewJob("something", 1, 2)
		j.Queue = "fwgtest"
		err := cl.Push(j)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = processOne(ctx, mgr)
		assert.Error(t, err)
		_, ok := err.(*NoHandlerError)
		assert.True(t, ok)
		assert.Equal(t, err, &NoHandlerError{JobType: "something"})

		j = faktory.NewJob("aworker", 1, 2)
		j.Queue = "fwgtest"
		err = cl.Push(j)
		assert.NoError(t, err)

		err = processOne(ctx, mgr)
		assert.NoError(t, err)
		return nil
	})

}

func TestThreadDump(t *testing.T) {
	t.Parallel()

	devnull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	assert.NoError(t, err)

	logg := &StdLogger{
		log.New(devnull, "", 0),
	}
	dumpThreads(logg)
}

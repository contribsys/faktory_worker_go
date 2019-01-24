package faktory_worker

import (
	"math/rand"
	"testing"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

func sometask(ctx Context, args ...interface{}) error {
	return nil
}

func TestRegistration(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	mgr.Register("somejob", sometask)
}

func TestContext(t *testing.T) {
	t.Parallel()
	job := faktory.NewJob("something", 1, 2)
	ctx := ctxFor(job)
	assert.Equal(t, ctx.Jid(), job.Jid)
	_, ok := ctx.Deadline()
	assert.False(t, ok)
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

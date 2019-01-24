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
	t.Parallel()
	rand.Seed(42)
	mgr := NewManager()
	mgr.WeightedPriorityQueues(map[string]int{"critical": 3, "default": 2, "bulk": 1})
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.getQueues())
	assert.Equal(t, []string{"bulk", "default", "critical"}, mgr.getQueues())

	mgr.WeightedPriorityQueues(map[string]int{"critical": 1, "default": 100, "bulk": 1000})
	assert.Equal(t, []string{"bulk", "default", "critical"}, mgr.getQueues())

	mgr.WeightedPriorityQueues(map[string]int{"critical": 1, "default": 1000, "bulk": 100})
	assert.Equal(t, []string{"default", "bulk", "critical"}, mgr.getQueues())

	mgr.WeightedPriorityQueues(map[string]int{"critical": 1, "default": 1, "bulk": 1})
	assert.Equal(t, []string{"critical", "bulk", "default"}, mgr.getQueues())
}

func TestStrictQueues(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	mgr.StrictPriorityQueues("critical", "default", "bulk")
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.getQueues())
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.getQueues())
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.getQueues())
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.getQueues())

	mgr.StrictPriorityQueues("default", "critical", "bulk")
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.getQueues())
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.getQueues())
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.getQueues())
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.getQueues())
}

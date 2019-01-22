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
	mgr.Queues = []string{"critical", "default", "bulk"}
	mgr.UseWeightedQueues([]int{3, 2, 1})
	assert.Equal(t, []string{"critical", "default", "bulk"}, mgr.getQueues())
	assert.Equal(t, []string{"critical", "bulk", "default"}, mgr.getQueues())

	mgr.UseWeightedQueues([]int{1, 100, 1000})
	assert.Equal(t, []string{"bulk", "default", "critical"}, mgr.getQueues())

	mgr.UseWeightedQueues([]int{1, 1000, 100})
	assert.Equal(t, []string{"default", "bulk", "critical"}, mgr.getQueues())

	mgr.UseWeightedQueues([]int{1, 1, 1})
	assert.Equal(t, []string{"default", "critical", "bulk"}, mgr.getQueues())
}

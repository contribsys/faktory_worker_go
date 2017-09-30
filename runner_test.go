package faktory_worker

import (
	"testing"

	"github.com/mperham/faktory"
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

func xTestOpen(t *testing.T) {
	t.Parallel()
	client, err := faktory.Open()
	assert.NoError(t, err)
	job := faktory.NewJob("somejob", 1, 2, 3)
	err = client.Push(job)
	assert.NoError(t, err)
}

func TestContext(t *testing.T) {
	t.Parallel()
	job := faktory.NewJob("something", 1, 2)
	ctx := ctxFor(job)
	assert.Equal(t, ctx.Jid(), job.Jid)
	_, ok := ctx.Deadline()
	assert.False(t, ok)
}

package faktory_worker

import (
	"testing"

	"github.com/contribsys/faktory"
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

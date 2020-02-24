package faktory_worker

import (
	"errors"
	"regexp"
	"testing"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

func TestSimpleContext(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	pool, err := faktory.NewPool(1)
	assert.NoError(t, err)
	mgr.Pool = pool

	job := faktory.NewJob("something", 1, 2)
	job.SetCustom("track", 1)

	//cl, err := faktory.Open()
	//assert.NoError(t, err)
	//cl.Push(job)

	ctx := ctxFor(mgr, job)
	assert.Equal(t, ctx.Jid(), job.Jid)
	assert.Empty(t, ctx.Bid())
	assert.Equal(t, "something", ctx.JobType())

	_, ok := ctx.Deadline()
	assert.False(t, ok)

	//assert.NoError(t, ctx.TrackProgress(45, "Working....", nil))

	err = ctx.Batch(func(b *faktory.Batch) error {
		return nil
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, NoAssociatedBatchError))

}

func TestBatchContext(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	pool, err := faktory.NewPool(1)
	assert.NoError(t, err)
	mgr.Pool = pool

	job := faktory.NewJob("something", 1, 2)
	job.SetCustom("track", 1)
	job.SetCustom("bid", "nosuchbatch")

	ctx := ctxFor(mgr, job)
	assert.Equal(t, ctx.Jid(), job.Jid)
	assert.Equal(t, "nosuchbatch", ctx.Bid())
	assert.Equal(t, "something", ctx.JobType())

	withServer(t, "ent", mgr, func(cl *faktory.Client) error {
		err = ctx.Batch(func(b *faktory.Batch) error {
			return nil
		})
		assert.Error(t, err)
		assert.Regexp(t, regexp.MustCompile("No such batch"), err.Error())
		return nil
	})
}

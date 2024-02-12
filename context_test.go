package faktory_worker

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

func TestSimpleContext(t *testing.T) {
	t.Parallel()

	pool, err := faktory.NewPool(1)
	assert.NoError(t, err)

	job := faktory.NewJob("something", 1, 2)
	job.SetCustom("track", 1)

	//cl, err := faktory.Open()
	//assert.NoError(t, err)
	//cl.Push(job)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	ctx = jobContext(ctx, pool, job)
	help := HelperFor(ctx)
	assert.Equal(t, help.Jid(), job.Jid)
	assert.Empty(t, help.Bid())
	assert.Equal(t, "something", help.JobType())

	_, ok := ctx.Deadline()
	assert.True(t, ok)

	//assert.NoError(t, ctx.TrackProgress(45, "Working....", nil))

	err = help.Batch(func(b *faktory.Batch) error {
		return nil
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, NoAssociatedBatchError))

}

func TestBatchContext(t *testing.T) {
	t.Parallel()

	pool, err := faktory.NewPool(1)
	assert.NoError(t, err)

	job := faktory.NewJob("something", 1, 2)
	job.SetCustom("track", 1)
	job.SetCustom("bid", "nosuchbatch")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	ctx = jobContext(ctx, pool, job)
	help := HelperFor(ctx)
	assert.Equal(t, help.Jid(), job.Jid)
	assert.Equal(t, "nosuchbatch", help.Bid())
	assert.Equal(t, "something", help.JobType())

	mgr := NewManager()
	mgr.Pool = pool

	withServer(t, "ent", mgr, func(cl *faktory.Client) error {
		err = help.Batch(func(b *faktory.Batch) error {
			return nil
		})
		assert.Error(t, err)
		assert.Regexp(t, regexp.MustCompile("No such batch"), err.Error())
		return nil
	})
}

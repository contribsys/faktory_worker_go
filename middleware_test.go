package faktory_worker

import (
	"context"
	"testing"
	"time"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

type middlewareValue string

var EXAMPLE = middlewareValue("a")

func TestMiddleware(t *testing.T) {
	mgr := NewManager()
	pool, err := faktory.NewPool(2)
	assert.NoError(t, err)
	mgr.Pool = pool

	mgr.Use(func(ctx context.Context, job *faktory.Job, next func(context.Context) error) error {
		modctx := context.WithValue(ctx, EXAMPLE, 4.0)
		return next(modctx)
	})

	counter := 0
	blahFunc := func(ctx context.Context, job *faktory.Job) error {
		assert.EqualValues(t, 4.0, ctx.Value(EXAMPLE))
		help := HelperFor(ctx)
		assert.Equal(t, job.Jid, help.Jid())
		assert.Equal(t, job.Type, help.JobType())
		assert.Equal(t, "", help.Bid())
		counter += 1
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	job := faktory.NewJob("blah", 1, 2)
	ctx = jobContext(ctx, mgr.Pool, job)
	assert.Nil(t, ctx.Value(EXAMPLE))
	assert.EqualValues(t, 0, counter)

	err = dispatch(ctx, mgr.middleware, job, blahFunc)

	assert.NoError(t, err)
	assert.EqualValues(t, 1, counter)
	assert.Nil(t, ctx.Value(EXAMPLE))

}

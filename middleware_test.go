package faktory_worker

import (
	"context"
	"testing"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	mgr := NewManager()
	pool, err := faktory.NewPool(2)
	assert.NoError(t, err)
	mgr.Pool = pool

	mgr.Use(func(ctx context.Context, job *faktory.Job, next func(context.Context) error) error {
		modctx := context.WithValue(ctx, "a", 4.0)
		return next(modctx)
	})

	counter := 0
	blahFunc := func(ctx context.Context, job *faktory.Job) error {
		assert.EqualValues(t, 4.0, ctx.Value("a"))
		counter += 1
		return nil
	}

	job := faktory.NewJob("blah", 1, 2)
	ctx := jobContext(mgr.Pool, job)
	assert.Nil(t, ctx.Value("a"))
	assert.EqualValues(t, 0, counter)

	err = dispatch(mgr.middleware, ctx, job, blahFunc)

	assert.NoError(t, err)
	assert.EqualValues(t, 1, counter)
	assert.Nil(t, ctx.Value("a"))

}

package faktory_worker

import (
	"context"
	"fmt"
	"testing"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

var (
	badGlobal = 1
)

func myFunc(ctx context.Context, args ...interface{}) error {
	badGlobal += 1
	return nil
}

func TestTesting(t *testing.T) {
	pool, err := faktory.NewPool(5)
	assert.NoError(t, err)
	perf := NewTestExecutor(pool)

	assert.EqualValues(t, 1, badGlobal)
	somejob := faktory.NewJob("sometype", 12, "foobar")
	err = perf.Execute(somejob, myFunc)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, badGlobal)

	ajob := faktory.NewJob("sometype", 12, "foobar")
	err = perf.Execute(ajob, func(ctx context.Context, args ...interface{}) error {
		helper := HelperFor(ctx)
		assert.Equal(t, 2, len(args))
		assert.EqualValues(t, 12, args[0])
		assert.EqualValues(t, "foobar", args[1])
		assert.EqualValues(t, ajob.Jid, helper.Jid())
		assert.EqualValues(t, "", helper.Bid())
		assert.EqualValues(t, ajob.Type, helper.JobType())
		return nil
	})
	assert.NoError(t, err)
	err = perf.Execute(ajob, func(ctx context.Context, args ...interface{}) error {
		return fmt.Errorf("Oops")
	})
	assert.Equal(t, "Oops", err.Error())
}

func TestRecursiveTest(t *testing.T) {
	pool, err := faktory.NewPool(5)
	assert.NoError(t, err)
	perf := NewTestExecutor(pool)

	somejob := faktory.NewJob("pushtype", 12, "foobar")
	res, err := perf.ExecuteWithResult(somejob, func(ctx context.Context, args ...interface{}) error {
		helper := HelperFor(ctx)
		helper.TrackProgress(10, "Starting", nil)

		assert.NotEmpty(t, helper.Jid())
		assert.Empty(t, helper.Bid())
		assert.Equal(t, "pushtype", helper.JobType())

		ajob := faktory.NewJob("sometype", 12, "foobar")
		ajob.Queue = "whale"
		helper.Push(ajob)

		helper.TrackProgress(100, "Finished", nil)
		return nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.EqualValues(t, 1, len(res.PushedJobs()))
	assert.EqualValues(t, "whale", res.PushedJobs()[0].Queue)
}

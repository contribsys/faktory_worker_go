package faktory_worker

import (
	"context"
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
	perf.Execute(somejob, myFunc)
	assert.EqualValues(t, 2, badGlobal)

	ajob := faktory.NewJob("sometype", 12, "foobar")
	perf.Execute(ajob, func(ctx context.Context, args ...interface{}) error {
		assert.Equal(t, 2, len(args))
		assert.EqualValues(t, 12, args[0])
		assert.EqualValues(t, "foobar", args[1])
		return nil
	})
}

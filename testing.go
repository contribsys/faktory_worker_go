package faktory_worker

import (
	"context"
	"encoding/json"

	faktory "github.com/contribsys/faktory/client"
)

type PerformExecutor interface {
	Execute(*faktory.Job, Perform) error
	ExecuteContext(context.Context, *faktory.Job, Perform) error
}

type testExecutor struct {
	*faktory.Pool
}

func NewTestExecutor(p *faktory.Pool) PerformExecutor {
	return &testExecutor{Pool: p}
}

func (tp *testExecutor) Execute(specjob *faktory.Job, p Perform) error {
	ctx := context.Background()
	return tp.ExecuteContext(ctx, specjob, p)
}

func (tp *testExecutor) ExecuteContext(ctx context.Context, specjob *faktory.Job, p Perform) error {
	// perform a JSON round trip to ensure Perform gets the arguments
	// exactly how a round trip to Faktory would look.
	data, err := json.Marshal(specjob)
	if err != nil {
		return err
	}
	var job faktory.Job
	err = json.Unmarshal(data, &job)
	if err != nil {
		return err
	}

	c := jobContext(ctx, tp.Pool, &job)
	return p(c, job.Args...)
}

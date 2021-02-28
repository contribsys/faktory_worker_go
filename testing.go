package faktory_worker

import (
	"context"
	"encoding/json"

	faktory "github.com/contribsys/faktory/client"
)

type PerformExecutor interface {
	// This executes the Perform func with full network
	// access to the helper.
	Execute(*faktory.Job, Perform) error

	// This stubs out the helper so your Perform func can
	// use it without network access.
	ExecuteWithResult(*faktory.Job, Perform) (ExecuteResult, error)
}

type testExecutor struct {
	*faktory.Pool
}

func NewTestExecutor(p *faktory.Pool) PerformExecutor {
	return &testExecutor{Pool: p}
}

func (tp *testExecutor) Execute(specjob *faktory.Job, p Perform) error {
	_, err := tp.exec(specjob, p, false)
	return err
}

func (tp *testExecutor) ExecuteWithResult(specjob *faktory.Job, p Perform) (ExecuteResult, error) {
	ctx, err := tp.exec(specjob, p, true)
	if err != nil {
		return nil, err
	}
	return HelperFor(ctx).(ExecuteResult), nil
}

type ExecuteResult interface {
	PushedJobs() []*faktory.Job
}

func (tp *testExecutor) exec(specjob *faktory.Job, p Perform, local bool) (context.Context, error) {
	// perform a JSON round trip to ensure Perform gets the arguments
	// exactly how a round trip to Faktory would look.
	data, err := json.Marshal(specjob)
	if err != nil {
		return nil, err
	}
	var job faktory.Job
	err = json.Unmarshal(data, &job)
	if err != nil {
		return nil, err
	}

	c := jobContext(local, tp.Pool, &job)
	return c, p(c, job.Args...)
}

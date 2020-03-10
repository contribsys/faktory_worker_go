package faktory_worker

import (
	"context"
	"encoding/json"

	faktory "github.com/contribsys/faktory/client"
)

type PerformExecutor interface {
	Execute(*faktory.Job, Perform) error
}

type testExecutor struct {
	*faktory.Pool
}

func NewTestExecutor(p *faktory.Pool) PerformExecutor {
	return &testExecutor{Pool: p}
}

func (tp *testExecutor) Execute(specjob *faktory.Job, p Perform) error {
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

	c := &defaultContext{
		Context: context.Background(),
		JID:     job.Jid,
		Type:    job.Type,
		Pool:    tp.Pool,
	}
	s, _ := job.GetCustom("bid")
	if s != nil {
		c.BID = s.(string)
	}
	return p(c, job.Args...)
}

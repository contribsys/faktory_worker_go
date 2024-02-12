package faktory_worker

import (
	"context"

	faktory "github.com/contribsys/faktory/client"
)

type Handler func(ctx context.Context, job *faktory.Job) error
type MiddlewareFunc func(ctx context.Context, job *faktory.Job, next func(ctx context.Context) error) error

// Use(...) adds middleware to the chain.
func (mgr *Manager) Use(middleware ...MiddlewareFunc) {
	mgr.middleware = append(mgr.middleware, middleware...)
}

func dispatch(ctx context.Context, chain []MiddlewareFunc, job *faktory.Job, perform Handler) error {
	if len(chain) == 0 {
		return perform(ctx, job)
	}

	link := chain[0]
	rest := chain[1:]
	return link(ctx, job, func(ctx context.Context) error {
		return dispatch(ctx, rest, job, perform)
	})
}

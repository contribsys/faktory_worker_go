# faktory\_worker\_go

## 1.7.0

- Implement hard shutdown timeout of 25 seconds. [#76]
  Your job funcs should implement `context` package semantics.
  If you use `Manager.Run()`, FWG will now gracefully shutdown.
  After a default delay of 25 seconds, FWG will cancel the root Context which should quickly cancel any lingering jobs running under that Manager.
  If your jobs run long and do not respond to context cancellation, you risk orphaning any jobs in-progress.
  They will linger on the Busy tab until the job's `reserve_for` timeout.

  Please also note that `RunWithContext` added in `1.6.0` does not implement the shutdown delay but the README example contains the code to implement it.

## 1.6.0

- Upgrade to Go 1.17 and Faktory 1.6.0.
- Add `Manager.RunWithContext(ctx context.Context) error` [#58]
  Allows the caller to directly control when FWG stops. See README for usage.

## 1.5.0

- Auto-shutdown worker process if heartbeat expires due to network issues [#57]
- Send process RSS to Faktory (only available on Linux)
- Fix connection issue which causes worker processes to appear and disappear on
  Busy tab. [#47]

## 1.4.0

- **Breaking API changes due to misunderstanding the `context` package.**
  I've had to make significant changes to FWG's public APIs to allow
  for mutable contexts. This unfortunately requires breaking changes:
```
Job Handler
Before: func(ctx worker.Context, args ...interface{}) error
After: func(ctx context.Context, args ...interface{}) error

Middleware Handler
Before: func(ctx worker.Context, job *faktory.Job) error
After: func(ctx context.Context, job *faktory.Job, next func(context.Context) error) error
```
  Middleware funcs now need to call `next` to continue job dispatch.
  Use `help := worker.HelperFor(ctx)` to get the old APIs provided by `worker.Context`
  within your handlers.
- Fix issue reporting job errors back to Faktory
- Add helpers for testing `Perform` funcs
```go
myFunc := func(ctx context.Context, args ...interface{}) error {
	return nil
}

pool, err := faktory.NewPool(5)
perf := worker.NewTestExecutor(pool)

somejob := faktory.NewJob("sometype", 12, "foobar")
err = perf.Execute(somejob, myFunc)
```

## 1.3.0

- Add new job context APIs for Faktory Enterprise features
- Misc API refactoring to improve test coverage
- Remove `faktory_worker_go` Pool interface in favor of the Pool now in `faktory/client`

## 1.0.1

- FWG will now dump all thread backtraces upon `kill -TTIN <pid>`,
  useful for debugging stuck job processing.
- Send current state back to Faktory so process state changes are visible on Busy page.
- Tweak log output formatting

## 1.0.0

- Allow process labels (visible in Web UI) to be set [#32]
- Add APIs to manage [Batches](https://github.com/contribsys/faktory/wiki/Ent-Batches).

## 0.7.0

- Implement weighted queue fetch [#20, nickpoorman]
- Initial version.

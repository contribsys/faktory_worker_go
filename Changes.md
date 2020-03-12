# faktory\_worker\_go

## HEAD

- **Breaking API overhaul due to misunderstanding the `context` package.**
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

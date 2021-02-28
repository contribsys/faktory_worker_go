# How to write tests

1. You can have a separate function for faktory manager setup that you can call in func main(), for e.g.

``` go
func faktoryManagerSetup() error {
  // create faktory manager
  mgr = worker.NewManager()

  // set logger to the app's logger
  mgr.Logger = app.Log

  // NewPool creates a new Pool object through which multiple clients
  // are managed.
  pool, err := faktory.NewPool(5)
  if err != nil {
    return err
  }

  mgr.Pool = pool

  mgr.Use(func(ctx context.Context, job *faktory.Job, next func(ctx context.Context) error) error {
    app.Log.Infof("Starting work on job %s of type %s\n", job.Jid, job.Type)
    err := next(ctx)
    app.Log.Infof("Finished work on job %s with error %v\n", job.Jid, err)
    return err
  })

  // use up to N goroutines to execute jobs
  mgr.Concurrency = 20

  // pull jobs from these queues, in this order of precedence
  mgr.ProcessStrictPriorityQueues("critical", "default", "bulk")

  // register a job type and the handler
  // jobs will go into the "default" queue
  mgr.Register("UpdateWiki", UpdateWikiHandler)

  // register another job type
  mgr.Register("UpdateReleaseStatus", UpdateReleaseStatusHandler)

  return nil
}

```

2. Now, when you are writing a test for a function where you enqueue a faktory job, you can call this function:

``` go
func TestMain(m *testing.M) {
  testSetUp()
  faktoryManagerSetup()
  exitCode := m.Run()
  os.Exit(exitCode)

```

3. Table driven testing is a very good way of avoiding duplicate code, and it works really well in this case. Let's take an example:

``` go
var payload []byte

func (suite *FunctionalTestSuite) TestJob_Errors() {
  uniqueID := uuid.NewUUID()
  testCases := []struct {
    name                string
    hasError            bool
    expectedError       string
    jobEnqueued         bool
  }{
    {
      "Malformed Payload",
      true,
      "unable to process payload",
      false,
    },
    {
      "Empty Payload",
      true,
      "empty payload",
      false,
    },
    {
      "Empty Metadata",
      true,
      "empty metadata",
      false,
    },
  }

  for _, testCase := range testCases {
    fn := func() {
      payload = []byte("test payload")
      pool, _ := faktory.NewPool(5)
      perf := worker.NewTestExecutor(pool)
      someJob := faktory.NewJob("UpdateWiki", uniqueID, payload)

      // now call the function that actually creates a job in your flow
      err = suite.processSomePayload(payload)

      if testCase.jobEnqueued && testCase.hasError {
        err = perf.Execute(someJob, func(ctx context.Context, args ...interface{}) error {
          return fmt.Errorf(testCase.expectedError)
        })
        suite.require.Error(err)
      }

      if !testCase.jobEnqueued && testCase.hasError {
        err = perf.Execute(someJob, func(ctx context.Context, args ...interface{}) error {
          return fmt.Errorf(testCase.expectedError)
        })
        suite.require.Error(err)
        suite.assert.Contains(err.Error(), testCase.expectedError)
      } else {
        err = perf.Execute(someJob, func(ctx context.Context, args ...interface{}) error {
          return nil
        })
        suite.NoError(err)
      }
    }

    suite.Run(testCase.name, fn)
  }
}

```

Another example for testing the handler response:

``` go
func (suite *FunctionalTestSuite) TestJob_Errors() {
  uniqueID := uuid.NewUUID()
  testCases := []struct {
    name                string
    hasError            bool
    jobEnqueued         bool
  }{
     {
       "Success",
       true,
       true,
     },
  }

  for _, testCase := range testCases {
    fn := func() {
      pool, _ := faktory.NewPool(5)
      perf := worker.NewTestExecutor(pool)
      var i interface{} = []string{uniqueID.String(), "payload"}
      jobArgs := []interface{}{i}
      someJob := faktory.NewJob("UpdateWiki", jobArgs...)

      err = perf.Execute(someJob, UpdateWikiHandler)
      suite.require.NotNil(err)
    }

    suite.Run(testCase.name, fn)
  }
}

```

# faktory_worker_go

![travis](https://travis-ci.org/contribsys/faktory_worker_go.svg?branch=master)

This repository provides a Faktory worker process for Go apps.  This
worker process fetches background jobs from the Faktory server and processes them.

How is this different from all the other Go background worker libraries?
They all use Redis or another "dumb" datastore.  This library is far
simpler because the Faktory server implements most of the data storage, retry logic,
Web UI, etc.

# Installation

You must install [Faktory](https://github.com/contribsys/faktory) first.
Then:

```
go get -u github.com/contribsys/faktory_worker_go
```

# Usage

To process background jobs, follow these steps:

1. Register your job types and their associated funcs
2. Set a few optional parameters
3. Start the processing

To stop the process, send the TERM or INT signal.

```go
import (
  "fmt"

  worker "github.com/contribsys/faktory_worker_go"
)

func someFunc(ctx worker.Context, args ...interface{}) error {
  fmt.Println("Working on job", ctx.Jid())
  return nil
}

func main() {
  mgr := worker.NewManager()

  // register job types and the function to execute them
  mgr.Register("SomeJob", someFunc)
  //mgr.Register("AnotherJob", anotherFunc)

  // use up to N goroutines to execute jobs
  mgr.Concurrency = 20

  // use N goroutines to fetch jobs plus N goroutines to report job results, with a pool
  // size of up to N*2 connections
  mgr.Dispatchers = 4

  // pull jobs from these queues, in this order of precedence
  mgr.ProcessStrictPriorityQueues("critical", "default", "bulk")

  // alternatively you can use weights to avoid starvation
  //mgr.ProcessWeightedPriorityQueues(map[string]int{"critical":3, "default":2, "bulk":1})

  // Start processing jobs, this method does not return
  mgr.Run()
}
```

See `test/main.go` for a working example.

# FAQ

* How do I specify the Faktory server location?

By default, it will use localhost:7419 which is sufficient for local development.
Use FAKTORY\_URL to specify the URL, e.g. `tcp://faktory.example.com:12345` or
use FAKTORY\_PROVIDER to specify the environment variable which does
contain the URL: FAKTORY\_PROVIDER=FAKTORYTOGO\_URL.  This level of
indirection is useful for SaaSes, Heroku Addons, etc.

* How do I push new jobs to Faktory?

```go
import (
  faktory "github.com/contribsys/faktory/client"
)

client, err := faktory.Open()
job := faktory.NewJob("SomeJob", 1, 2, 3)
err = client.Push(job)
```

See the Faktory client for
[Go](https://github.com/contribsys/faktory/blob/master/client/client.go) or
[Ruby](https://github.com/contribsys/faktory-ruby/blob/master/lib/faktory/client.rb).
You can implement a Faktory client in any programming langauge.
See [the wiki](https://github.com/contribsys/faktory/wiki) for details.

# Author

Mike Perham, @mperham

# License

This codebase is licensed via the Mozilla Public License, v2.0. https://choosealicense.com/licenses/mpl-2.0/

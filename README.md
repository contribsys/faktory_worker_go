# faktory_worker_go

This repository provides a Faktory worker process for Go apps.  This
worker process fetches background jobs from the Faktory server and processes them.

How is this different from all the other Go background worker libraries?
They all use Redis or another "dumb" datastore.  This library is far
simpler because the Faktory server implements most of the data storage, retry logic,
Web UI, etc.

# Installation

```
go get -u github.com/contribsys/faktory_worker_go
```

# Usage

```go
import (
  worker "github.com/contribsys/faktory_worker_go"
)

func main() {
  mgr := worker.NewManager()

  // register the types of jobs and how to process them
  mgr.Register("SomeJob", func() worker.Worker { return &SomeWorker{} })
  mgr.Register("AnotherJob", func() worker.Worker { return &AnotherWorker{} })

  // use up to N goroutines to execute jobs
  mgr.Concurrency = 20
  // pull jobs from these queues, in this order of precedence
  mgr.Queues = []string{"critical", "default", "bulk"}
  mgr.Start()
}
```

# License

This codebase is licensed MPL-2.0. https://choosealicense.com/licenses/mpl-2.0/

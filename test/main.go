package main

import (
	"fmt"
	"time"

	worker "github.com/contribsys/faktory_worker_go"
	"github.com/mperham/faktory/util"
)

func someFunc(ctx worker.Context, args ...interface{}) error {
	fmt.Println("Working on job", ctx.Jid())
	fmt.Println("Context", ctx)
	fmt.Println("Args", args)
	time.Sleep(1 * time.Second)
	return nil
}

func main() {
	util.LogInfo = true

	mgr := worker.NewManager()

	// register job types and the function to execute them
	mgr.Register("SomeJob", someFunc)
	mgr.Register("SomeWorker", someFunc)
	//mgr.Register("AnotherJob", anotherFunc)

	// use up to N goroutines to execute jobs
	mgr.Concurrency = 2

	// pull jobs from these queues, in this order of precedence
	mgr.Queues = []string{"critical", "default", "bulk"}

	// Start processing jobs, this method does not return
	mgr.Run()
}

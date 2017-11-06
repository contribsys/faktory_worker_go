package main

import (
	"fmt"
	"time"

	"github.com/contribsys/faktory"
	"github.com/contribsys/faktory/util"
	worker "github.com/contribsys/faktory_worker_go"
)

func someFunc(ctx worker.Context, args ...interface{}) error {
	util.Infof("Working on job %s", ctx.Jid())
	util.Infof("Context %v", ctx)
	util.Infof("Args %v", args)
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
	mgr.Concurrency = 20

	// pull jobs from these queues, in this order of precedence
	mgr.Queues = []string{"critical", "default", "bulk"}

	var quit bool
	mgr.On(worker.Shutdown, func() {
		quit = true
	})
	go func() {
		for {
			if quit {
				return
			}
			produce()
			time.Sleep(1 * time.Second)
		}
	}()

	// Start processing jobs, this method does not return
	mgr.Run()
}

// Push something for us to work on.
func produce() {
	cl, err := faktory.Open()
	if err != nil {
		panic(err)
	}

	err = cl.Push(faktory.NewJob("SomeJob", 1, 2, "hello"))
	if err != nil {
		panic(err)
	}
	fmt.Println(cl.Info())
}

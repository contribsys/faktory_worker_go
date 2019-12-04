package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	faktory "github.com/contribsys/faktory/client"
	worker "github.com/contribsys/faktory_worker_go"
)

func someFunc(ctx worker.Context, args ...interface{}) error {
	log.Printf("Working on job %s\n", ctx.Jid())
	log.Printf("Context %v\n", ctx)
	log.Printf("Args %v\n", args)
	time.Sleep(1 * time.Second)
	return nil
}

func batchFunc(ctx worker.Context, args ...interface{}) error {
	log.Printf("Working on job %s\n", ctx.Jid())
	if ctx.Bid() != "" {
		log.Printf("within %s...\n", ctx.Bid())
	}
	log.Printf("Context %v\n", ctx)
	log.Printf("Args %v\n", args)
	return nil
}

func main() {
	mgr := worker.NewManager()
	mgr.Use(func(perform worker.Handler) worker.Handler {
		return func(ctx worker.Context, job *faktory.Job) error {
			log.Printf("Starting work on job %s of type %s with custom %v\n", ctx.Jid(), ctx.JobType(), job.Custom)
			err := perform(ctx, job)
			log.Printf("Finished work on job %s with error %v\n", ctx.Jid(), err)
			return err
		}
	})

	// register job types and the function to execute them
	mgr.Register("SomeJob", someFunc)
	mgr.Register("SomeWorker", someFunc)
	mgr.Register("ImportImageJob", batchFunc)
	mgr.Register("ImportImageSuccess", batchFunc)
	//mgr.Register("AnotherJob", anotherFunc)

	// use up to N goroutines to execute jobs
	mgr.Concurrency = 20

	// pull jobs from these queues, in this order of precedence
	mgr.ProcessStrictPriorityQueues("critical", "default", "bulk")

	var quit bool
	mgr.On(worker.Shutdown, func() {
		quit = true
	})
	go func() {
		for {
			if quit {
				return
			}
			batch()
			produce()
			time.Sleep(1 * time.Second)
		}
	}()

	// Start processing jobs, this method does not return
	mgr.Run()
}

func batch() {
	cl, err := faktory.Open()
	if err != nil {
		panic(err)
	}

	hash, err := cl.Info()
	if err != nil {
		panic(err)
	}
	desc := hash["server"].(map[string]interface{})["description"].(string)
	if !strings.Contains(desc, "Enterprise") {
		return
	}

	// Batch example
	// We want to import all images associated with user 1234.
	// Once we've imported those two images, we want to fire
	// a success callback so we can notify user 1234.
	b := faktory.NewBatch(cl)
	b.Description = "Import images for user 1234"
	b.Success = faktory.NewJob("ImportImageSuccess", "user 1234")
	// Once we call Jobs(), the batch is off and running
	err = b.Jobs(func() error {
		b.Push(faktory.NewJob("ImportImageJob", "1"))
		return b.Push(faktory.NewJob("ImportImageJob", "2"))
	})
	if err != nil {
		panic(err)
	}

	st, err := cl.BatchStatus(b.Bid)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v", st)
}

// Push something for us to work on.
func produce() {
	cl, err := faktory.Open()
	if err != nil {
		panic(err)
	}

	job := faktory.NewJob("SomeJob", 1, 2, "hello")
	job.Custom = map[string]interface{}{
		"hello": "world",
	}
	err = cl.Push(job)
	if err != nil {
		panic(err)
	}
	fmt.Println(cl.Info())
}

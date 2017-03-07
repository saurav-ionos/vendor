package main

import "fmt"

//var LogWorkerQueue chan chan LogMsgRequest

func StartLogDispatcher(nworkers int, workQueue chan LogMsgRequest, logPipeLineAddr string, logPipeLineType string, priority string) {
	// First, initialize the channel we are going to but the workers' work channels into.
	LogWorkerQueue := make(chan chan LogMsgRequest, nworkers)

	// Now, create all of our workers.
	for i := 0; i < nworkers; i++ {
		fmt.Println("Starting worker", i+1)
		worker := NewWorker(i+1, LogWorkerQueue, logPipeLineAddr, logPipeLineType, priority)
		worker.Start()
	}

	go func() {
		for {
			select {
			case work := <-workQueue:
				go func(work LogMsgRequest) {
					worker := <-LogWorkerQueue

					worker <- work
				}(work)
			}
		}
	}()
}

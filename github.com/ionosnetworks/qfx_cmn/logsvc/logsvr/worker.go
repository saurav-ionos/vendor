package main

import (
	"fmt"
	"net"
	"time"

	"github.com/ionosnetworks/qfx_cmn/msgq/producer"
)

// NewWorker creates, and returns a new Worker object. Its only argument
// is a channel that the worker can add itself to whenever it is done with its
// work.
func NewWorker(id int, workerQueue chan chan LogMsgRequest, logPipeLineAddr string, logPipeLineType string, workerType string) Worker {
	// Create, and return the worker.
	worker := Worker{
		ID:              id,
		Work:            make(chan LogMsgRequest),
		WorkerQueue:     workerQueue,
		QuitChan:        make(chan bool),
		LogPipeLineAddr: logPipeLineAddr,
		LogPipeLineType: logPipeLineType,
        WorkerType:      workerType,
	}

	return worker
}

const (
	KAFKA = "KAFKA"
    HIGH_PRIORITY = "HIGH_PRIORITY"
    LOW_PRIORITY = "LOW_PRIORITY"
    KAFKA_HIGH_PRIO_TOPIC = "LOGMSG_HIGH_PRIO"
    KAFKA_LOW_PRIO_TOPIC = "LOGMSG_LOW_PRIO"
)

type Worker struct {
	ID              int
	Work            chan LogMsgRequest
	WorkerQueue     chan chan LogMsgRequest
	QuitChan        chan bool
	LogPipeLineAddr string
	LogPipeLineType string
    WorkerType      string
}

// This function "starts" the worker by starting a goroutine, that is
// an infinite "for-select" loop.
func (w *Worker) Start() {
	go func() {
		var conn net.Conn
		if w.LogPipeLineType != KAFKA {
			conn = GetConnectionToLogPipeLine(w.LogPipeLineAddr)
			for conn == nil {
				conn = GetConnectionToLogPipeLine(w.LogPipeLineAddr)
				time.Sleep(3 * time.Second)
			}
		}
		for {
			// Add ourselves into the worker queue.
			w.WorkerQueue <- w.Work

			select {
			case work := <-w.Work:
				// Receive a work request.
				fmt.Printf("worker%d: Received work request, work=%s\n", w.ID, work)
				if w.LogPipeLineType != KAFKA {
					_, err := conn.Write([]byte(work.Msg))
					if err != nil {
						fmt.Println("Conn to LogPipeLine returned error")
						conn = nil
						for conn == nil {
							conn = GetConnectionToLogPipeLine(w.LogPipeLineAddr)
							time.Sleep(3 * time.Second)
						}
					}
				} else {
					fmt.Printf("msg:%s", work.Msg)
                    var topic string
                    topic = KAFKA_HIGH_PRIO_TOPIC
                    if w.WorkerType == LOW_PRIORITY {
                       topic = KAFKA_LOW_PRIO_TOPIC
                    }
					producer.SendAsyncMessage(topic, "", []byte(work.Msg))
				}
			case <-w.QuitChan:
				// We have been asked to stop.
				fmt.Printf("Worker%d stopping\n", w.ID)
				return
			}
		}
	}()
}

// Stop tells the worker to stop listening for work requests.
//
// Note that the worker will only stop *after* it has finished its work.
func (w *Worker) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}

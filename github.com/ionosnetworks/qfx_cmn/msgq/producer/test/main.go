package main

import (
	"fmt"
    "os"
    "os/signal"
    "syscall"
    "github.com/ionosnetworks/msgq/producer"
)

func main() {

    if len(os.Args) != 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s <broker>\n",
            os.Args[0])
        os.Exit(1)
    }
  
    broker := os.Args[1]
    brokers := []string{broker}
    producer.Init("testProducer", brokers)
    producer.PrintConfig()

    //producer.SendSyncMessage("topic1","1234",[]byte ("test message"))
    producer.SendAsyncMessage("LOGMSG_LOW","12345",[]byte ("test message"))
    wait := make(chan os.Signal, 1)
    signal.Notify(wait, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
    <-wait
}

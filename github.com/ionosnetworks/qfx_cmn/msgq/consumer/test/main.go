package main

import (
	"fmt"
    "os"
    "os/signal"
    "syscall"
    "github.com/ionosnetworks/msgq/consumer"
)

func main() {

    if len(os.Args) != 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s <broker>\n",
            os.Args[0])
        os.Exit(1)
    }
  
    broker := os.Args[1]
    brokers := []string{broker}
    mymap := make(map[string][]string)
    mymap["group1"] = []string{"topic1","topic2"}
    ch := make (chan []byte,10)

    consumer.Init("testProducer", brokers, mymap, ch)
    for msg := range ch {
        fmt.Printf("Msg Rcvd: %s\n",msg)
    }
    consumer.PrintConfig()
     wait := make(chan os.Signal, 1)
    signal.Notify(wait, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
    <-wait

}

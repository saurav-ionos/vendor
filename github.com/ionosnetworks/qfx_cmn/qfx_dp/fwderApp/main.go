package main

import (
	"github.com/ionosnetworks/qfx_dp/fwder"
)

func main() {
	done := make(chan struct{}, 1)
	FwderToDpChannel := make(chan string, 1024)
	go fwder.StartFwder(FwderToDpChannel)
	<-done
}

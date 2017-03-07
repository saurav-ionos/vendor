package main

import (
	"github.com/ionosnetworks/qfx_dp/cp"
	"github.com/ionosnetworks/qfx_dp/dp"
	"github.com/ionosnetworks/qfx_dp/fwder"
	"github.com/ionosnetworks/qfx_dp/qfsync"
	"github.com/ionosnetworks/qfx_dp/slcemulator"
)

func main() {

	done := make(chan struct{})
	cp.Start()
	FwderToDpChannel := make(chan fwder.StorMsg, 1024)

	go dp.InitIcaDp(FwderToDpChannel, QfsToPipelineCh)
	// Start the SLC emulator
	slcemulator.Start()

	go fwder.StartFwder(FwderToDpChannel)

	qfsync.Init(QfsToPipelineCh)

	<-done
}

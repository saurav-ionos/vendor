package demux

import (
	"fmt"
	"net"
	"os"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_cmn/msgsvc/msgcli"
)

var logger blog.Logger
var glMsgSrvr string = "127.0.0.1"
var accesskey string = "0123456789"
var secret string = "abcdefghijklmnopqrstuvwxyz"
var serverport string = "8080"
var logSrvr string = "192.168.1.141:2000"
var glMsgCli *msgcli.MsgCli
var demuxID string
var ctx string = "demuxer"

func Start() {

	if srvr := os.Getenv("MSGSVC_SLC_AP"); srvr != "" {
		glMsgSrvr = srvr
	}

	// Create the logger
	if srvr := os.Getenv("LOG_SERVER"); srvr != "" {
		logSrvr = srvr
	}
	var err error
	logger = blog.New(logSrvr, accesskey, secret)
	if logger == nil {
		panic("could not create logger")
	}
	logger.SetLevel(blog.Debug)

	demuxID = os.Getenv("SLC_UUID")
	if demuxID == "" {
		panic("demuxID not specified")
	}

	fmt.Println(glMsgSrvr, demuxID, serverport, accesskey, secret)

	glMsgCli, err = msgcli.New(glMsgSrvr, demuxID, serverport,
		accesskey, secret, "", nil)

	if err != nil {
		panic(err)
	}

	demuxPackets(glMsgCli.MsgRecv)
}

func demuxPackets(ch chan msgcli.MsgPkt) {

nextPacket:
	for x := range ch {
		// forward to capdisc service
		conn, err := net.Dial("tcp4", "capdisc-service:3000")
		if err != nil {
			logger.Err(ctx, "error connecting to capdisc service",
				blog.Fields{"err": err.Error()})
			continue
		}
		// We have a valid connection forward the packet to it
		toSend := len(x.PayLoad)
		written := 0
		for toSend > 0 {
			nbytes, err := conn.Write(x.PayLoad[written:])
			if err != nil {
				continue nextPacket
			}
			toSend -= nbytes
			written += nbytes

		}
	}
}

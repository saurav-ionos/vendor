package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	mclient "github.com/ionosnetworks/qfx_cmn/msgsvc/msgcli"

	msgc "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs"
	qfxctrl "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs/QfxCtrl"
	cnts "github.com/ionosnetworks/qfx_cp/qfxConsts"
)

const (
	SVC_KEY_FILE = "/keys/keyfile"
)

var (
	serverport = "8080"
	glMsgSvr   = ""
	key        kr.AccessKey
)

func main() {

	key = kr.New(SVC_KEY_FILE)
	glMsgSvr = os.Args[1]

	switch os.Args[2] {
	case "multiclient":
		multipleclients() // Multiple client test with unicast.
	case "multicast":
		multicast(false) // Multicast testcase
	case "logintest":
		loginTest()
	case "clientinfo":
		clientinfo()
	default:
		fmt.Println("Options : multicast multiclient logintest clientinfo ")
	}

}

func clientinfo() {

	client := os.Args[3]
	fmt.Println("Usage :: MsgSvrAddr clientinfo  client DstIDs ... ")
	if len(os.Args) > 3 {
		dstlist := os.Args[4:]

		mcli, err := mclient.New(glMsgSvr, client, serverport, key.Key, key.Secret, "", nil)

		if err != nil {
			fmt.Println("Failed to initialize Msg Client")
			return
		}
		defer mcli.Close()

		err, statusmap := mcli.GetClientStatus(dstlist...)
		fmt.Println("Client status : ", err, statusmap)
	}
}

func loginTest() {

	client := os.Args[3]

	mcli, err := mclient.New(glMsgSvr, client, serverport, key.Key, key.Secret, "", nil)

	if err != nil {
		fmt.Println("Failed to initialize Msg Client")
		return
	}
	defer mcli.Close()
	go recvLoginMsg(mcli)

	for i := 0; ; i++ {
		mcli.Write(qfxctrl.MsgTypeCPE_STATUS,
			msgc.CreateCPEMessage(client, qfxctrl.MsgTypeCPE_STATUS, cnts.CPE_ONLINE, int32(i), 1))
		time.Sleep(10 * time.Second)
	}
}

// Testing multiple Clients
func multipleclients() {

	start, _ := strconv.Atoi(os.Args[3])
	diffCount, _ := strconv.Atoi(os.Args[4])
	cliCount, _ := strconv.Atoi(os.Args[5])

	fmt.Println("Usage :: MsgSvrAddr clientCount diffCount clientcount")

	for i := 0; i < cliCount; i++ {
		client := "cpe" + strconv.Itoa(i+start)
		dstlist := "cpe" + strconv.Itoa(i+diffCount+start)
		fmt.Println(glMsgSvr, client, " -----> ", dstlist)
		go msgClient(client, serverport, key.Key, key.Secret, false, dstlist)
	}
	fmt.Println("spawned all clients.")

	time.Sleep(time.Second * 90)

}

// this is to test multicast
func multicast(inLoop bool) {

	client := os.Args[3]

	fmt.Println("Usage :: MsgSvrAddr testcaseid ClientId  DstIDs ... ")
	if len(os.Args) > 3 {
		dstlist := os.Args[4:]
		msgClient(client, serverport, key.Key, key.Secret, inLoop, dstlist...)
	} else {
		msgClient(client, serverport, key.Key, key.Secret, inLoop, "")
	}
}

func msgClient(client, serverport, accesskey, secret string, inLoop bool, dstlist ...string) {

	mcli, err := mclient.New(glMsgSvr, client, serverport, accesskey, secret, "", nil)

	if err != nil {
		fmt.Println("Failed to initialize Msg Client")
		return
	}
	defer mcli.Close()
	go recvFunc(mcli)

	time.Sleep(time.Second * 10)

	if len(dstlist) > 0 {

		xfer, errMap := mcli.InitXfer(dstlist...)

		fmt.Println("ErrMap ", errMap)

		if xfer != nil {

			payload := []byte("Sample message from " + client)
			//payload := extra

			for {
				for i := 0; i < 10; i++ {
					xfer.Write(payload)
				}
				if inLoop == false {
					break
				}
				// xfer.Revalidate()
				fmt.Println("Sent 10 pkts")
				time.Sleep(10 * time.Second)
			}
			xfer.Close()
		}
	}
	for inLoop == true {
		fmt.Println("..")
		time.Sleep(time.Minute)
	}

	time.Sleep(time.Minute)
}

func recvFunc(mcli *mclient.MsgCli) {
	count := 0
	for x := range mcli.MsgRecv {
		count++

		fmt.Println("Recieved  ", count, len(x.PayLoad), x.Source,
			x.ClientTs, x.CntrlRcvTs, x.CntrlDstTs, x.DestTs)
	}
}

func recvLoginMsg(mcli *mclient.MsgCli) {
	count := 0
	for x := range mcli.MsgRecv {
		count++
		fmt.Println("Recieved  ", count, len(x.PayLoad), x.Source)

		buf := x.PayLoad

		msg2 := qfxctrl.GetRootAsQfxCtrlMsg(buf, 0)

		utable := new(flatbuffers.Table)

		if msg2.Msg(utable) {

			switch msg2.Msgtype() {

			case qfxctrl.MsgTypeCTRL_MSG_ACK:
				ackM := new(qfxctrl.QfxMsgAck)

				ackM.Init(utable.Bytes, utable.Pos)
				dst := string(ackM.Dstid())

				fmt.Println("Ack Received for ", dst, ackM.Msgid())
			}
		}
	}
}

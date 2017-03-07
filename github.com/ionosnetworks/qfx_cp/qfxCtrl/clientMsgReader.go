package main

import (
	"fmt"
	"os"

	flatbuffers "github.com/google/flatbuffers/go"
	msgqc "github.com/ionosnetworks/qfx_cmn/msgq/consumer"
	msgqp "github.com/ionosnetworks/qfx_cmn/msgq/producer"

	msgc "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs"
	msg "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs/QfxCtrl"
	cnts "github.com/ionosnetworks/qfx_cp/qfxConsts"
)

var (
	readTopicList = []string{"test"}
	msgqGroup     = "group1"
	writeTopic    = "topic1"
)

func InitMsgq(msgQueueAP string) chan []byte {

	brokers := []string{msgQueueAP}
	groupMap := make(map[string][]string)
	groupMap[msgqGroup] = readTopicList

	ch := make(chan []byte, 10)
	msgqc.Init(cnts.GLOBAL_CONTROLLER_UUID, brokers, groupMap, ch)
	msgqp.Init(cnts.GLOBAL_CONTROLLER_UUID, brokers)

	return ch
}

func ReadMessageFromController(ch chan []byte) {
	for val := range ch {
		//fmt.Printf("Msg: %s\n", string(val[:]))
		parseMsg(val)
	}

	// TODO:: If the packet is for msg controller, act on it.
}

func parseMsg(buf []byte) {

	msg2 := msg.GetRootAsQfxCtrlMsg(buf, 0)
	src := ""

	utable := new(flatbuffers.Table)

	if msg2.Msg(utable) {

		switch msg2.Msgtype() {

		case msg.MsgTypeCPE_STATUS:

			cpeM := new(msg.CpeStatus)
			cpeM.Init(utable.Bytes, utable.Pos)
			src = string(cpeM.CpeId())
			fmt.Println("CPE ID", src)
			fmt.Println("Status", cpeM.Status())

		case msg.MsgTypeCTRL_LOGIN:

			ctrlM := new(msg.CtrlLogin)
			ctrlM.Init(utable.Bytes, utable.Pos)
			src = string(ctrlM.CtrlId())
			fmt.Println("Ctrl Login ", src)
			fmt.Println("Ctrl type ", ctrlM.Ctrltype())

		case msg.MsgTypeCTRL_LOGOUT:
			ctrlM := new(msg.CtrlLogout)
			ctrlM.Init(utable.Bytes, utable.Pos)
			src = string(ctrlM.CtrlId())
			fmt.Println("Ctrl Logout ", src)

		case msg.MsgTypeCTRL_MSG_ACK:
			ackM := new(msg.QfxMsgAck)

			ackM.Init(utable.Bytes, utable.Pos)
			fmt.Println("Ack from ", ackM.Dstid())

		}
	}

	if msg2.NeedAck() == 1 {
		// Send ack..
		pkt := msgc.CreateMessageAck(src, msg2.Msgid())

		fmt.Println("Sending ack ", pkt[:10])
		msgqp.SendSyncMessage(writeTopic, "1234", pkt)
	}
}

func main() {

	if msgqAddr := os.Getenv("MSGSVC_MSGQ_ADDR"); msgqAddr != "" {
		ch := InitMsgq(msgqAddr)
		ReadMessageFromController(ch)
	}
}

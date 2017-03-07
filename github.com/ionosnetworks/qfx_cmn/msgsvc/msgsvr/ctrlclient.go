package main

import (
	"bytes"
	"encoding/json"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/ionosnetworks/qfx_cmn/blog"
	msgqc "github.com/ionosnetworks/qfx_cmn/msgq/consumer"
	msgqp "github.com/ionosnetworks/qfx_cmn/msgq/producer"
	cmn "github.com/ionosnetworks/qfx_cmn/msgsvc/common"
	qfxctrlmsg "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs"
	qfxctrl "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs/QfxCtrl"
	cnts "github.com/ionosnetworks/qfx_cp/qfxConsts"
)

const (
	// This is the UUID of the main controller. if the
	// the packet is sent with ID as the destination
	// packet will be pushed to kafka bus.
	controllerID = cnts.GLOBAL_CONTROLLER_UUID
)

var (
	readTopicList   = []string{"topic1", "topic2"}
	msgqGroup       = "group1"
	writeTopicMap   = make(map[int32]string) // "test"
	defaultTopic    = "test"
	msgqInitialized = false
)

/*
  We will have this routine running only if controller is writing messages to a bus.
	In our case, Global controller will write mesages to kafka bus.
*/
func (msgsvr *MsgSvr) ControllerMessageClient(msgQueueAP string) {

	// Register with the server as a normal client.
	SetControllerForClient(controllerID, msgsvr.msgCtrlUUID, 0)

	// Send
	var aplist []cmn.ClientAccessPoint
	ap := msgsvr.msgLbAddr + ":" + msgsvr.port
	aplist = append(aplist, cmn.ClientAccessPoint{AccessPoint: ap,
		Type: cmn.LOCAL_MSG_CTRL_IP})

	var bin_buf bytes.Buffer
	enc := json.NewEncoder(&bin_buf)
	enc.Encode(aplist)

	jsonstr := string(bin_buf.Bytes())

	SetClientIp(controllerID, jsonstr)

	ch := make(chan []byte, 10)

	brokers := []string{msgQueueAP}
	groupMap := make(map[string][]string)
	groupMap[msgqGroup] = readTopicList

	msgqc.Init(msgsvr.msgCtrlUUID, brokers, groupMap, ch)
	msgqp.Init(msgsvr.msgCtrlUUID, brokers)

	// Populate the map
	writeTopicMap[qfxctrl.MsgTypeCPE_STATUS] = "test"
	msgqInitialized = true
	// Keep reading the messages from kafka bus. Send it to clients.
	msgsvr.ReadMessageFromController(ch)

}

/*
  This function will read the messages from kafka bus.
*/
func (msgsvr *MsgSvr) ReadMessageFromController(ch chan []byte) {

	for buf := range ch {

		msg2 := qfxctrl.GetRootAsQfxCtrlMsg(buf, 0)

		utable := new(flatbuffers.Table)

		if msg2.Msg(utable) {

			switch msg2.Msgtype() {

			case qfxctrl.MsgTypeCTRL_MSG_ACK:

				ackM := new(qfxctrl.QfxMsgAck)

				ackM.Init(utable.Bytes, utable.Pos)
				dst := string(ackM.Dstid())

				if netConn, _ := msgsvr.getConnForClient(dst); netConn == nil {
					logger.Info(ctx, "Client not logged in ", blog.Fields{"ID": dst})
				} else {

					err := cmn.CreateAndSendMessage(*netConn, cnts.GLOBAL_CONTROLLER_UUID, dst,
						cmn.CLIENT_MSG, cmn.CLIENT_MSG, buf)
					if err != nil {
						// Need to re-evaluate path.
					}
				}

			}
		}
	}

	// TODO:: If the packet is for msg controller, act on it.
}

/*
 We will write the message to kafka bus.
*/
func SendMessageToController(pkt cmn.MsgPkt) {

	if msgqInitialized == true {
		writeTopic := writeTopicMap[pkt.MesgMetaHeader.MsgSubType]
		msgqp.SendSyncMessage(writeTopic, "1234", *pkt.Payload)
	}
}

func sendLogoutMessagetoController(clientId string) {

	writeTopic := ""
	found := false

	payload := qfxctrlmsg.CreateCPEMessage(clientId, qfxctrl.MsgTypeCPE_STATUS, cnts.CPE_OFFLINE, 1, 0)
	if msgqInitialized == true {
		if writeTopic, found = writeTopicMap[qfxctrl.MsgTypeCPE_STATUS]; found == false {
			writeTopic = defaultTopic
		}
		msgqp.SendSyncMessage(writeTopic, "1234", payload)
	}
}

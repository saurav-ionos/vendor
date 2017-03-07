package main

import (
	"fmt"

	flatbuffers "github.com/google/flatbuffers/go"
	msgc "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs"
	msg "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs/QfxCtrl"
)

func parseMsg(buf []byte) {

	msg2 := msg.GetRootAsQfxCtrlMsg(buf, 0)

	fmt.Println("Mtype ", msg2.Msgtype())
	utable := new(flatbuffers.Table)

	if msg2.Msg(utable) {

		switch msg2.Msgtype() {

		case msg.MsgTypeCPE_STATUS:

			cpeM := new(msg.CpeStatus)
			cpeM.Init(utable.Bytes, utable.Pos)
			fmt.Println("CPE ID", string(cpeM.CpeId()))
			fmt.Println("Status", cpeM.Status())

		case msg.MsgTypeCTRL_LOGIN:

			ctrlM := new(msg.CtrlLogin)
			ctrlM.Init(utable.Bytes, utable.Pos)
			fmt.Println("Ctrl Login ", string(ctrlM.CtrlId()))
			fmt.Println("Ctrl type ", ctrlM.Ctrltype())

		case msg.MsgTypeCTRL_LOGOUT:
			ctrlM := new(msg.CtrlLogout)
			ctrlM.Init(utable.Bytes, utable.Pos)
			fmt.Println("Ctrl Logout ", string(ctrlM.CtrlId()))

		}
	}
}

func main() {
	id := "12345678"
	ctid := "ctrl" + id
	parseMsg(msgc.CreateCPEMessage(id, msg.MsgTypeCPE_STATUS, msgc.CPE_ONLINE, 0, 0))
	parseMsg(msgc.CreateCtrlLoginMessage(ctid, msg.MsgTypeCTRL_LOGIN, 0, 1, 0))
	parseMsg(msgc.CreateCtrlLogoutMessage(ctid, 2, 0))
}

package QfxCtrlMsgs

import (
	flatbuffers "github.com/google/flatbuffers/go"

	msg "github.com/ionosnetworks/qfx_cp/QfxCtrlMsgs/QfxCtrl"
)

func CreateCPEMessage(id string, msgType, status, msgid, needAck int32) []byte {

	builder := flatbuffers.NewBuilder(0)

	cpeId := builder.CreateString(id)

	msg.CpeStatusStart(builder)
	msg.CpeStatusAddCpeId(builder, cpeId)
	msg.CpeStatusAddStatus(builder, status)
	cpeMsg := msg.CpeStatusEnd(builder)

	msg.QfxCtrlMsgStart(builder)

	msg.QfxCtrlMsgAddMsg(builder, cpeMsg)
	msg.QfxCtrlMsgAddMsgtype(builder, msgType)
	msg.QfxCtrlMsgAddMsgid(builder, msgid)
	msg.QfxCtrlMsgAddNeedAck(builder, needAck)

	msgFB := msg.QfxCtrlMsgEnd(builder)
	builder.Finish(msgFB)
	return builder.FinishedBytes()
}

func CreateCtrlLoginMessage(id string, msgType, ctrlType, msgid, needAck int32) []byte {

	builder := flatbuffers.NewBuilder(0)

	ctrlId := builder.CreateString(id)

	msg.CtrlLoginStart(builder)
	msg.CtrlLoginAddCtrlId(builder, ctrlId)
	msg.CtrlLoginAddCtrltype(builder, ctrlType)
	cpeMsg := msg.CtrlLoginEnd(builder)

	msg.QfxCtrlMsgStart(builder)

	msg.QfxCtrlMsgAddMsg(builder, cpeMsg)
	msg.QfxCtrlMsgAddMsgtype(builder, msgType)
	msg.QfxCtrlMsgAddMsgid(builder, msgid)
	msg.QfxCtrlMsgAddNeedAck(builder, needAck)

	msgFB := msg.QfxCtrlMsgEnd(builder)
	builder.Finish(msgFB)
	return builder.FinishedBytes()
}

func CreateCtrlLogoutMessage(id string, msgid, needAck int32) []byte {

	builder := flatbuffers.NewBuilder(0)

	ctrlId := builder.CreateString(id)

	msg.CtrlLogoutStart(builder)
	msg.CtrlLogoutAddCtrlId(builder, ctrlId)
	cpeMsg := msg.CtrlLogoutEnd(builder)

	msg.QfxCtrlMsgStart(builder)

	msg.QfxCtrlMsgAddMsg(builder, cpeMsg)
	msg.QfxCtrlMsgAddMsgtype(builder, msg.MsgTypeCTRL_LOGOUT)
	msg.QfxCtrlMsgAddMsgid(builder, msgid)
	msg.QfxCtrlMsgAddNeedAck(builder, needAck)

	msgFB := msg.QfxCtrlMsgEnd(builder)
	builder.Finish(msgFB)
	return builder.FinishedBytes()
}

func CreateMessageAck(id string, msgid int32) []byte {

	builder := flatbuffers.NewBuilder(0)
	dstId := builder.CreateString(id)
	msg.QfxMsgAckStart(builder)
	msg.QfxMsgAckAddMsgid(builder, msgid)
	msg.QfxMsgAckAddDstid(builder, dstId)
	msgAck := msg.QfxMsgAckEnd(builder)

	msg.QfxCtrlMsgStart(builder)

	msg.QfxCtrlMsgAddMsg(builder, msgAck)
	msg.QfxCtrlMsgAddMsgtype(builder, msg.MsgTypeCTRL_MSG_ACK)
	msg.QfxCtrlMsgAddMsgid(builder, msgid)

	msgFB := msg.QfxCtrlMsgEnd(builder)
	builder.Finish(msgFB)
	return builder.FinishedBytes()
}

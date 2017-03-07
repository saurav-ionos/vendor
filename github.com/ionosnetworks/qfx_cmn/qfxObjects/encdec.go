package qfxObjects

import (
	"errors"

	flatbuffers "github.com/google/flatbuffers/go"
	msg "github.com/ionosnetworks/qfx_cmn/qfxMsgs/dp2ctrl"
)

func (m *MsgWrapper) GobDecode(buf []byte) error {

	msg2 := msg.GetRootAsQfxCtrlMsg(buf, 0)

	utable := new(flatbuffers.Table)

	if msg2.Msg(utable) {
		m.MsgType = msg2.Msgtype()
		m.MsgId = msg2.Msgid()
		m.NeedAck = msg2.NeedAck()

		switch m.MsgType {

		case SITE_UP:
			sup := new(msg.SiteUp)
			sup.Init(utable.Bytes, utable.Pos)
			m.Data = Siteup{Id: string(sup.EntityId()), Entity: int(sup.EntityType())}

		case SITE_DOWN:
			sup := new(msg.SiteDown)
			sup.Init(utable.Bytes, utable.Pos)
			m.Data = Sitedown{Id: string(sup.EntityId()), Entity: int(sup.EntityType())}

		case QFX_MSG_ACK:
			sup := new(msg.SiteDown)
			sup.Init(utable.Bytes, utable.Pos)
			m.Data = nil

		default:
			return errors.New("No such data type")
		}
	}
	return nil

}

func (m MsgWrapper) GobEncode() ([]byte, error) {

	switch m.MsgType {

	case SITE_UP:
		createSiteUpMsg(m)

	case SITE_DOWN:
		createSiteDownMsg(m)

	case QFX_MSG_ACK:
		createMsgAck(m)

	default:
		return make([]byte, 0), errors.New("No such message type")

	}
	return nil, nil
}

func createMsgAck(m MsgWrapper) ([]byte, error) {

	builder := flatbuffers.NewBuilder(0)

	msg.QfxCtrlMsgStart(builder)

	msg.QfxCtrlMsgAddMsgtype(builder, QFX_MSG_ACK)
	msg.QfxCtrlMsgAddMsgid(builder, m.MsgId)
	msg.QfxCtrlMsgAddNeedAck(builder, 0) // No Ack for ACK.

	msgFB := msg.QfxCtrlMsgEnd(builder)
	builder.Finish(msgFB)
	return builder.FinishedBytes(), nil
}

func createSiteDownMsg(m MsgWrapper) ([]byte, error) {

	builder := flatbuffers.NewBuilder(0)
	f := m.Data.(Siteup)

	cpeId := builder.CreateString(f.Id)

	msg.SiteUpStart(builder)
	msg.SiteUpAddEntityId(builder, cpeId)
	msg.SiteUpAddEntityType(builder, ENTITY_CPE)

	siteupMsg := msg.SiteUpEnd(builder)

	msg.QfxCtrlMsgStart(builder)

	msg.QfxCtrlMsgAddMsg(builder, siteupMsg)
	msg.QfxCtrlMsgAddMsgtype(builder, SITE_DOWN)
	msg.QfxCtrlMsgAddMsgid(builder, m.MsgId)
	msg.QfxCtrlMsgAddNeedAck(builder, m.NeedAck)

	msgFB := msg.QfxCtrlMsgEnd(builder)
	builder.Finish(msgFB)
	return builder.FinishedBytes(), nil
}

func createSiteUpMsg(m MsgWrapper) ([]byte, error) {

	builder := flatbuffers.NewBuilder(0)
	f := m.Data.(Siteup)

	cpeId := builder.CreateString(f.Id)

	msg.SiteUpStart(builder)
	msg.SiteUpAddEntityId(builder, cpeId)
	msg.SiteUpAddEntityType(builder, ENTITY_CPE)

	siteupMsg := msg.SiteUpEnd(builder)

	msg.QfxCtrlMsgStart(builder)

	msg.QfxCtrlMsgAddMsg(builder, siteupMsg)
	msg.QfxCtrlMsgAddMsgtype(builder, SITE_UP)
	msg.QfxCtrlMsgAddMsgid(builder, m.MsgId)
	msg.QfxCtrlMsgAddNeedAck(builder, m.NeedAck)

	msgFB := msg.QfxCtrlMsgEnd(builder)
	builder.Finish(msgFB)
	return builder.FinishedBytes(), nil
}

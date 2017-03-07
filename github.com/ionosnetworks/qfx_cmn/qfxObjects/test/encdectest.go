package main

import (
	"fmt"

	msgc "github.com/ionosnetworks/qfx_cmn/qfxObjects"
)

func parseMsg(buf []byte) {

	var m msgc.MsgWrapper

	m.GobDecode(buf)
	fmt.Println(m)
	switch m.MsgType {

	case msgc.SITE_UP:
		fmt.Println("Site up :", m.Data.(msgc.Siteup))

	case msgc.SITE_DOWN:
		fmt.Println("Site down :", m.Data.(msgc.Sitedown))

	case msgc.QFX_MSG_ACK:
		fmt.Println("Message Ack   :")

	}

}

func main() {

	// Site up
	{
		id := "1234"
		siteup := msgc.Siteup{Id: id, Entity: msgc.ENTITY_CPE}
		m := msgc.MsgWrapper{MsgId: 1, NeedAck: 1, MsgType: msgc.SITE_UP, Data: siteup}

		buf, _ := m.GobEncode()
		parseMsg(buf)
	}
	// Site down
	{
		id := "4567"
		sitedown := msgc.Sitedown{Id: id, Entity: msgc.ENTITY_CPE}
		m := msgc.MsgWrapper{MsgId: 2, NeedAck: 1, MsgType: msgc.SITE_DOWN, Data: sitedown}

		buf, _ := m.GobEncode()
		parseMsg(buf)
	}

	// Message Ack
	{

		ack := msgc.QfxMsgAck{}
		m := msgc.MsgWrapper{MsgId: 3, NeedAck: 1, MsgType: msgc.QFX_MSG_ACK, Data: ack}

		buf, _ := m.GobEncode()
		parseMsg(buf)
	}

}

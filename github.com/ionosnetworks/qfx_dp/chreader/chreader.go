package chreader

import (
	"fmt"
	"io"

	"github.com/ionosnetworks/qfx_cmn/msgsvc/msgcli"
)

type Chreader chan msgcli.MsgPkt

func (c Chreader) Read(b []byte) (int, error) {
	p := <-c
	fmt.Println("data len: ", len(p.PayLoad), "source", p.Source)
	copy(b, p.PayLoad)
	return len(p.PayLoad), nil
}

type MsgWriter struct {
	p *msgcli.MsgXfer
}

func NewMsgWriter(m *msgcli.MsgXfer) io.Writer {
	return &MsgWriter{p: m}
}

func (m *MsgWriter) Write(b []byte) (int, error) {
	m.p.Write(b)
	return len(b), nil
}

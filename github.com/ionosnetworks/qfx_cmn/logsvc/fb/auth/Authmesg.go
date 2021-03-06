// automatically generated by the FlatBuffers compiler, do not modify

package auth

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type Authmesg struct {
	_tab flatbuffers.Table
}

func GetRootAsAuthmesg(buf []byte, offset flatbuffers.UOffsetT) *Authmesg {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &Authmesg{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *Authmesg) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Authmesg) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *Authmesg) NodeID() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Authmesg) NodeName() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Authmesg) Secret() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Authmesg) AccessKey() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func AuthmesgStart(builder *flatbuffers.Builder) {
	builder.StartObject(4)
}
func AuthmesgAddNodeID(builder *flatbuffers.Builder, nodeID flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(nodeID), 0)
}
func AuthmesgAddNodeName(builder *flatbuffers.Builder, nodeName flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(nodeName), 0)
}
func AuthmesgAddSecret(builder *flatbuffers.Builder, secret flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(2, flatbuffers.UOffsetT(secret), 0)
}
func AuthmesgAddAccessKey(builder *flatbuffers.Builder, accessKey flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(3, flatbuffers.UOffsetT(accessKey), 0)
}
func AuthmesgEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}

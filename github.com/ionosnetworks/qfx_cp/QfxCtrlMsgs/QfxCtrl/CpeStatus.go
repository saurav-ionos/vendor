// automatically generated by the FlatBuffers compiler, do not modify

package QfxCtrl

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type CpeStatus struct {
	_tab flatbuffers.Table
}

func GetRootAsCpeStatus(buf []byte, offset flatbuffers.UOffsetT) *CpeStatus {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &CpeStatus{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *CpeStatus) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *CpeStatus) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *CpeStatus) CpeId() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *CpeStatus) Status() int32 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt32(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CpeStatus) MutateStatus(n int32) bool {
	return rcv._tab.MutateInt32Slot(6, n)
}

func CpeStatusStart(builder *flatbuffers.Builder) {
	builder.StartObject(2)
}
func CpeStatusAddCpeId(builder *flatbuffers.Builder, cpeId flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(cpeId), 0)
}
func CpeStatusAddStatus(builder *flatbuffers.Builder, status int32) {
	builder.PrependInt32Slot(1, status, 0)
}
func CpeStatusEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
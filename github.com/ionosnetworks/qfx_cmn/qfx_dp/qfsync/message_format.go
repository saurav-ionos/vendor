//Message format between dataplane peers
package qfsync

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"

	"github.com/ionosnetworks/qfx_dp/infra"
)

const (
	MsgIDSyncHeartBeat int32 = iota
	MsgIDDestSyncInit
	MsgIDFileOrderMapping
	MsgIDInitialFileMetaComplete
	MsgIDBatchMetaDataFromDest
	MsgIDChunkAck
	MsgIDSyncStartFromDest
	MsgIDSyncEndFromSrc
	MsgIDBatchMetaEndFromDest
	MsgIDBatchMetaCompleteFromSrc
	MsgIDBatchEndFromSrc
)

type SyncHeartBeat struct {
	SyncID uint32
	CsID   infra.CsID
}

type DestSyncInit struct {
	SyncID  uint32
	CsID    infra.CsID
	LastIdx int64
}

type FileOrderMapping struct {
	SyncID     uint32
	StartIndex int64
	EndIndex   int64
	FileList   []string
	last       bool
}

type InitialFileMetaComplete struct {
	SyncID  uint32
	LastIdx int64
}

type BatchMetaFromDest struct {
	SyncID          uint32
	CsID            infra.CsID
	Findex          int64
	BlockSize       int64
	StartFileOffset int64
	FileEnd         bool
	Cksum           [][csumSize]byte
}

type BatchEndFromSrc struct {
	SyncID uint32
	CsID   infra.CsID
	Findex int64
	Size   int64
}

type SyncStartFromDest struct {
	SyncID   uint32
	CsID     infra.CsID
	SyncOpID uint32
}

type BatchMetaEndFromDest struct {
	SyncID   uint32
	CsID     infra.CsID
	SyncOpID uint32
}

type SyncEndFromSrc struct {
	SyncID   uint32
	SyncOpID uint32
}

type ChunkAck struct {
	SyncID uint32
	UUID   infra.UUID
	CsID   infra.CsID
}

type MsgWrapper struct {
	MsgID int32
	Data  interface{}
}

func (m MsgWrapper) Encode() ([]byte, error) {

	var w bytes.Buffer
	encoder := gob.NewEncoder(&w)
	err := encoder.Encode(m)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

//Custom encoder for MsgWrapper
func (m MsgWrapper) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	err := binary.Write(w, binary.LittleEndian, &m.MsgID)
	if err != nil {
		return nil, err
	}
	encoder := gob.NewEncoder(w)
	switch m.MsgID {
	case MsgIDSyncHeartBeat:
		hb := m.Data.(SyncHeartBeat)
		err = encoder.Encode(hb)
	case MsgIDDestSyncInit:
		f := m.Data.(DestSyncInit)
		err = encoder.Encode(f)
	case MsgIDFileOrderMapping:
		f := m.Data.(FileOrderMapping)
		err = encoder.Encode(f)
	case MsgIDInitialFileMetaComplete:
		f := m.Data.(InitialFileMetaComplete)
		err = encoder.Encode(f)
	case MsgIDBatchMetaDataFromDest:
		f := m.Data.(BatchMetaFromDest)
		err = encoder.Encode(f)
	case MsgIDChunkAck:
		f := m.Data.(ChunkAck)
		err = encoder.Encode(f)
	case MsgIDSyncStartFromDest:
		f := m.Data.(SyncStartFromDest)
		err = encoder.Encode(f)
	case MsgIDBatchMetaEndFromDest:
		f := m.Data.(BatchMetaEndFromDest)
		err = encoder.Encode(f)
	case MsgIDSyncEndFromSrc:
		f := m.Data.(SyncEndFromSrc)
		err = encoder.Encode(f)
	case MsgIDBatchEndFromSrc:
		f := m.Data.(BatchEndFromSrc)
		err = encoder.Encode(f)
	}
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

//Custom decoder for MsgWrapper
func (m *MsgWrapper) GobDecode(buf []byte) error {
	var err error
	//w := bytes.NewBuffer(buf[0:4])
	w := bytes.NewBuffer(buf)
	err = binary.Read(w, binary.LittleEndian, &m.MsgID)
	if err != nil {
		logger.DebugS("test-ctx", err.Error())
		return err
	}
	decoder := gob.NewDecoder(w)
	switch m.MsgID {
	case MsgIDSyncHeartBeat:
		hb := new(SyncHeartBeat)
		err = decoder.Decode(hb)
		m.Data = hb
	case MsgIDDestSyncInit:
		di := new(DestSyncInit)
		err = decoder.Decode(di)
		m.Data = di
	case MsgIDFileOrderMapping:
		f := new(FileOrderMapping)
		err = decoder.Decode(f)
		m.Data = f
	case MsgIDInitialFileMetaComplete:
		f := new(InitialFileMetaComplete)
		err = decoder.Decode(f)
		m.Data = f
	case MsgIDBatchMetaDataFromDest:
		f := new(BatchMetaFromDest)
		err = decoder.Decode(f)
		m.Data = f
	case MsgIDChunkAck:
		f := new(ChunkAck)
		err = decoder.Decode(f)
		m.Data = f
	case MsgIDSyncStartFromDest:
		f := new(SyncStartFromDest)
		err = decoder.Decode(f)
		m.Data = f
	case MsgIDBatchEndFromSrc:
		f := new(BatchEndFromSrc)
		err = decoder.Decode(f)
		m.Data = f
	}
	return err
}

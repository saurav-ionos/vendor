package qfsync

import (
	"encoding/gob"
	"fmt"
	"io"
	"sync"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_cmn/msgsvc/msgcli"
)

type Consumer interface {
	Consume(buf []byte)
}

var consumerTable struct {
	m map[string]Consumer
	sync.Mutex
}

func RegisterConsumer(uuid string, c Consumer) {
	fmt.Println("register consumer ", uuid, c)
	consumerTable.Lock()
	defer consumerTable.Unlock()
	consumerTable.m[uuid] = c
	logger.Debug("transport", "Consumer registered for", blog.Fields{
		"uuid": uuid})
	fmt.Println("register consumer done", uuid, c)
}

func processIncomingMsgs(ch chan msgcli.MsgPkt) error {

	for x := range ch {
		//logger.Debug("test-sync", "packet received", blog.Fields{
		//	"src":    x.Source,
		//	"length": len(x.PayLoad)})
		consumerTable.Lock()
		consumer, ok := consumerTable.m[x.Source]
		consumerTable.Unlock()
		if ok {
			consumer.Consume(x.PayLoad)
		} else {
			logger.ErrS("test-sync", "consumer not found")
		}
	}
	return nil
}

func PrepareListener(r io.Reader) {
	fmt.Println(r)
	for {
		dec := gob.NewDecoder(r)
		m := new(MsgWrapper)
		err := dec.Decode(&m)
		if err != nil {
			logger.ErrS("test-sync",
				"could nor decode "+err.Error())
			continue
		}
		switch m.MsgID {
		case MsgIDSyncHeartBeat:
			hb := (*SyncHeartBeat)(m.Data.(*SyncHeartBeat))
			//logger.Debug("test-sync",
			//	"hb received from", blog.Fields{
			//		"csid": hb.CsID.String()})
			updateOnlineStatus(hb)
		case MsgIDDestSyncInit:
			di := (*DestSyncInit)(m.Data.(*DestSyncInit))
			logger.Debug("test-sync",
				"dest sync init received", blog.Fields{
					"csid":      di.CsID.String(),
					"lastindex": di.LastIdx})
			handleDestInit(di)
		case MsgIDFileOrderMapping:
			f := (*FileOrderMapping)(m.Data.(*FileOrderMapping))
			updateFileMappingForSync(f)
		case MsgIDBatchMetaDataFromDest:
			f := (*BatchMetaFromDest)(m.Data.(*BatchMetaFromDest))
			handleBatchMetaFromDest(f)
		case MsgIDChunkAck:
			fmt.Printf("Ack msg %v\n", m)
			ack := (*ChunkAck)(m.Data.(*ChunkAck))
			handleChunkAck(ack)
		case MsgIDSyncStartFromDest:
			//f := (*SyncStartFromDest)(m.Data.(*SyncStartFromDest))
			//handleSyncStartFromDest(f)
		case MsgIDBatchEndFromSrc:
			f := (*BatchEndFromSrc)(m.Data.(*BatchEndFromSrc))
			handleBatchEndFromSrc(f)

		}
	}
}

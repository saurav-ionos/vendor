// This file contains destination specific sync code
package qfsync

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/infra"
	"github.com/pkg/errors"
)

var batchNo int32 = 1
var dstCtx string = "SYNC_DEST"
var batchesb4DestShudWait int = 100
var numAllowedPending int = 10

// A destinations view of itself
type dstdst struct {
	sroot       string // root path for the sync
	forder      FileOrder
	src         *cloudSeed
	syncReqChan chan struct {
		ei int64
		si int64
	}
	batchChannel chan struct{}
}

func (dst *dstdst) init() error {
	return nil
}

func (d *dstdst) sendChunkAck(w io.Writer, syncid uint32, uuid infra.UUID) {

	logger.Debug("test-sync",
		"chunk ack for", blog.Fields{"syncid": syncid})
	ack := ChunkAck{
		SyncID: syncid,
		UUID:   uuid,
		CsID:   infra.GetLocalCsID(),
	}
	msg := MsgWrapper{
		MsgID: MsgIDChunkAck,
		Data:  ack,
	}
	logger.Debug("test-sync", "sending acknowledgement for",
		blog.Fields{
			"uuid": ack.UUID.String()})
	b, err := msg.Encode()
	if err != nil {
		logger.ErrS("test-sync", "could not encode"+err.Error())
		return
	}
	err = d.src.SendMsg(b)
	if err != nil {
		logger.ErrS("test-sync",
			"could not send acknowledgement"+err.Error())
		return
	}
}

func (d *dstdst) startQfsDest(syncID uint32) error {
	var err error
	// send the last index to the source
	forderPrefix := fmt.Sprintf("/usr/local/ica/IONOS-DISK/%d", syncID)
	d.forder, err = buildNewFileOrder(d.sroot, DefBatchSize, forderPrefix,
		nil)
	// trigger the sync thread
	if err != nil {
		return errors.Wrap(err, "build file order error")
	}
	d.keepSyncing(syncID)

	return nil
}

func (d *dstdst) keepSyncing(syncID uint32) {

	go d.doFullSync(syncID)
	for {
		<-time.After(time.Second * 120)
		d.syncReqChan <- struct {
			ei int64
			si int64
		}{ei: d.forder.GetLastIdx(), si: 1}

	}
}

func (d *dstdst) doFullSync(syncID uint32) error {

	for x := range d.syncReqChan {
		ebm := BatchMetaFromDest{
			SyncID:    syncID,
			CsID:      localCsID,
			BlockSize: DefSyncBlockSize,
		}

		for i := x.ei; i >= x.si; i-- {
			var fileOffset int64 = 0
			<-d.batchChannel
			fpath, err := d.forder.GetFileNameAtIdx(i)
			if err != nil {
				return errors.Wrap(err, "error getting file idx")
			}
			fi, err := os.Stat(fpath)
			if err != nil {
				// Need to fetch file from source
				ebm.Findex = i
				msg := MsgWrapper{
					MsgID: MsgIDBatchMetaDataFromDest,
					Data:  ebm,
				}
				if err = d.sendMsgToSrc(&msg); err != nil {
					logger.ErrS(dstCtx, err.Error())
					return err
				}
				continue
			}
			// File exists. Read the file in batches and send
			// meta data to the source
			fh, err := os.Open(fpath)
			if err != nil {
				logger.ErrS(dstCtx, err.Error())
				continue
			}
			cdata := make([]byte, DefSyncBatchDataSize)
			for {
				nbytes, err := fh.Read(cdata)
				if nbytes == 0 && err != nil {
					fh.Close()
					break
				}
				data := cdata[0:nbytes]
				result := doBatchCksum(int(DefSyncBlockSize), data)
				bm := BatchMetaFromDest{
					SyncID:          syncID,
					CsID:            localCsID,
					Findex:          i,
					BlockSize:       DefSyncBlockSize,
					StartFileOffset: fileOffset,
					Cksum:           result,
				}
				fileOffset += int64(nbytes)
				if fileOffset == fi.Size() {
					bm.FileEnd = true
				}
				msg := MsgWrapper{
					MsgID: MsgIDBatchMetaDataFromDest,
					Data:  bm,
				}
				if err = d.sendMsgToSrc(&msg); err != nil {
					logger.ErrS(dstCtx, err.Error())
				}
			}
		}
	}
	return nil
}

func (d *dstdst) sendMsgToSrc(msg *MsgWrapper) error {
	msgBuf, err := msg.Encode()
	if err != nil {
		logger.ErrS("qfx_dp-dest", "msgbuf encode error "+err.Error())
		return err
	}
	err = d.src.SendMsg(msgBuf)
	if err != nil {
		logger.ErrS("qfx_dp-dest",
			"error sending metadata info"+err.Error())
		return err
	}
	return nil
}

package qfsync

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/infra"
	"github.com/ionosnetworks/qfx_dp/pktq"
	"github.com/pkg/errors"
)

const (
	heartBeatInterval = time.Second * time.Duration(5)
	disableCsTimeout  = 3 * heartBeatInterval
	srcCtx            = "SYNC_SRC"
)

// The states of the involved cloud seed devices
const (
	REQUEST = 1
	ACK     = 0
)

// A source's view of the destination
type sdst struct {
	cs           *cloudSeed
	lastIdx      int64
	syncOpID     uint32
	chunkMap     map[infra.UUID]bool
	chunkMapLock *sync.Mutex
	sync.Mutex
}

type batchDesc struct {
	findex      int64
	startOffset int64
}

type batchCksum struct {
	csid  infra.CsID
	cksum [][csumSize]byte
}

// A source's view of itself
type srcsrc struct {
	sroot         string //root path of sync
	forder        FileOrder
	dsts          map[infra.CsID]*sdst
	batchNotifyCh chan int32
	bmCollator    struct {
		m  *map[batchDesc][]*BatchMetaFromDest
		m1 map[batchDesc][]*BatchMetaFromDest
		m2 map[batchDesc][]*BatchMetaFromDest
		sync.Mutex
	}
	syncReqChan chan struct{}
}

type ChunkCreateReq struct {
	Findex      int64
	Fname       string
	StartOffset int64
	EndOffset   int64
}

type Ack struct {
	D infra.CsID
	U infra.UUID
}

type ChunkReqSet struct {
	S uint32
	M map[ChunkCreateReq][]infra.CsID
	C chan int32
}

func (src *srcsrc) init() error {
	return nil
}

type checkField func(d *sdst) bool

func (src *srcsrc) doWaitForCs(timeout time.Duration,
	c checkField) ([]*sdst, bool) {
	localTimeOut := time.Second * 2
waitForCs:
	for {
		allup := true
		var activeList []*sdst
		select {
		case <-time.After(timeout):
			for _, x := range src.dsts {
				x.Lock()
				state := c(x)
				logger.Info("qfx_dp-src", "dest state", blog.Fields{
					"csid":  x.cs.csid.String(),
					"state": state})
				if state {
					activeList = append(activeList, x)
				} else {
					allup = false
				}
				x.Unlock()
			}
			return activeList, allup
		case <-time.After(localTimeOut):
			for _, x := range src.dsts {
				fmt.Println("status", c(x))
				x.Lock()
				state := c(x)
				logger.Info("qfx_dp-src", "dest state", blog.Fields{
					"csid":  x.cs.csid.String(),
					"state": state})
				if !state {
					x.Unlock()
					continue waitForCs
				} else {
					activeList = append(activeList, x)
				}
				x.Unlock()
			}
			return activeList, true

		}
	}
}

func (src *srcsrc) doSync(syncID uint32) error {
	// Wait for all the destination CPEs to come online
	logger.InfoS("qfx_dp-src", "waiting for dest cloud seeds to be online")

	_, ok := src.doWaitForCs(time.Minute*5, func(x *sdst) bool {
		return x.cs.Alive()
	})
	if ok {
		logger.InfoS("qfx_dp-src",
			"all dest cloud seed devices up and online")
	}
	src.initDest()
	_, err := src.buildFileOrder(syncID)

	if err != nil {
		return errors.Wrap(err, "file order failed")
	}
	go func() {

		for {
			time.Sleep(time.Minute * 2)
			_, err = src.forder.Order()
			if err != nil {
				logger.ErrS("qfx_dp-src",
					"error in generating file order")
				continue
			}
		}
	}()

	src.doWaitForMetaAndSync(syncID)

	return nil
}

func (src *srcsrc) buildFileOrder(syncID uint32) (int64, error) {
	var sroot string
	forderPrefix := fmt.Sprintf("/usr/local/ica/IONOS-DISK/%d",
		syncID)
	sroot = src.sroot
	logger.DebugS("qfx_dp-src", "building new file order")
	fileOrder, err := buildNewFileOrder(sroot, DefBatchSize,
		forderPrefix, func(f FileOrder, si, ei int64) error {
			logger.Debug("qfx_dp-src", "callback called",
				blog.Fields{
					"si": si,
					"ei": ei})
			return src.sendFileOrder(syncID, si, ei)
		})
	src.forder = fileOrder
	lastidx, err := fileOrder.Order()
	logger.Debug("qfx_dp-src", "done building file order",
		blog.Fields{"lastidx": lastidx})
	return lastidx, err
}

func (src *srcsrc) sendFileOrder(syncID uint32, startIndex,
	lastIndex int64) error {
	var err error
	//find the minimum index from which data is
	// to be sent
	// start sending from the last idx
	fOrderMap := FileOrderMapping{
		SyncID:     syncID,
		StartIndex: startIndex,
		EndIndex:   lastIndex,
	}
	fOrderMap.FileList, err =
		src.forder.GetFileOrder(fOrderMap.StartIndex,
			fOrderMap.EndIndex)
	if err != nil {
		return errors.Wrap(err, "build file order failed")
	}
	msg := MsgWrapper{
		MsgID: MsgIDFileOrderMapping,
		Data:  fOrderMap,
	}
	b, err := msg.Encode()
	if err != nil {
		return errors.Wrap(err, "encoding file order failed")
	}
	//	noerr, errs := src.bcastMsg(b)
	src.bcastMsg(b)
	/*
		if noerr {
			for _, err := range errs {
				logger.ErrS(srcCtx, "send file order "+err.Error())
			}
		}*/
	return nil
}

func (src *srcsrc) doWaitForMetaAndSync(syncID uint32) {
	for {
		select {
		case <-time.After(time.Second * 60):
			//src.calculateCksumAndFindDiff(syncID)
			select {
			case src.syncReqChan <- struct{}{}:
			default:
			}
		case <-src.syncReqChan:
			src.calculateCksumAndFindDiff(syncID)
		}
	}
}

func (src *srcsrc) calculateCksumAndFindDiff(syncID uint32) error {
	var m *map[batchDesc][]*BatchMetaFromDest
	src.bmCollator.Lock()
	m = src.bmCollator.m
	if len(*m) > 0 {
		if m == &src.bmCollator.m1 {
			src.bmCollator.m = &src.bmCollator.m2
		} else {
			src.bmCollator.m = &src.bmCollator.m1
		}
	} else {
		src.bmCollator.Unlock()
		logger.Info(srcCtx, "No batches available to compute diff",
			blog.Fields{"syncid": syncID})
		return nil
	}
	src.bmCollator.Unlock()
	for desc, cksums := range *m {
		fpath, err := src.forder.GetFileNameAtIdx(desc.findex)
		if err != nil {
			logger.ErrS(srcCtx, err.Error())
			continue
		}
		crm := make(map[ChunkCreateReq][]infra.CsID)
		dList, err := generateChunkReq(fpath, cksums, crm)
		if err != nil {
			logger.ErrS(srcCtx, err.Error())
		}
		go func(findex int64) {
			if len(crm) > 0 {
				batchChunkReqAndSend(syncID, crm)
			}
			be := BatchEndFromSrc{
				SyncID: syncID,
				CsID:   localCsID,
				Findex: findex,
				Size:   -1,
			}

			if errors.Cause(err) == io.EOF {
				// Get the file size
				fi, err := os.Stat(fpath)
				if err != nil {
					logger.ErrS(srcCtx, "error in stat")
					return
				}
				be.Size = fi.Size()
			}

			msg := MsgWrapper{
				MsgID: MsgIDBatchEndFromSrc,
				Data:  be,
			}
			b, err := msg.Encode()
			if err != nil {
				logger.Err(srcCtx, "gob error",
					blog.Fields{"err": err.Error()})
				return
			}
			src.mcastMsg(b, dList)

		}(desc.findex)
		delete(*m, desc)
	}
	logger.Debug(srcCtx, "length of map at end of batch",
		blog.Fields{
			"len": len(*m),
		})
	return nil
}

func generateChunkReq(fpath string,
	bml []*BatchMetaFromDest,
	crm map[ChunkCreateReq][]infra.CsID) ([]infra.CsID, error) {

	var dList []infra.CsID
	fi, err := os.Stat(fpath)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: stat failed", fpath)
	}
	fh, err := os.Open(fpath)
	if err != nil {
		return nil, errors.Wrapf(err, "%s : file open failed", fpath)
	}
	defer fh.Close()
	buf := make([]byte, DefSyncBatchDataSize)
	nbytes, err := fh.ReadAt(buf, bml[0].StartFileOffset)
	if nbytes == 0 && err != nil {
		fh.Close()
		return nil, errors.Wrapf(err, "%s file read error", fpath)
	}
	buffer := buf[0:nbytes]
	result := doBatchCksum(int(DefSyncBlockSize), buffer)
	for _, dest := range bml {
		// fmt.Printf("for dest : %s", dest.CsID)
		for i, r := range result {
			if dest.Cksum == nil ||
				(i < len(dest.Cksum) && r != dest.Cksum[i]) {
				offset := dest.StartFileOffset +
					DefSyncBlockSize*int64(i)
				endOffset := offset + DefSyncBlockSize
				if endOffset > fi.Size() {
					endOffset = fi.Size()
				}

				logger.Debug(srcCtx, "diff!!", blog.Fields{
					"csid":   dest.CsID.String(),
					"offset": offset,
					"fname":  fpath,
				})
				cr := ChunkCreateReq{
					Findex:      dest.Findex,
					Fname:       fpath,
					StartOffset: offset,
					EndOffset:   endOffset,
				}
				crm[cr] = append(crm[cr], dest.CsID)
			}
		}
		if dest.FileEnd {
			// Send a request to create chunks for
			// rest of the file
			lastOffset := dest.StartFileOffset +
				int64(len(dest.Cksum))*DefSyncBlockSize
			if lastOffset < fi.Size() {
				cr := ChunkCreateReq{
					Findex:      dest.Findex,
					Fname:       fpath,
					StartOffset: lastOffset,
					EndOffset:   fi.Size(),
				}
				crm[cr] = append(crm[cr], dest.CsID)
			}

		}
		dList = append(dList, dest.CsID)
	}
	return dList, err
}

func batchChunkReqAndSend(syncID uint32,
	m map[ChunkCreateReq][]infra.CsID) {
	crs := ChunkReqSet{
		S: syncID,
		M: m,
		C: make(chan int32),
	}

	qfsToPipelineCh <- crs
	<-crs.C
	logger.DebugS(srcCtx, "received map ack from dp")
}

func (src *srcsrc) initDest() {
	for _, d := range src.dsts {
		d.Lock()
		d.chunkMapLock = new(sync.Mutex)
		d.chunkMap = make(map[infra.UUID]bool, 1024)
		d.Unlock()
	}

}

func (src *srcsrc) handleBatchMeta(bm *BatchMetaFromDest) {
	fname, err := src.forder.GetFileNameAtIdx(bm.Findex)
	logger.Debug(srcCtx, "meta received from dest",
		blog.Fields{
			"file index": bm.Findex,
			"file":       fname,
			"index":      bm.StartFileOffset,
			"csid":       bm.CsID.String(),
		})
	if err != nil {
		return
	}
	fi, err := os.Stat(fname)
	if err != nil {
		// Source does not have a file ignore maadi!
		return
	}

	if bm.StartFileOffset >= fi.Size() {
		logger.Debug(srcCtx, "startoffset > file size ignore meta",
			blog.Fields{
				"fname":  fname,
				"findex": bm.Findex,
			})
		return
	}
	b := batchDesc{
		findex:      bm.Findex,
		startOffset: bm.StartFileOffset,
	}
	src.bmCollator.Lock()
	m := src.bmCollator.m
	shudTrigger := true
	(*m)[b] = append((*m)[b], bm)
	for _, d := range *m {
		if len(d) <= 1 {
			shudTrigger = false
		}
	}
	src.bmCollator.Unlock()
	if shudTrigger {
		src.syncReqChan <- struct{}{}
		logger.DebugS(srcCtx, "trigger from handleBatchMeta")
	}

}

func (src *srcsrc) handleChunkAck(ack *ChunkAck) {
	logger.Debug("qfx_dp-src", "Chunk Ack received", blog.Fields{
		"csid":   ack.CsID.String(),
		"uuid":   ack.UUID.String(),
		"syncID": ack.SyncID,
	})
	d, ok := src.dsts[ack.CsID]
	if ok {
		d.Lock()

		d.chunkMapLock.Lock()
		_, ok := d.chunkMap[ack.UUID]
		if ok {
			delete(d.chunkMap, ack.UUID)
			pqChan := pktq.GetChan()
			if pqChan != nil {
				pqChan <- ack.UUID
			}
		}
		d.chunkMapLock.Unlock()
		d.Unlock()
	} else {
		logger.Err("qfx_dp-src", "destination not found", blog.Fields{
			"csid": ack.CsID.String()})
	}
}

func (src *srcsrc) mcastMsg(msg []byte,
	dest []infra.CsID) (bool, map[infra.CsID]error) {

	errMap := make(map[infra.CsID]error)
	noerr := true
	if dest != nil {
		for _, d := range dest {
			errMap[d] = src.dsts[d].cs.SendMsg(msg)
			if errMap[d] != nil {
				noerr = true
			}
		}
	}
	return noerr, errMap
}

func (src *srcsrc) bcastMsg(msg []byte) (bool, map[infra.CsID]error) {
	errMap := make(map[infra.CsID]error)
	noerr := true
	for _, d := range src.dsts {
		errMap[d.cs.csid] = d.cs.SendMsg(msg)
		if errMap[d.cs.csid] != nil {
			noerr = false
		}
	}
	return noerr, errMap
}

// Helper functions

func handleDestInit(di *DestSyncInit) error {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[di.SyncID]
	syncRelTable.Unlock()
	if !ok {
		logger.Warn("qfx_dp", "sync relationship not found",
			blog.Fields{"syncid": s.SyncID})
		return errors.New("syncid not found")
	}

	// We have a valid sync relationship. check if i am the source
	if !s.source {
		logger.Warn("qfx_dp", "dest init directed to dest",
			blog.Fields{"source csid": di.CsID.String()})
	}
	return nil
}

func handleBatchMetaFromDest(bm *BatchMetaFromDest) error {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[bm.SyncID]
	syncRelTable.Unlock()
	if !ok {
		logger.Warn("qfx_dp", "handle bm sync relationship not found",
			blog.Fields{"syncid": s.SyncID})
		return errors.New("syncid not found")
	}
	s.src.handleBatchMeta(bm)
	return nil
}

func handleChunkAck(ack *ChunkAck) error {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[ack.SyncID]
	syncRelTable.Unlock()
	if !ok {
		logger.Warn("qfx_dp", "handle ca sync relationship not found",
			blog.Fields{"syncid": s.SyncID})
		return errors.New("syncid not found")
	}
	logger.Info("qfx_dp", "Handling chunk ack for ",
		blog.Fields{"syncid": s.SyncID})
	s.src.handleChunkAck(ack)
	return nil
}

package qfsync

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_cmn/msgsvc/msgcli"
	"github.com/ionosnetworks/qfx_dp/infra"
)

// temp variables for testing
var tmpSrcCsID string = "08f5baea023d46da54e44a2c5aaad17f"
var tmpDstCsID1 string = "ded731565cef841830b3160d068cbb55"
var tmpDstCsID2 string = "d41d8cd98f00b204e9800998ecf8427e"

var syncRelTable struct {
	m map[uint32]*syncRel
	sync.Mutex
}

type QfSyncFlags uint32

type Syncer interface {
	Init() error
	Sync() error
	GetFileName(idx int64) (string, error)
}

// A syncRel struct represents
// a sync relationship
type syncRel struct {
	SyncID    uint32
	SyncType  QfSyncFlags
	Priority  uint32
	PhyIntf   string
	source    bool
	fileOrder FileOrder
	src       *srcsrc // available on the source
	dst       *dstdst // available on the destinations
}

var (
	DefBatchSize         int32 = 5
	DefSyncBatchDataSize int64 = 128 * 1024 * 1024 //128 MiB
	DefSyncBlockSize     int64 = 4 * 1024 * 1024   //4 MiB
	csHealthCheckTimeout       = time.Second * 10
)

var localCsID infra.CsID
var logger blog.Logger
var glMsgSrvr string = "192.168.1.141"
var accesskey string = "0123456789"
var secret string = "abcdefghijklmnopqrstuvwxyz"
var serverport string = "8080"
var glMsgCli *msgcli.MsgCli
var qfsToPipelineCh chan ChunkReqSet

func init() {

	// Initialize the logger
	// logFile, err := os.OpenFile("/var/log/ionos/qfsync.log",
	//		os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	//logFile, err := net.Dial("tcp", "127.0.0.1:2000")
	//if err != nil {
	//	panic(err)
	//}
	var err error
	logger = blog.New("127.0.0.1:2000", accesskey, secret)
	localCsID = infra.GetLocalCsID()
	syncRelTable.m = make(map[uint32]*syncRel)
	consumerTable.m = make(map[string]Consumer)
	logger.SetLevel(blog.Debug)

	//initialize global message server handlers
	glMsgCli, err = msgcli.New(glMsgSrvr, localCsID.String(), serverport,
		accesskey, secret, "", nil)
	if err != nil {
		panic(err)
	}
	//var ch chreader.Chreader = glMsgCli.MsgRecv
	go processIncomingMsgs(glMsgCli.MsgRecv)
	// go monitorRamdisk()
	//time.Sleep(time.Second * 15)

}

func Init(ch chan ChunkReqSet) {
	qfsToPipelineCh = ch

}

// Sync relation ship configuration APIs

func CreateSyncRel(syncID uint32) *syncRel {
	v := new(syncRel)
	v.SyncID = syncID
	syncRelTable.Lock()
	syncRelTable.m[syncID] = v
	syncRelTable.Unlock()
	return v

}

func (s *syncRel) SetSrc(csid string, srcdir string) {

	var src *cloudSeed
	var err error
	if localCsID.String() == csid {
		// I am the source
		s.source = true
		s.dst = nil
		s.src = new(srcsrc)
		s.src.sroot = srcdir
		s.src.dsts = make(map[infra.CsID]*sdst)
		s.src.syncReqChan = make(chan struct{}, 100)
		s.src.batchNotifyCh = make(chan int32)
		s.src.bmCollator.m1 = make(map[batchDesc][]*BatchMetaFromDest)
		s.src.bmCollator.m2 = make(map[batchDesc][]*BatchMetaFromDest)
		s.src.bmCollator.m = &s.src.bmCollator.m1
	} else {
		s.dst = new(dstdst)
		for {
			src, err = createCloudSeed(infra.DecodeStringToCsID(tmpSrcCsID))
			if err != nil {
				logger.Err("sync-rel", "Error creating cloud seed",
					blog.Fields{
						"csid": tmpSrcCsID,
					})
				time.Sleep(time.Second)
			} else {
				break
			}
		}
		s.dst.src = src
		s.dst.syncReqChan = make(chan struct {
			ei int64
			si int64
		}, numAllowedPending)
		s.dst.batchChannel = make(chan struct{}, batchesb4DestShudWait)
		for i := batchesb4DestShudWait; i > 0; i-- {
			s.dst.batchChannel <- struct{}{}

		}
	}

}

func (s *syncRel) SetDest(csid string, dstDir string) {

	var cs *cloudSeed
	var err error
	if s.source {
		dst := new(sdst)
		dd := infra.DecodeStringToCsID(csid)

		for {
			cs, err = createCloudSeed(dd)
			if err != nil {
				logger.Err("sync-rel", "Error creating cloud seed",
					blog.Fields{
						"csid": csid,
					})
				time.Sleep(time.Second)
			} else {
				fmt.Println("created dest1 cloud seed")
				logger.InfoS("sync-rel", "created cloud seed")
				break
			}
		}
		dst.cs = cs
		s.src.dsts[dd] = dst
	} else {
		if localCsID.String() == csid {
			s.dst.sroot = dstDir
		}
	}

}

func (s *syncRel) SetPrio(prio uint32) {
	s.Priority = prio
}

func (s *syncRel) Init() error {
	if s.source {
		return s.src.init()
	}
	return s.dst.init()
}

func GetSyncRel(syncID uint32) Syncer {
	syncRelTable.Lock()
	defer syncRelTable.Unlock()
	return syncRelTable.m[syncID]
}

func (s *syncRel) Sync() error {

	done := make(chan struct{})
	if s.source {
		s.src.doSync(s.SyncID)
	} else {
		//time.Sleep(time.Hour)
		s.dst.startQfsDest(s.SyncID)
	}
	close(done)
	return nil
}

func (s *syncRel) GetFileName(idx int64) (string, error) {
	var err error
	var fname string
	if s.source {
		fname, err = s.src.forder.GetFileNameAtIdx(idx)
	} else {
		fname, err = s.dst.forder.GetFileNameAtIdx(idx)
	}
	return fname, err

}

func updateFileMappingForSync(f *FileOrderMapping) error {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[f.SyncID]
	syncRelTable.Unlock()
	// Check if there is a file order mapping
	// object present
	if ok == false {
		logger.Debug("test-ctx", "No sync relation found: ",
			blog.Fields{"syncid": f.SyncID})
		return errors.New("No such sync relation")
	}
	err := s.dst.forder.SetFileOrder(f.StartIndex, f.EndIndex, f.FileList)
	if err != nil {
		logger.ErrS("qfx_dp-dst", err.Error())
		return err
	}
	fmt.Println("received file order from source", f.EndIndex, f.StartIndex)
	s.dst.syncReqChan <- struct {
		ei int64
		si int64
	}{ei: f.EndIndex, si: f.StartIndex}
	logger.Debug("test-ctx", "update file mapping: ",
		blog.Fields{"startIndex": f.StartIndex, "endIndex": f.EndIndex})
	return nil
}

func UpdateOutstandingChunkMap(syncID uint32,
	chunkID infra.UUID,
	dest []infra.CsID) {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[syncID]
	syncRelTable.Unlock()
	if !ok {
		// logger.Err("test-sync", "sync not found", blog.Fields{"syncIDi": syncID})
		fmt.Println("SyncID not found", syncID)
		return

	}
	for _, dst := range dest {
		for _, cpe := range s.src.dsts {
			if cpe.cs.csid == dst {
				// Create an entry
				cpe.chunkMapLock.Lock()
				cpe.chunkMap[chunkID] = false
				cpe.chunkMapLock.Unlock()
				break

			}

		}

	}

}
func SendAck(syncID uint32, chunkID infra.UUID) {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[syncID]
	syncRelTable.Unlock()
	if !ok {
		// logger.Err("test-sync", "sync not found", blog.Fields{"syncIDi": syncID})
		fmt.Println("SyncID not found", syncID)
		return
	}
	d := s.dst
	d.sendChunkAck(d.src.mq, syncID, chunkID)
}

func handleBatchEndFromSrc(b *BatchEndFromSrc) {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[b.SyncID]
	syncRelTable.Unlock()

	if ok {
		select {
		case s.dst.batchChannel <- struct{}{}:
			logger.DebugS(dstCtx, "free token for meta end from src")
		default:
			logger.Warn("sync-ctx", "could not free meta token for",
				blog.Fields{
					"syncid": b.SyncID,
				})
		}
		if b.Size > 0 {
			// A file has ended. I have the size of the file.
			// will truncate it
			fname, err := s.dst.forder.GetFileNameAtIdx(b.Findex)
			if err != nil {
				logger.ErrS(dstCtx, err.Error())
				return
			}
			err = os.Truncate(fname, b.Size)
			if err != nil {
				logger.ErrS(dstCtx, "error truncating file")
				return
			}
			logger.Debug(dstCtx, "truncate file length to",
				blog.Fields{
					"fname": fname,
					"size":  b.Size,
				})
		}
	}

}

// Check if the destination in a given sync has any outstanding/ unacked chunks
/*
func isSyncComplete(syncID uint32, dest infra.CsID) (bool, error) {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[syncID]
	syncRelTable.Unlock()
	if !ok {
		logger.Warn("test-ctx", "check sync -- sync relationship not found",
			blog.Fields{"syncid": s.SyncID})
		return false, errors.New("syncid not found")
	}
	d, ok := s.src.dsts[dest]
	if ok {
		d.Lock()

		d.chunkMapLock.Lock()
		for k := range d.chunkMap {
			if d.chunkMap[k] == false {
				return false, nil
			}
		}
		d.chunkMapLock.Unlock()
		d.Unlock()
	}
	return true, nil
}
*/

/*
func handleSyncStartFromDest(f *SyncStartFromDest) {
	syncRelTable.Lock()
	s, ok := syncRelTable.m[f.SyncID]
	syncRelTable.Unlock()

	if !ok {
		logger.Err("qfx_src", "No sync relation found", blog.Fields{
			"syncid": f.SyncID})
	}

	fmt.Println("sync start from dest received", *f)

	dst := s.src.dsts[f.CsID]
	dst.Lock()
	dst.syncStarted = true
	dst.syncOpID = f.SyncOpID
	dst.Unlock()
}
*/

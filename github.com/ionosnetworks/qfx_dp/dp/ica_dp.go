package dp

import (
	"bytes"
	"crypto/aes"
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/infra"
	p "github.com/ionosnetworks/qfx_dp/pipeline"
	"github.com/ionosnetworks/qfx_dp/pktq"
	"github.com/ionosnetworks/qfx_dp/qfsync"
)

// Fields in header
const (
	HEAD_VER      = 1
	HEAD_LEN      = 8
	HEAD_HOP      = 1
	HEAD_SYNC     = 4
	HEAD_CID      = 16
	HEAD_PRIO     = 8
	HEAD_KEY_IND  = 8
	HEAD_KEY_LAST = 1
)

var ctx = "ICA-DP"

var log blog.Logger

const (
	TX_JOB int = iota
	RX_JOB
)

const (
	NUM_PRIO_LEVELS int = 2

	NUM_DISC_READ_EXEC_V1B_CONTEXT int = 1
	NUM_DISC_READ_EXEC_V1_CONTEXT  int = 1

	NUM_DISC_WRITE_EXEC_V1B_CONTEXT int = 1
	NUM_DISC_WRITE_EXEC_V1_CONTEXT  int = 1

	NUM_CHUNKSUM_EXEC_V1B_CONTEXT int = 1
	NUM_CHUNKSUM_EXEC_V1_CONTEXT  int = 1

	NUM_CHUNKSUM_EXEC_V2_CONTEXT    int = 1
	NUM_ENCRYPTION_EXEC_V1B_CONTEXT int = 1
	NUM_ENCRYPTION_EXEC_V1_CONTEXT  int = 1

	NUM_ENCRYPTION_EXEC_V2_CONTEXT int = 1
	NUM_FORWARD_EXEC_V1B_CONTEXT   int = 1
	NUM_FORWARD_EXEC_V1_CONTEXT    int = 1

	ICA_MAX_V1_CPE_PENDING_REQ = 2
	ICA_MAX_V2_CPE_PENDING_REQ = 70

	CPE_VERSION_1  = "1"
	CPE_VERSION_1b = "1b"
	CPE_VERSION_1f = "1f"
	CPE_VERSION_2  = "2"

	JOB_MEM_LOW  = 70
	JOB_MEM_HIGH = 50
)

// var dpcpConn net.Conn
var sitename string
var top *p.Topology
var currentCPEversion string
var NUM_ICA_RX_PENDING_REQUEST = int32(ICA_MAX_V2_CPE_PENDING_REQ)

// RespChan ... Channel to process response from the pipeline
var RespChan chan interface{}

type batchMap struct {
	m map[*BatchInfo]map[infra.UUID]struct{}
	sync.Mutex
}

func dumpIcaState(sig chan os.Signal) {
	for {
		signal := <-sig
		fname := fmt.Sprintf("/tmp/ica_stack_trace.%d",
			time.Now().Unix())
		f, err := os.OpenFile(fname,
			os.O_CREATE|os.O_WRONLY, 0777)
		if err == nil {
			fmt.Fprintln(f, "Recieved sinal", signal,
				"dumping stack trace and system variables")
			pprof.Lookup("goroutine").WriteTo(f, 1)
			fmt.Fprintln(f, "CPE ID", sitename)
			fmt.Fprintln(f, "cpe version", currentCPEversion)
			for i, x := range top.GetPendingRequests() {
				fmt.Fprintln(f, "Stage: ", i, "PendingReq", x)
			}
		} else {
			fmt.Println(err)
		}
		f.Close()
	}
}

// InitIcaDp ... Method to initialize the pipeline
func InitIcaDp(FwdToDpChan chan string,
	QfsCh chan qfsync.ChunkReqSet) {
	// log = logger
	log = blog.New("127.0.0.1:2000", "12", "12")
	log.SetLevel(blog.Debug)

	sig1 := make(chan os.Signal, 1)

	/* Install the signal handlers for debugging */
	signal.Notify(sig1, syscall.SIGUSR1)

	log.Info(ctx, "Initializing data plane", nil)
	fi, err := os.Open("/etc/ionos-cpeid.conf")
	defer fi.Close()
	if err != nil {
		fmt.Println("Fatal: CPE-ID conf not found")
		// os.Exit(-1)
	}
	cpeID := make([]byte, 32)
	_, err = fi.Read(cpeID)
	if err != nil {
		fmt.Println("Fatal: Could not read CPE-ID")
		//os.Exit(-1)
	}
	sitename = string(cpeID)
	log.Info(ctx, "", blog.Fields{"Sitename": sitename})

	/* See what version of CPE are we running on */
	fi, err = os.Open("/etc/ionos-cpe-version.conf")
	if err != nil {
		if os.IsNotExist(err) {
			currentCPEversion = "1"
		}
	}
	version := make([]byte, 2)
	_, err = fi.Read(version)
	if err != nil {
		currentCPEversion = "1"
		NUM_ICA_RX_PENDING_REQUEST = ICA_MAX_V1_CPE_PENDING_REQ
	} else {
		currentCPEversion = string(version)
		if string(version) == "1b" ||
			strings.Contains(string(version), "2") == true ||
			string(version) == "1f" {
			/* if tx win size is configured as 2 by default
			 * make it 9 for v2 cpes as default
			 */
		}
	}
	go createLFTTopology()
	RespChan = make(chan interface{}, 1024)

	go processPipelineResponse(RespChan)

	go processMsgFromSyncMgr(QfsCh)

	time.Sleep(10 * time.Second)
	if len(os.Args) > 1 {
		if os.Args[1] == "dest" {
			go monitorRamdisk()
		}
	}

	go dumpIcaState(sig1)
}

func monitorRamdisk() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	os.Mkdir("/mnt/ftp/1", 0777)
	err = watcher.Add("/mnt/ftp/1")
	if err != nil {
		panic(err)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				if filepath.Ext(event.Name) == ".partial" {
					fmt.Println("file created", event.Name)
					fpath := "STOR " + event.Name
					fmt.Println("Found path sending stor", fpath)
					handleStorMsg(fpath, top, true)

				}
			}

		case err := <-watcher.Errors:
			fmt.Println(err)
		}

	}

}

func handleStorMsg(storMsg string, top *p.Topology, updateTimer bool) {
	/* We have received a STOR message, we need to pass
	 * the chunk to layers in the pipeline
	 */
	log.Debug(ctx, "Recived", blog.Fields{"Msg": storMsg})

	req := new(RxJobReqResp)
	getChunkInfo(&req.cfo, storMsg)
	processReq := new(p.ProcessReqResp)
	processReq.Interests = []string{"chunkunsum", "stich"} //rxJobFlowEntry.iFlow
	processReq.CurrentInterest = 0
	processReq.Data = req
	processReq.MsgType = RX_JOB
	processReq.RespChan = RespChan
	log.Info(ctx, "Pushed for processing", blog.Fields{"Msg": storMsg})
	top.PushToInterestedChannels("chunkunsum", processReq)
}

func handleSyncMsg(syncMsg qfsync.ChunkReqSet) {
	for k, v := range syncMsg.M {
		var jobletMod JobletMod
		var joblet Joblet
		var msg BatchInfo
		msg.syncID = syncMsg.S
		msg.dest = v
		joblet.JobletFileName = k.Fname
		fi, err := os.Stat(joblet.JobletFileName)
		if err != nil {
			fmt.Println("Error on stat file", err.Error())
		}
		joblet.ActionType = "ADD"
		joblet.IsDir = fi.IsDir()
		joblet.FileSize = uint64(fi.Size())
		joblet.ModTime = strconv.FormatInt(fi.ModTime().Unix(), 10)
		joblet.Mod = nil
		joblet.JobletId = uint32(k.Findex)
		jobletMod.StartOffset = uint64(k.StartOffset)
		jobletMod.EndOffset = uint64(k.EndOffset)
		jobletMod.Size = uint64(jobletMod.EndOffset -
			jobletMod.StartOffset)
		joblet.Mod = append(joblet.Mod, jobletMod)
		msg.joblets = nil
		msg.joblets = append(msg.joblets, joblet)
		msg.bmc = syncMsg.C
		//TODO Batch the msgs before starting Xfer for smaller files
		go startBatchXfer(msg)
	}
}

func processMsgFromSyncMgr(ch chan qfsync.ChunkReqSet) {
	for {
		msg := <-ch
		// fmt.Printf("Sync Msg: %+v\n", msg)
		go handleSyncMsg(msg)
	}
}

func createLFTTopology() {
	discser1 := new(DiscSer)
	discser2 := new(DiscSer)
	forwarder := new(Forwarder)
	checksum1 := new(Checksum)
	checksum2 := new(Checksum)
	encryption := new(Encryption)
	decryption := new(Encryption)
	//var done chan bool
	var err error

	/* Initialize random number generator */
	rand.Seed(time.Now().UnixNano())

	/* Define the number of execution contexts according to the
	 * CPE version
	 */
	NUM_CHUNKSUM_EXEC_CONTEXT := NUM_CHUNKSUM_EXEC_V2_CONTEXT
	NUM_ENCRYPTION_EXEC_CONTEXT := NUM_ENCRYPTION_EXEC_V2_CONTEXT
	NUM_DISC_WRITE_EXEC_CONTEXT := NUM_DISC_WRITE_EXEC_V1B_CONTEXT
	NUM_DISC_READ_EXEC_CONTEXT := NUM_DISC_READ_EXEC_V1B_CONTEXT
	NUM_FORWARD_EXEC_CONTEXT := NUM_FORWARD_EXEC_V1B_CONTEXT

	/* Reduce the number of threads when running on single core
	 * machine
	 */
	if currentCPEversion == CPE_VERSION_1 {
		NUM_DISC_WRITE_EXEC_CONTEXT = NUM_DISC_WRITE_EXEC_V1_CONTEXT
		NUM_DISC_READ_EXEC_CONTEXT = NUM_DISC_READ_EXEC_V1_CONTEXT
		NUM_FORWARD_EXEC_CONTEXT = NUM_FORWARD_EXEC_V1_CONTEXT
		NUM_CHUNKSUM_EXEC_CONTEXT = NUM_CHUNKSUM_EXEC_V1_CONTEXT
		NUM_ENCRYPTION_EXEC_CONTEXT = NUM_ENCRYPTION_EXEC_V1_CONTEXT
		NUM_ICA_RX_PENDING_REQUEST = ICA_MAX_V1_CPE_PENDING_REQ
	} else if currentCPEversion == CPE_VERSION_1b ||
		currentCPEversion == CPE_VERSION_1f {
		NUM_DISC_WRITE_EXEC_CONTEXT = NUM_DISC_WRITE_EXEC_V1B_CONTEXT
		NUM_DISC_READ_EXEC_CONTEXT = NUM_DISC_READ_EXEC_V1B_CONTEXT
		NUM_FORWARD_EXEC_CONTEXT = NUM_FORWARD_EXEC_V1B_CONTEXT
		NUM_CHUNKSUM_EXEC_CONTEXT = NUM_CHUNKSUM_EXEC_V1B_CONTEXT
		NUM_ENCRYPTION_EXEC_CONTEXT = NUM_ENCRYPTION_EXEC_V1B_CONTEXT
		NUM_ICA_RX_PENDING_REQUEST = ICA_MAX_V1_CPE_PENDING_REQ
	}
	top, err = p.CreateTopology("ICA-DP", log)

	if err != nil {
		os.Exit(-1)
	}

	stage, err := top.AddStage("DiscWriter", discser1,
		NUM_DISC_WRITE_EXEC_CONTEXT, NUM_PRIO_LEVELS)

	if err != nil {
		fmt.Println("Fatal: Adding Disc Writer failed")
		os.Exit(1)
	}

	stage.AddInterest("stich")

	stage, err = top.AddStage("DiscReader", discser2,
		NUM_DISC_READ_EXEC_CONTEXT, NUM_PRIO_LEVELS)

	if err != nil {
		fmt.Println("Fatal: Adding Disc Reader failed")
		os.Exit(1)
	}
	stage.AddInterest("chunk")

	stage, err = top.AddStage("Forwarder", forwarder,
		NUM_FORWARD_EXEC_CONTEXT, NUM_PRIO_LEVELS)

	if err != nil {
		fmt.Println("Fatal: Adding Forwarder failed")
		os.Exit(1)
	}
	stage.AddInterest("forward")

	stage, err = top.AddStage("Checksum", checksum1,
		NUM_CHUNKSUM_EXEC_CONTEXT, NUM_PRIO_LEVELS)

	if err != nil {
		fmt.Println("Fatal: Adding Checksum stage failed")
		os.Exit(1)
	}
	stage.AddInterest("chunksum")

	stage, err = top.AddStage("CheckUnsum", checksum2,
		NUM_CHUNKSUM_EXEC_CONTEXT, NUM_PRIO_LEVELS)

	if err != nil {
		fmt.Println("Fatal: Adding CheckUnsum stage failed")
		os.Exit(1)
	}
	stage.AddInterest("chunkunsum")

	stage, err = top.AddStage("Encryption", encryption,
		NUM_ENCRYPTION_EXEC_CONTEXT, NUM_PRIO_LEVELS)

	if err != nil {
		fmt.Println("Fatal: Adding Encryption stage failed")
		os.Exit(1)
	}

	stage.AddInterest("encrypt")

	stage, err = top.AddStage("Decryption", decryption,
		NUM_ENCRYPTION_EXEC_CONTEXT, NUM_PRIO_LEVELS)

	if err != nil {
		fmt.Println("Fatal: Adding Decryption stage failed")
		os.Exit(1)
	}
	stage.AddInterest("decrypt")

	top.InitTopology()
	top.Start()

	log.Info(ctx, "All operational", nil)

}

type JobletMod struct {
	StartOffset uint64
	EndOffset   uint64
	// contribution of this joblet to the size of the job
	Size uint64
}

type Joblet struct {
	JobletId       uint32
	JobletFileName string
	IsDir          bool
	FileSize       uint64
	ModTime        string
	ActionType     string
	Corrupt        bool
	Rcvd           bool
	Mod            []JobletMod
}

type BatchInfo struct {
	syncID  uint32
	joblets []Joblet
	dest    []infra.CsID
	prio    int
	bmc     chan int32
}

type JobReq struct {
	// syncID          uint32
	Joblets         []Joblet
	dest            []infra.CsID
	chunkSize       uint64
	totalSize       uint64
	totalChunks     uint32
	headerSpace     uint64
	writeOffset     uint64
	buffer          []byte
	ChunkActualSize uint64
	chunkDir        string
	// prio            int
}

/* This is the order how an outgoing chunk is formed -
 * 1) Read Chunk
 * 2) Encrypt
 * 3) Check sum
 * 4) Forward
 */
func getHeaderSpace(dest []infra.CsID, iflow []string, chunksize uint64,
	blocksize uint64) (headerSpace uint64, writeOffset uint64) {
	//	log.Debug(ctx, "BlockSize=", blocksize)
	for _, x := range iflow {
		var headroom uint64
		if x == "encrypt" {
			headroom = top.GetStage(x).HeaderSpace()
			writeOffset += headroom
			rem := (headerSpace + chunksize) % blocksize
			if rem != 0 {
				/* This would be pad at the end */
				//		log.Debug(ctx, "Padding =", rem)
				headroom += rem
			}
			/* I need block size amount of storage
			 * for the IV */
			headroom += blocksize
			writeOffset += blocksize
		} else {
			headroom = top.GetStage(x).HeaderSpace()
			writeOffset += headroom
		}
		headerSpace += headroom
	}
	// accommodate the outer header {[]dest, prio, keyIndex, keyLast}
	buffer := &bytes.Buffer{}

	gob.NewEncoder(buffer).Encode(dest)
	byteSlice := buffer.Bytes()
	// fmt.Printf("%q\n", byteSlice)
	/*
	 * Outer Header Format (tentative)
	 * ver | len | hop | syncID |
	 * uuid | dest[] | prio | keyIndex | keyLast |
	 */
	length := uint64(HEAD_VER +
		HEAD_LEN +
		HEAD_HOP +
		HEAD_SYNC +
		HEAD_CID +
		len(byteSlice) +
		HEAD_PRIO +
		HEAD_KEY_IND +
		HEAD_KEY_LAST)

	// length := uint64(1 + 8 + 1 + 4 + 16 + len(byteSlice) + 8 + 8 + 1)
	headerSpace += length
	writeOffset += length
	fmt.Println("In getHeaderSpace: Header Space, writeOffset:",
		headerSpace, writeOffset)
	return headerSpace, writeOffset
}

func monitorBatchChan(inCh chan struct{}, count uint32, outCh chan int32) {
	//TODO bring in heartbeat here
	for {
		<-inCh
		count--
		if count == 0 {
			outCh <- 0
			return
		}
	}
}

func startBatchXfer(msg BatchInfo) {
	chunkSize := uint64(4 * 1024 * 1024) // 4MB
	iFlow := []string{"chunk", "chunksum", "forward"}
	var mIndex uint32
	var totalSize uint64
	for mIndex = 0; mIndex < uint32(len(msg.joblets)); mIndex++ {
		for k := 0; k < len(msg.joblets[mIndex].Mod); k++ {
			totalSize += msg.joblets[mIndex].Mod[k].Size
		}
	}
	totalChunks :=
		uint32(math.Ceil(float64(totalSize) /
			float64(chunkSize)))
	headerSpace, writeOffset :=
		getHeaderSpace(msg.dest, iFlow, (chunkSize),
			aes.BlockSize)

	fmt.Println("size and chunks: ", totalSize, totalChunks)

	var i uint32
	var uuid = make([]infra.UUID, totalChunks)

	for i = 0; i < totalChunks; i++ {
		uuid[i] = infra.GenUUID()
		// Update the chunk map of the destination
		qfsync.UpdateOutstandingChunkMap(msg.syncID,
			uuid[i],
			msg.dest)
	}
	batchChan := make(chan struct{})
	go monitorBatchChan(batchChan, totalChunks, msg.bmc)

	for i = 1; i <= (totalChunks); i++ {
		cReq := new(JobReq)
		cReq.Joblets = msg.joblets
		cReq.chunkDir = "/mnt/ftp/1"
		cReq.dest = msg.dest
		cReq.totalSize = totalSize
		cReq.totalChunks = totalChunks
		cReq.chunkSize = chunkSize
		cReq.headerSpace = headerSpace
		cReq.writeOffset = writeOffset
		req := new(p.ProcessReqResp)
		req.MsgType = TX_JOB
		req.SyncID = msg.syncID
		req.Prio = 0
		req.RespChan = RespChan
		req.Interests = iFlow
		req.CurrentInterest = 0
		req.Data = cReq
		req.ChunkNum = i
		req.UUID = uuid[i-1]

		pq := pktq.Get()
		numDest := uint32(len(cReq.dest))
		var payload []byte
		pq.Insert(payload, req.SyncID, req.UUID,
			numDest, batchChan)

		_ = top.PushToInterestedChannels(req.Interests[req.CurrentInterest],
			req)
	}
	return
}

func processPipelineResponse(ch chan interface{}) {
	for {
		inVal := <-ch
		resp := inVal.(p.PipelineResp)
		// fmt.Println("Received something", resp)
		if resp.MsgType == RX_JOB {
			if resp.Status == SUCCESS {
				log.Info(ctx, "Successfully stitched chunk ",
					blog.Fields{"UUID": resp.UUID.String()})
				fmt.Println("Successfully stitched chunk ",
					resp.UUID.String())
				rxJobReq := resp.Req.Data.(*RxJobReqResp)
				fmt.Println("Removing stitched chunk",
					rxJobReq.cfo.ChunkPath)
				err := os.Remove(rxJobReq.cfo.ChunkPath)
				if err != nil {
					log.Err(ctx, "Unable to remove file",
						blog.Fields{"path": rxJobReq.cfo.ChunkPath})
				}
				qfsync.SendAck(resp.SyncID, resp.UUID)
			}
		} else if resp.MsgType == TX_JOB {
			if resp.Status != SUCCESS {
				log.Info(ctx, "Recreating chunk ",
					blog.Fields{"UUID": resp.UUID.String()})
				fmt.Println("Recreating chunk ",
					resp.UUID.String())
				req := resp.Req
				req.CurrentInterest = 0
				txJobReq := resp.Req.Data.(*JobReq)
				txJobReq.headerSpace, txJobReq.writeOffset =
					getHeaderSpace(txJobReq.dest, req.Interests,
						(txJobReq.chunkSize),
						aes.BlockSize)
				txJobReq.buffer = nil //txJobReq.buffer[:0]
				req.Data = txJobReq
				_ = top.PushToInterestedChannels(
					req.Interests[req.CurrentInterest],
					req)
			} else {
				log.Info(ctx, "Chunk created successfully",
					blog.Fields{"UUID": resp.UUID.String()})
				fmt.Println("Chunk created successfully",
					resp.UUID.String())
				req := resp.Req.Data.(*JobReq)
				req = req
			}
		}
	}
}

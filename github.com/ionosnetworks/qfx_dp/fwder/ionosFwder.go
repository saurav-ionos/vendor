package fwder

import (
	"fmt"
	"launchpad.net/gommap"
	"net/http"
	"os"

	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"github.com/fsnotify/fsnotify"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ionosnetworks/qfx_cmn/blog"
	sm "github.com/ionosnetworks/qfx_dp/fwder/scheduler"
	"github.com/ionosnetworks/qfx_dp/slcemulator"

	"time"

	"github.com/ionosnetworks/qfx_dp/infra"
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

type ftpResp struct {
	ID     string
	path   string
	status bool
}

// TenantInfo ... Information regarding tenant. Currently unused
type TenantInfo struct {
	ID   uint32
	Name string
}

var ctx = "IonosFwder"

const (
	CPE           = iota
	SLC           = iota
	MAX_IN_FLIGHT = 250
)

type nhInfo struct {
	address     string
	port        int
	regionID    string
	personality int
}

type forwarderTable struct {
	m    map[infra.CsID]nhInfo
	lock *sync.Mutex
}

type slcInfo struct {
	address  string
	port     int
	regionID string
}

var localSlc slcInfo

var siteFwdTable forwarderTable

var fwderToDpChannel chan string

var ftpdChannel chan ftpResp

var fwderToFtpdChannel chan string

var logger blog.Logger

func createFwdTable() {
	/* Create a Map for holding forward rules [cpeID : nhInfo] */
	siteFwdTable.m = make(map[infra.CsID]nhInfo)
	siteFwdTable.lock = new(sync.Mutex)
	logger.Info(ctx, "Created Forwarding Table", nil)
}

func (fwd forwarderTable) Get(cpeID infra.CsID) (fwdRule nhInfo, ok bool) {
	fwd.lock.Lock()
	fwdRule, ok = fwd.m[cpeID]
	fwd.lock.Unlock()
	return
}

func (fwd forwarderTable) Put(cpeID infra.CsID, rule nhInfo) (ok error) {
	fwd.lock.Lock()
	fwd.m[cpeID] = rule
	fwd.lock.Unlock()
	return nil
}

func (fwd forwarderTable) Delete(cpeID infra.CsID) (ok bool) {
	fwd.lock.Lock()
	delete(fwd.m, cpeID)
	fwd.lock.Unlock()
	return true
}

func monitorFwderToFtpdChannel(channel chan string) {
	i := 0
	for {
		msg := <-channel
		logger.Info(ctx, "Sending msg to ftpd", blog.Fields{"msg": msg})
		fmt.Println("Sending msg to ftpd", msg)
		//TODO Send to FTPD
		var resp ftpResp
		resp.ID = "GCE-ASIA"
		resp.path = msg
		if i%2 == 0 {
			resp.status = true
		} else {
			resp.status = false
		}
		// i++
		ftpdChannel <- resp
	}
}

func deriveKey(path string) (c infra.UUID) {
	e := strings.Split(path, "-")
	f := strings.Split(e[4], ".")
	chunkID := e[0] + e[1] + e[2] + e[3] + f[0]
	b := []byte(chunkID)
	d := make([]byte, hex.DecodedLen(len(b)))
	_, _ = hex.Decode(d, b)
	copy(c[:], d)
	// fmt.Println("Bytes", c)
	return
}

func monitorFtpdChannel(channel chan ftpResp) {
	for {
		msg := <-channel
		key := deriveKey(msg.path)
		status := msg.status
		if status == false {
			go sm.RequeueEntry(key, msg.ID)
		} else {
			go sm.DeleteEntry(key, msg.ID)
		}
	}
}

func requestFwdRuleFromSLC(cpeID infra.CsID) *nhInfo {
	localSlc := os.Getenv("LOCAL_SLC_IP")
	if localSlc == "" {
		logger.ErrS(ctx, "No SLC configured")
		return nil
	}

	url := fmt.Sprintf("http://%s/%s", localSlc, cpeID.String())
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.ErrS(ctx, "Error forming request to slc"+err.Error())
		return nil
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Err(ctx, "error performing GET At URL",
			blog.Fields{
				"url": url,
				"err": err.Error()})
	}
	defer resp.Body.Close()
	var n slcemulator.NhInfo
	if err = json.NewDecoder(resp.Body).Decode(&n); err != nil {
		logger.ErrS(ctx, err.Error())
		return nil
	}
	nh := new(nhInfo)
	nh.address = n.SlcIP
	nh.port = 130001
	nh.regionID = n.Name
	nh.personality = SLC
	// fmt.Printf("%+v\n", nh)
	return nh
}

func getFwdRule(fwdTable forwarderTable,
	cpeID infra.CsID, cHop uint32) (rule nhInfo, status bool) {
	logger.Info(ctx, "Getting fwd rule for ",
		blog.Fields{"cpeID": cpeID.String()})
	status = true
	rule, ok := fwdTable.Get(cpeID)
	if ok != true {
		if cHop < 2 {
			logger.Info(ctx, "Did not find a fwd rule for ",
				blog.Fields{"cpeID": cpeID.String()})
			r := requestFwdRuleFromSLC(cpeID)
			rule = *r
			err := fwdTable.Put(cpeID, rule)
			if err != nil {
				logger.Info(ctx,
					"Failed to insert fwd rule", nil)
				status = false
			}
		} else {
			logger.Info(ctx,
				"Not requesting SLC for fwd rule"+
					"Ignoring the destination",
				blog.Fields{"cpeID": cpeID.String()})
			status = false
		}
	} else {
		logger.Info(ctx, "Found fwd rule for ",
			blog.Fields{"cpeID": cpeID.String()})
	}
	return
}

func sendMsgToDP(msg string) {
	// Do Nothing. Pass through to DP
	fwderToDpChannel <- msg
}

func readAndUpdateCurrentHop(path string) (hop uint32, err error) {
	// STOR /mnt/ftp/1/jobID1-98-100-4194522-0.partial
	// TODO Parse and get current hop
	file, err := os.OpenFile(path, os.O_RDWR, 0777)
	if err != nil {
		return 0, err
	}
	mmap, err := gommap.Map(file.Fd(), gommap.PROT_READ|gommap.PROT_WRITE,
		gommap.MAP_SHARED)
	if err != nil {
		fmt.Println("Error mmaping file", err.Error())
		return 0, err
	}
	// end := bytes.Index(mmap, []byte("\n"))
	hop = uint32(mmap[9])
	mmap[9] += 1
	// fmt.Println(string([]byte(mmap[:end])))
	mmap.Sync(gommap.MS_SYNC)
	mmap.UnsafeUnmap()
	file.Close()
	fmt.Printf("Current Hop : %d, Path: %s", hop, path)
	return hop, nil

}

func monitorRamdisk() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	//TODO This path might be different
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
					fmt.Println("created", event.Name)
					fpath := event.Name
					if strings.HasPrefix(fpath,
						"/mnt/ftp/1/.") {
						// Do nothing
					} else {
						go handleStorMsg(fpath)
					}

				}
			}

		case err := <-watcher.Errors:
			fmt.Println(err)
		}

	}

}

func parseHeader(fpath string) (uuid infra.UUID, cpeList []infra.CsID,
	prio int, err error) {
	f, _ := os.Open(fpath)
	defer func() {
		f.Close()
	}()
	// Read version
	unused1 := make([]byte, HEAD_VER)
	_, _ = f.Read(unused1)

	// Read the length of outer header
	bufLen := make([]byte, HEAD_LEN)
	_, _ = f.Read(bufLen)
	num := binary.LittleEndian.Uint32(bufLen)
	fmt.Println("Bytes of header left to read: ", num)

	// Read hop and syncID
	unused := make([]byte, HEAD_HOP+HEAD_SYNC)
	_, _ = f.Read(unused)

	// Read Chunk ID
	buf1 := make([]byte, HEAD_CID)
	_, _ = f.Read(buf1)

	var cID [16]byte
	for i := 0; i < HEAD_CID; i++ {
		cID[i] = buf1[i]
	}
	var chunkID infra.UUID = cID
	uuid = cID
	fmt.Println("ChunkID: ", chunkID)

	// Read the outer header
	// Decode dcpe
	num = num - HEAD_CID - HEAD_HOP - HEAD_SYNC
	buf := make([]byte, num)
	_, _ = f.Read(buf)
	dcpe := []infra.CsID{}
	end := uint32(HEAD_PRIO + HEAD_KEY_IND + HEAD_KEY_LAST)
	key := uint32(HEAD_KEY_IND + HEAD_KEY_LAST)
	gob.NewDecoder(bytes.NewBuffer(buf[:num-end])).Decode(&dcpe)
	for _, i := range dcpe {
		cpeList = append(cpeList, i)
	}

	prio = int(binary.LittleEndian.Uint32(buf[num-end : num-key]))

	return

}

func handleStorMsg(fpath string) {
	uuid, cpeList, prio, err := parseHeader(fpath)
	err = err
	currentHop, err := readAndUpdateCurrentHop(fpath)
	if err != nil {
		logger.Err(ctx, "Unable to get current Hop",
			blog.Fields{"Err": err.Error()})
		return
	}
	for _, cpe := range cpeList {
		logger.Info(ctx, "Handling the destination",
			blog.Fields{"cpeID": cpe.String()})
		nextHop, status := getFwdRule(siteFwdTable, cpe, currentHop)
		if status == false {
			// No fwd rule available
			continue
		}

		if nextHop.personality == CPE {
			// we are at destination CPE
			go sendMsgToDP(fpath)
		} else {
			nhRegion := nextHop.regionID
			scheduler := sm.FindSchedulerInstance(nhRegion)
			if scheduler == nil {
				logger.Err(ctx, "Unable to find Scheduler",
					blog.Fields{"cpeID": string(cpe[:16])})
				continue
			}
			// go scheduler.CancelSched()
			go scheduler.InsertToHeap(uuid, fpath, prio, true)
		}
	}
}

func dumpFwdTable() {
	for {
		siteFwdTable.lock.Lock()
		logger.Info(ctx, "Dumping the Forwarding Table", nil)
		for key, index := range siteFwdTable.m {
			logger.Info(ctx, "Index, Key, Value",
				blog.Fields{"index": index,
					"key": key, "value": siteFwdTable.m[key]})
		}
		siteFwdTable.lock.Unlock()
		time.Sleep(time.Second * 2)
	}
}

/*
func monitorFwderToDpChannel(channel chan StorMsg) {
	logger.Info(ctx, "Waiting for STOR to be sent to DP", nil)
	for {
		msg := <-channel
		logger.Info(ctx, "NH is CPE .. sending to channel", nil)
		msg = msg
		// TODO Open socket and send if fwder and DP are different processes
	}
}
*/

// StartFwder ... Initializes and starts forwarder
func StartFwder(fwderChannel chan string) {
	// fo, err := os.OpenFile("test.log", os.O_RDWR|os.O_CREATE, 0644)
	logger = blog.New("192.168.1.141:2000", "12", "12")
	logger.SetLevel(blog.Debug)
	//	logger = log
	logger.Info(ctx, "Starting Fwder", nil)
	createFwdTable()
	fmt.Println("Started forwarder")

	/* Create a Channel for receiving STOR msgs (from ICA-DP or FTPD)*/
	// storChannel = make(chan storMsg, 1024)
	// storChannel = channel
	// go monitorStorChannel(storChannel)

	fwderToDpChannel = fwderChannel // make(chan StorMsg, 1024)
	// go monitorFwderToDpChannel(fwderToDpChannel)

	ftpdChannel = make(chan ftpResp, 1024)
	go monitorFtpdChannel(ftpdChannel)

	fwderToFtpdChannel = make(chan string, 1024)
	go monitorFwderToFtpdChannel(fwderToFtpdChannel)

	sm.InitScheduler(logger, fwderToFtpdChannel)
	if len(os.Args) <= 1 {
		go monitorRamdisk()
	}
	// go dumpFwdTable()

}

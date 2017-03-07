package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/infra"
	"github.com/ionosnetworks/qfx_dp/priorityQ"
)

type siEntry struct {
	chunkPath string
	vtime     float64
	prio      int
}

type Scheduler interface {
	Load() error
	Save() error
	InsertToHeap(infra.UUID, string, int, bool)
	CancelSched()
}

var ctx = "IonosScheduler"
var logger blog.Logger
var fwderToFtpdChannel chan string

type schedulerInstance struct {
	/* Priority Queues */
	// 2 queues will be required one for Platinum and one for others
	pqs []*priorityQ.PriorityQ

	/* Number of chunks in the heap*/
	numChunks uint32

	/* Next hop ID */
	nh string

	/* Lock on the scheduler Instance */
	Lock *sync.Mutex

	/* Cond to check empty */
	empty *sync.Cond

	/* Number of entries in heap*/
	count uint32

	/* Number of entries sent to FTPD and not acked */
	inFlight uint32

	/* Cond to check in flight entries */
	full *sync.Cond

	/* Channel to which entries have to be written */
	//	reqChan chan siEntry

	/* Chunks in transit */
	inFlightTable map[infra.UUID]siEntry
	ifLock        *sync.Mutex

	/* Current vTime */
	currentVtime float64
	vTimeLock    *sync.Mutex

	/* Channele to indicate closure */
	done chan struct{}

	/* Context cancel */
	//XXX Probably not a good idea to have context inside a struct
	cancel context.CancelFunc
}

type schedTable struct {
	/* NH region ID to scheduler Instance map */
	m    map[string]schedulerInstance
	lock *sync.Mutex
}

var SiteSchedTable schedTable

// Can be invoked to stop a particular scheduler instance
func (sched *schedulerInstance) CancelSched() {
	/*
		for i := 0; i < 15; i++ {
			time.Sleep(time.Second * 1)
		}
	*/
	sched.cancel()
}

func InitScheduler(log blog.Logger, channel chan string) {
	fwderToFtpdChannel = channel
	logger = log

	SiteSchedTable.m = make(map[string]schedulerInstance)
	SiteSchedTable.lock = new(sync.Mutex)
	logger.Info(ctx, "Creating scheduler table", nil)
}

func (table schedTable) Get(nh string) (sched schedulerInstance, ok bool) {
	table.lock.Lock()
	sched, ok = table.m[nh]
	table.lock.Unlock()
	return
}

func (table schedTable) Put(nh string, sched schedulerInstance) (ok error) {
	table.lock.Lock()
	table.m[nh] = sched
	table.lock.Unlock()
	return
}
func (table schedTable) Delete(nh string) (ok bool) {
	table.lock.Lock()
	delete(table.m, nh)
	table.lock.Unlock()
	return true
}

func (sched *schedulerInstance) Load() (err error) {
	rule, ok := SiteSchedTable.Get(sched.nh)
	if ok == true {
		*sched = rule
		err = nil
	} else {
		err = errors.New("RULE NOT FOUND")
		*sched = *sched
	}
	return
}

func (sched *schedulerInstance) Save() (ok error) {
	ok = SiteSchedTable.Put(sched.nh, *sched)
	if ok != nil {
		logger.Info(ctx, "Failed to insert scheduler ", nil)
	}
	return
}

/* internal method to initialize a SI */
func (sched *schedulerInstance) initSchedInstance(nPrio int) {
	//logger.Info("Creating new sched instance")
	sched.Lock = new(sync.Mutex)
	sched.ifLock = new(sync.Mutex)
	sched.vTimeLock = new(sync.Mutex)
	sched.inFlightTable = make(map[infra.UUID]siEntry)
	sched.empty = sync.NewCond(new(sync.Mutex))
	sched.full = sync.NewCond(new(sync.Mutex))
	sched.done = make(chan struct{})

	sched.pqs = make([]*priorityQ.PriorityQ, nPrio)
	sched.count = 0
	sched.inFlight = 0
	for i := 0; i < nPrio; i++ {
		sched.pqs[i] = priorityQ.CreatePriorityQ()
	}
	sched.currentVtime = 1
	context, cancel := context.WithCancel(context.Background())
	sched.cancel = cancel
	go sched.drainHeap(context)
	return
}

/* Exposed API to find SI corresponding to a NH */
func FindSchedulerInstance(nh string) Scheduler {
	scheduler := new(schedulerInstance)
	scheduler.nh = nh
	err := scheduler.Load()
	if err != nil {
		// Initialize the sched instance with number of priority Queues required
		scheduler.initSchedInstance(2)
		err = scheduler.Save()
		if err != nil {
			logger.Err(ctx, "Unable to save scheduler table", nil)
			return nil
		}
	}
	return scheduler
}

func (scheduler *schedulerInstance) InsertToHeap(uuid infra.UUID, chunk string,
	prio int, check bool) {
	// We have the next hop, insert the chunk to corresponding heap
	// logger.Info("Loaded scheduler for nh", nhRegion, scheduler)
	if scheduler == nil {
		logger.Err(ctx, "Cannot insert to invalid heap", nil)
		return
	}
	entry := new(siEntry)
	entry.chunkPath = chunk
	entry.prio = prio
	scheduler.Lock.Lock()
	if check == true {
		/* Check if it already exists in Heap */
		exists := scheduler.verifyIfEntryExists(uuid)
		if exists == true {
			// fmt.Println("already in heap ")
			logger.Info(ctx, "already in heap. Not adding entry ",
				blog.Fields{"path": entry.chunkPath})
			scheduler.Lock.Unlock()
			return
		}
	}
	vtime := scheduler.calculateVtime(prio)
	entry.vtime = vtime
	scheduler.pqs[prio].Push(entry, vtime, 1)
	scheduler.addToInFlightTable(uuid, entry)
	scheduler.Lock.Unlock()

	scheduler.empty.L.Lock()
	scheduler.count++
	scheduler.empty.L.Unlock()
	scheduler.empty.Signal()

	logger.Info(ctx, "Pushed to heap : ", blog.Fields{"chunk": chunk,
		"vtime": vtime})
	scheduler.Save()
}

func (sched *schedulerInstance) calculateVtime(jobPrio int) (currentVtime float64) {
	// TODO add logic to calculate vtime
	sched.vTimeLock.Lock()
	currentVtime = sched.currentVtime
	sched.currentVtime += 1

	// logger.Info(ctx, "Scheduler:", blog.Fields{"sched": sched})
	logger.Info(ctx, "Current vtime of scheduler",
		blog.Fields{"sched vtime": sched.currentVtime})
	sched.Save()
	sched.vTimeLock.Unlock()
	return
}

func (sched *schedulerInstance) verifyIfEntryExists(key infra.UUID) (exists bool) {
	exists = false
	sched.ifLock.Lock()
	_, exists = sched.inFlightTable[key]
	sched.ifLock.Unlock()
	return
}

func (sched *schedulerInstance) addToInFlightTable(key infra.UUID, entry *siEntry) {
	sched.ifLock.Lock()
	sched.inFlightTable[key] = *entry
	sched.ifLock.Unlock()
}

func (sched *schedulerInstance) drainHeap(context context.Context) {
	for {
		select {
		case <-context.Done():
			logger.Info(ctx, "Scheduler instance cancelled from ctx", nil)
			return
		default:
			sched.empty.L.Lock()
			if sched.count == 0 {
				sched.empty.Wait()
			}
			sched.empty.L.Unlock()

			//TODO wait for notification from FTPD
			// <-sched.respChannel
			for _, val := range sched.pqs {
				if val.Length() == 0 {
					continue
				}
				logger.Info(ctx, "elements in heap: ",
					blog.Fields{"length": val.Length()})
				sched.Lock.Lock()
				entry := val.Pop()
				sched.Lock.Unlock()
				sched.empty.L.Lock()
				sched.count--
				sched.empty.L.Unlock()
				sientry := entry.(*siEntry)
				logger.Info(ctx, "Popped Entry from heap",
					blog.Fields{"entry": sientry.chunkPath,
						"vtime": sientry.vtime})
				//TODO send popped entry to DTE
				fwderToFtpdChannel <- sientry.chunkPath
				sched.full.L.Lock()
				sched.inFlight++
				sched.full.L.Unlock()
				break
			}
		}
	}
}

func RequeueEntry(key infra.UUID, region string) {
	scheduler := new(schedulerInstance)
	scheduler.nh = region
	err := scheduler.Load()
	if err != nil {
		logger.Err(ctx, "Unable to get scheduler ", nil)
		return
	}
	logger.Info(ctx, "Requeuing entry ", blog.Fields{"key": key})
	scheduler.ifLock.Lock()
	entry, ok := scheduler.inFlightTable[key]
	if ok == true {
		go scheduler.InsertToHeap(key, entry.chunkPath, entry.prio, false)
	} else {
		//TODO is this an error scenario ? What needs to be done
	}
	scheduler.ifLock.Unlock()
	scheduler.Save()
}

func DeleteEntry(key infra.UUID, region string) {
	scheduler := new(schedulerInstance)
	scheduler.nh = region
	err := scheduler.Load()
	if err != nil {
		logger.Err(ctx, "Unable to get scheduler ", nil)
		return
	}
	logger.Info(ctx, "Deleting entry from in flight table",
		blog.Fields{"key": key.String()})
	scheduler.ifLock.Lock()
	delete(scheduler.inFlightTable, key)
	scheduler.ifLock.Unlock()
	scheduler.Save()
}

func DumpSchedulerHeaps() {
	for {
		SiteSchedTable.lock.Lock()
		logger.Info(ctx, "Dumping the scheduler heaps", nil)
		for key, _ := range SiteSchedTable.m {
			value := SiteSchedTable.m[key]
			for _, val := range value.pqs {
				logger.Info(ctx, "elements in heap: ",
					blog.Fields{"length": val.Length()})
				for val.Length() > 0 {
					entry := val.Pop()
					sientry := entry.(*siEntry)
					logger.Info(ctx, "Entry",
						blog.Fields{"entry": sientry.chunkPath,
							"vtime": sientry.vtime})
				}
			}
		}
		SiteSchedTable.lock.Unlock()
		time.Sleep(time.Second * 2)
	}
}

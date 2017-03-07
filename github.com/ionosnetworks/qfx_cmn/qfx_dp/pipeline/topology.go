package pipeline

import (
	"errors"
	// l "github.com/Sirupsen/logrus"
	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/priorityQ"
	//	"path"
	//	"reflect"
	"sync"
)

/* A job flows through the pipeline according to a series of
 * its interests
 */
type InterestFlow []string

// var log *l.Entry
var log blog.Logger
var ctx = "pipeline"

/* A set of stage for a topology */
type Topology struct {
	name string
	/* stageTable is the collection of all the stages that have been
	 * initialized in the system
	 */
	stageTable     map[string]*Stage
	stageTableLock *sync.RWMutex
	/* Interest Mapping keeps a mapping from an "interest"
	 * to all the InputChannels that are interested in the
	 * particular interest. A stage while registering itself to a topology
	 * advertises all the input channels and their
	 * interests. Once a stage has some processed
	 * output data it asks the toplogy for the channels
	 * "interested" in its out put data and pushes
	 * the data to those channels
	 */
	interestMapping map[string][]chan ProcessReqResp

	/* Interest to stage map */
	interestStage map[string]*Stage
}

/* CreateTopology() creates a topology of Stages.
 * Stages can be added prior to execution of a topology
 */

func CreateTopology(name string, logger blog.Logger) (tp *Topology, err error) {
	log = logger
	/*
		type Empty struct{}
		pkgName := path.Base(reflect.TypeOf(Empty{}).PkgPath())
		log = l.WithFields(l.Fields{
			"pkg": pkgName,
		})
	*/

	tp = new(Topology)
	if tp == nil {
		err = errors.New("Failed creating topology")
		tp = nil
		return
	}
	tp.stageTable = make(map[string]*Stage)
	tp.stageTableLock = &sync.RWMutex{}
	tp.interestMapping = make(map[string][]chan ProcessReqResp)
	tp.interestStage = make(map[string]*Stage)
	tp.name = name
	err = nil
	log.Info(ctx, "Created topology", blog.Fields{"Name": tp.name})
	return
}

func (tp *Topology) UpdateInterestMapping(interest string,
	inch chan ProcessReqResp) {
	tp.interestMapping[interest] =
		append(tp.interestMapping[interest], inch)
}

func (tp *Topology) UpdateInterestStage(interest string, stage *Stage) {
	tp.interestStage[interest] = stage
}

func (tp *Topology) GetChannels(interest string) (chanList []chan ProcessReqResp,
	ok bool) {
	chanList, ok = tp.interestMapping[interest]
	return chanList, ok
}

func (tp *Topology) PushToInterestedChannels(interest string,
	req *ProcessReqResp) (pushed bool) {
	chanList, ok := tp.interestMapping[interest]
	if ok {
		for _, x := range chanList {
			x <- *req
		}
		pushed = true
	} else {
		pushed = false
		log.Err(ctx, "No one interested for req", blog.Fields{"Interest": interest})
	}
	return
}

/* CreateStage() creates a Stage. It expects following parameters
 * - name :- Human readable name of the stage
 * - stops :- A type implementing StageOps interface
 * - nec :- Number of go routines to be spawed for this stage's operation
 * - nprio :- Number of strict priority levels this stage will process
 * - inch :- Inward channels on which requests would be sent to this stage
 * Returns:-
 * - out :- Channel on which output would be sent
 * - stage:- Pointer to the stage object
 * - err :- Error if any else nil
 */
func (tp *Topology) AddStage(name string, stops StageOps,
	nec int, nprio int) (stage *Stage, err error) {
	stage = new(Stage)
	if stage != nil {
		for i := 0; i < nprio; i++ {
			pq := priorityQ.CreatePriorityQ()
			stage.pqs = append(stage.pqs, pq)
		}
		stage.pauseControl = make([]*sync.Mutex, nec)
		for i := 0; i < nec; i++ {
			stage.pauseControl[i] = new(sync.Mutex)
		}
		stage.heapLock = new(sync.Mutex)
		stage.done = make(chan bool)
		stage.nec = nec
		stage.stops = stops
		stage.name = name
		stage.pendingRequests = 0
		stage.empty = &sync.Cond{L: &sync.Mutex{}}
		stage.jobCurrentVtMap = make([]jobVtMapEntry, MAX_PARALLEL_JOBS)
		tp.stageTableLock.Lock()
		tp.stageTable[name] = stage
		tp.stageTableLock.Unlock()
		stage.tp = tp
		err = nil
		log.Info(ctx, "Stage created", blog.Fields{"Name": name})

	} else {
		stage = nil
		err = errors.New("Could not create a stage")
		return
	}
	return
}

func (tp *Topology) GetStage(interest string) *Stage {
	return tp.interestStage[interest]
}

func (tp *Topology) InitTopology() {
	var initStageList []*Stage
	i := 0
	for _, val := range tp.stageTable {
		if val.stops.Init() == true {
			initStageList = append(initStageList, val)
		} else {
			log.Err(ctx, "Failed initializing stage , unwinding topology", blog.Fields{"Name": val.name})
			for j := i; j > 0; j-- {
				initStageList[j].stops.Exit()
			}
		}
	}
}

func (tp *Topology) Start() {
	i := 0
	var initStageList []*Stage
	for _, val := range tp.stageTable {
		if val.Start() == true {
			initStageList = append(initStageList, val)
		} else {
			log.Err(ctx, "Failed starting stage , unwinding topology", blog.Fields{"Name": val.name})
			for j := i; j > 0; j-- {
				initStageList[j].Stop()
			}
		}
	}
}

func (tp *Topology) Stop() {
	for _, val := range tp.stageTable {
		val.Stop()
		log.Info(ctx, "Stopped stage", blog.Fields{"Name": val.name})
	}

}

func (tp *Topology) Pause(iFlow InterestFlow) {
	for _, interest := range iFlow {
		tp.GetStage(interest).Pause()
	}
}

func (tp *Topology) Resume(iFlow InterestFlow) {
	for _, interest := range iFlow {
		tp.GetStage(interest).Resume()
	}
}

func (tp *Topology) Cancel(jobId uint32, iFlow InterestFlow) (nCancel int) {
	nCancel = 0
	for _, interest := range iFlow {
		stage := tp.GetStage(interest)
		stage.Pause()
		nCancel += stage.Cancel(jobId)
		stage.Resume()
	}
	return nCancel
}

/*
func (tp *Topology) HandlePrioChange(jobId uint32,
	oldStrictPrio int, newStrictPrio int, newPrio int, iFlow InterestFlow) {
	for _, interest := range iFlow {
		tp.GetStage(interest).HandlePrioChange(jobId, oldStrictPrio,
			newStrictPrio, newPrio)
	}
}
*/

func (tp *Topology) DumpStages() {

	tp.stageTableLock.RLock()
	for x, v := range tp.stageTable {
		log.Debug(ctx, "Dumping Stage", blog.Fields{"key:": x, "value": v})
	}
	tp.stageTableLock.RUnlock()
}

func (tp *Topology) GetPendingRequests() map[string]uint32 {
	stageReq := make(map[string]uint32)
	tp.stageTableLock.RLock()
	for x, v := range tp.stageTable {
		stageReq[x] = v.GetPendingRequests()
	}
	tp.stageTableLock.RUnlock()
	return stageReq
}

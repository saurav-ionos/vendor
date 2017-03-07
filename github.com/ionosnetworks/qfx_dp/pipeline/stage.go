/**
 * Package pipepine provides primitives to build a software pipeline
 * Multiple stages can be connected together in a Directed Acyclic Graph
 * in an initial toptology phase to build up the complete pipeline
 */
package pipeline

import (
	"reflect"
	"runtime"
	"sync"

	//	log "github.com/Sirupsen/logrus"
	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/infra"
	"github.com/ionosnetworks/qfx_dp/priorityQ"
)

const (
	JOB_STRICT_PRIO_HIGH   int = 0
	JOB_STRICT_PRIO_NORMAL int = 1
)

const (
	MAX_PARALLEL_JOBS int = 65536
)

const (
	VIRTUAL_TIME_WEIGHT_CONSTANT float64 = 1
)

const (
	APPROX_ZERO float64 = 0.01
	APPROX_ONE  float64 = 0.99
)

type PipelineResp struct {
	SyncID  uint32
	UUID    infra.UUID
	MsgType int
	Status  uint32
	Req     *ProcessReqResp
}

/* Each stage consumes a ProcessReqResp
 * for processing and Emits a ProcessReqResp
 * as an output
 */
type ProcessReqResp struct {
	SyncID          uint32
	Prio            int
	Interests       InterestFlow
	CurrentInterest int
	Data            interface{}
	ChunkNum        uint32
	RespChan        chan interface{}
	UUID            infra.UUID
	MsgType         int
}

type jobVtMapEntry struct {
	currentVtime float64
	quotient     int
}

/* Stage struct defines a stage in the pipeline
 * It contains references to all the variables
 * a stage uses. It implements the following
 * functions
 * - Start() - Starts a stage operation
 * - Stop() - Stops a stage operation
 */

type Stage struct {
	name string
	/* The topology to which this stage
	 * belongs to
	 */
	tp    *Topology
	nec   int
	stops StageOps
	/* A slice of pointers to the priority queues
	 * for the stage. The slice would contain
	 * the priority queues in decreasing order of
	 * priority
	 */
	pqs []*priorityQ.PriorityQ
	/* Pointer to a slice containing all the
	 * input channels to this stage
	 */
	inch []chan ProcessReqResp
	done chan bool

	/* The number of pending requests this stage has */
	pendingRequests uint32
	/* Condition on which consume routine will wait */
	empty *sync.Cond
	/* The following locks are used to pause the stage's
	 * execution context and resume them
	 */
	pauseControl []*sync.Mutex

	/* Heap Lock - To stop inflow and out flow of
	 * elements from heap
	 */
	heapLock *sync.Mutex

	/* jobCurrentVtMap keeps track of all the
	 * running jobs in the session. We assume that two simulataneous
	 * executing jobs won't have more than a gap of MAX_PARALLEL_JOBS
	 */
	jobCurrentVtMap []jobVtMapEntry
}

/* To qualify as a stage a type needs to implement
 * the StageOps interface
 * - Init() - Is called after the stage is created and the internal
 *            data structures have been populated
 * - Process() - is called for each ProcessReqResp that is passed
 *               as an input to the stage.
 * - Exit() - Exit is called after all the process routines are done
 *            executing
 */
type StageOps interface {
	Init() bool
	Process(name string, req *ProcessReqResp) bool
	Exit()
	HeaderSpace() uint64
}

/* AddInterest() - Add an intereset fort this stage.
 * This stage would receive all the interested values
 * on its input channels. This creates a interest and
 * it's associated channel and puts it in the topology specific
 * interest table
 */

func (stage *Stage) AddInterest(interest string) (inch chan ProcessReqResp) {
	inch = make(chan ProcessReqResp)
	stage.inch = append(stage.inch, inch)
	stage.tp.UpdateInterestMapping(interest, inch)
	stage.tp.UpdateInterestStage(interest, stage)
	return
}

func (stage *Stage) emit(resp ProcessReqResp) {
	log.Debug(ctx, "emiting from", blog.Fields{"Name": stage.name})
	resp.CurrentInterest++
	if resp.CurrentInterest >= len(resp.Interests) {
		log.Debug(ctx, "No where to emit", nil)
		return
	}
	chs, ok := stage.tp.GetChannels(resp.Interests[resp.CurrentInterest])

	if ok != false {
		for _, x := range chs {
			log.Debug(ctx, "Emitted to ",
				blog.Fields{"Name": resp.Interests[resp.CurrentInterest]}) //, "Channel": x})
			x <- resp
		}
	} else {
		log.Debug(ctx, "Nowehere to emit", nil)
	}
}

func (stage *Stage) runUnitExecContext(id int) {
	/* Launch a function that reads from the priority queue
	 * and passes each element to the stage's process function
	 */
	for {
		select {
		case <-stage.done:
			log.Info(ctx, "Finishing one unit context", nil)
			break
		default:
			/* Wait for a condition here */
			stage.empty.L.Lock()
			if stage.pendingRequests == 0 {
				stage.empty.Wait()
			}
			stage.empty.L.Unlock()
			stage.pauseControl[id].Lock()
			for _, val := range stage.pqs {
				if val.Length() == 0 {
					/* Move to a lower prio
					 * queue
					 */
					continue
				} else {
					req := val.Pop()
					if req == nil {
						continue
					}
					resp := req.(ProcessReqResp)
					ok := stage.stops.Process(stage.name,
						&resp)
					stage.empty.L.Lock()
					stage.pendingRequests--
					stage.empty.L.Unlock()
					if ok {
						stage.emit(resp)
					} else {
						log.Err(ctx,
							"Process returned failure",
							blog.Fields{"Name": stage.name, "Resp": resp})
					}
					break
				}
			}
			stage.pauseControl[id].Unlock()
		}
		runtime.Gosched()
	}
}

/*
 * Start a stage
 */

func (stage *Stage) Start() bool {
	for j := 0; j < stage.nec; j++ {
		go stage.runUnitExecContext(j)
	}
	/* Now that go routines are on standby, pull
	 * requests from the input channels and dump
	 * them on to the priorityQ. The go routines
	 * are responsible for pull requests from the
	 * prioityQ , process them and push them to the
	 * output channel
	 */
	go stage.consume(stage.inch)
	return true
}

/* Pause a stage */
func (stage *Stage) Pause() {

	for j := 0; j < stage.nec; j++ {
		stage.pauseControl[j].Lock()
	}
	stage.heapLock.Lock()
}

/* Resume a stage */
func (stage *Stage) Resume() {
	for j := 0; j < stage.nec; j++ {
		stage.pauseControl[j].Unlock()
	}
	stage.heapLock.Unlock()
}

/* Stop a stage */
func (stage *Stage) Stop() {
	stage.empty.Broadcast()
	for i := 0; i < stage.nec; i++ {
		stage.done <- true
	}
}

/* Cancel a job from stage */
func (stage *Stage) Cancel(jobId uint32) (nCancel int) {
	nCancel = 0
	/* Remove all the elements from the heap
	 * for this job and requeue the heap
	 */
	for _, x := range stage.pqs {
		nCancel += x.Delete(jobId)
	}
	//	log.Debug(ctx, "Stage:", stage.name, "cancelled entries", nCancel)
	stage.empty.L.Lock()
	stage.pendingRequests -= uint32(nCancel)
	stage.empty.L.Unlock()
	return nCancel
}

/* Needs the heap to be in quicent stage. Heaplock should be held
 * when calling
 * this
 */
/*
func (stage *Stage) HandlePrioChange(jobId uint32,
	oldStrictPrio int, newStrictPrio int, newPrio int) {
	// Invalidate the entry in the jobTableVtMap
	vtmapIndex := jobId % uint32(MAX_PARALLEL_JOBS)
	stage.jobCurrentVtMap[vtmapIndex].quotient = -1
	stage.jobCurrentVtMap[vtmapIndex].currentVtime = -1

	_, elems := stage.pqs[oldStrictPrio].Scavenge(jobId)
	for _, req := range elems {
		newReq := req.(ProcessReqResp)
		newReq.JobPrio = newPrio
		newReq.JobStrictPrio = newStrictPrio
		newvtime := calculateVtime(stage,
			stage.pqs[newStrictPrio].PeekTop(),
			newStrictPrio,
			newPrio,
			int(jobId))
		stage.pqs[newStrictPrio].Push(newReq, newvtime, jobId)
		//		log.Debugf("changing prio of %+v to %f\n", newReq, newvtime)
	}

}
*/

func (stage *Stage) HeaderSpace() uint64 {
	return stage.stops.HeaderSpace()
}

func (stage *Stage) GetPendingRequests() uint32 {
	return stage.pendingRequests
}

func (stage *Stage) consume(inch []chan ProcessReqResp) {

	cases := make([]reflect.SelectCase, len(inch))

	for i, ch := range inch {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv,
			Chan: reflect.ValueOf(ch)}
	}
	remaining := len(cases)
	for remaining >= 0 {
		chosen, value, ok := reflect.Select(cases)
		if !ok {
			cases[chosen].Chan = reflect.ValueOf(nil)
			remaining -= 1
			continue
		}
		intf := value.Interface()
		req := intf.(ProcessReqResp)
		if req.Prio >= len(stage.pqs) {
			log.Err(ctx, "Invalid request JobStrictPrio more than available prioirty queues", nil)
			continue
		}
		stage.heapLock.Lock()
		top := stage.pqs[req.Prio].PeekTop()
		vtime := calculateVtime(stage, top, req.Prio,
			req.Prio, int(req.SyncID))
		stage.pqs[req.Prio].Push(req, vtime, uint32(req.SyncID))
		stage.heapLock.Unlock()
		runtime.Gosched()
		stage.empty.L.Lock()
		stage.pendingRequests++
		stage.empty.L.Unlock()
		stage.empty.Signal()
	}
}

/**
 * We assume currently there is not a gap of more that 65536 between
 * two simultaneously executing jobs in the system
 */
func calculateVtime(stage *Stage, top float64, jobStrictPrio int, jobPrio int,
	jobId int) (currentVtime float64) {
	jobCurrentVtMap := stage.jobCurrentVtMap
	vtmapIndex := jobId % MAX_PARALLEL_JOBS
	vtmapQuot := jobId / MAX_PARALLEL_JOBS
	if top < APPROX_ZERO {
		jobCurrentVtMap[vtmapIndex].quotient = vtmapQuot
		jobCurrentVtMap[vtmapIndex].currentVtime = 1
		currentVtime = 1
		return
	}
	/* We have seen this job earlier in the current
	 * session if the quotient of the current job with
	 * MAX_PARALLEL_JOBS is same as in the jobCurrentVtMap
	 * and there is a virtual time associated with it which
	 * will be always >= 1
	 */
	if jobCurrentVtMap[vtmapIndex].quotient == vtmapQuot &&
		jobCurrentVtMap[vtmapIndex].currentVtime > APPROX_ONE {
		currentVtime = jobCurrentVtMap[vtmapIndex].currentVtime
		currentVtime = currentVtime +
			VIRTUAL_TIME_WEIGHT_CONSTANT/float64(jobPrio)
		jobCurrentVtMap[vtmapIndex].currentVtime = currentVtime
	} else {
		jobCurrentVtMap[vtmapIndex].currentVtime = top
		jobCurrentVtMap[vtmapIndex].quotient = vtmapQuot
		currentVtime = jobCurrentVtMap[vtmapIndex].currentVtime
	}

	return
}

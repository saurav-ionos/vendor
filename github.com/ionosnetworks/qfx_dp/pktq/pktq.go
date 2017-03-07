package pktq

import (
	"fmt"
	"sync"

	"github.com/ionosnetworks/qfx_dp/infra"
)

type qEntry struct {
	data       []byte
	syncID     uint32
	retryCount uint32
	numDest    uint32
	respChan   chan struct{}
}

type pktQ struct {
	txtoken chan struct{}
	sync.Mutex
	q      map[infra.UUID]*qEntry
	rxChan chan infra.UUID
}

const (
	PQ_SIZE = 10
)

var packetQueue *pktQ
var initialized bool = false

func (pq *pktQ) handleAck(uuid infra.UUID) {
	//TODO Handle ack
	pq.Lock()
	defer pq.Unlock()
	entry, ok := pq.q[uuid]
	if ok {
		entry.numDest--
		if entry.numDest == 0 {
			delete(pq.q, uuid)
			<-pq.txtoken
			entry.respChan <- struct{}{}
		} else {
			pq.q[uuid] = entry
		}
	}
}

func (pq *pktQ) processAcks() {
	for {
		msg := <-pq.rxChan
		go pq.handleAck(msg)
	}
}

func New() *pktQ {
	pq := new(pktQ)
	pq.txtoken = make(chan struct{}, PQ_SIZE)
	pq.q = make(map[infra.UUID]*qEntry, PQ_SIZE)
	pq.rxChan = make(chan infra.UUID, 1024)
	go pq.processAcks()
	return pq
}

func (pq *pktQ) Insert(payload []byte, syncID uint32,
	chunkID infra.UUID, length uint32, ch chan struct{}) {
	// Wait till token is available
	pq.txtoken <- struct{}{}
	entry := &qEntry{
		syncID:     syncID,
		data:       payload,
		numDest:    length,
		respChan:   ch,
		retryCount: 0,
	}
	pq.Lock()
	pq.q[chunkID] = entry
	pq.Unlock()
	fmt.Println("Inserted to pkt Q", syncID, chunkID)
}

func Get() *pktQ {
	if initialized == false {
		packetQueue = New()
		initialized = true
	}
	return packetQueue
}

func GetChan() chan infra.UUID {
	if initialized == false {
		return nil
	}
	return packetQueue.rxChan
}

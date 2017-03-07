/**
 * This package implementa a priority queue for
 * the Requests using the
 * Go's inbuilt heap container type
 */
package priorityQ

import (
	"container/heap"
	"sync"
	"sync/atomic"
)

type PriorityQueue []pqNode

type PriorityQ struct {
	priorityQ PriorityQueue
	sync.Mutex
	length int32
}

type pqNode struct {
	data     interface{}
	priority float64
	jobId    uint32
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	req := x.(pqNode)
	*pq = append(*pq, req)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}

func CreatePriorityQ() (pq *PriorityQ) {
	pq = new(PriorityQ)
	pq.length = 0
	return
}

func (pq *PriorityQ) Push(data interface{}, prio float64, jobId uint32) {
	pq.Lock()
	if len(pq.priorityQ) == 0 {
		pq.priorityQ = make([]pqNode, 1)
		pq.priorityQ[0].data = data
		pq.priorityQ[0].priority = prio
		heap.Init(&pq.priorityQ)
	} else {
		heap.Push(&pq.priorityQ, pqNode{data: data,
			priority: prio, jobId: jobId})
	}
	atomic.AddInt32(&pq.length, 1)
	pq.Unlock()
}

func (pq *PriorityQ) Pop() (data interface{}) {
	pq.Lock()
	defer pq.Unlock()
	if len(pq.priorityQ) > 0 {
		req := heap.Pop(&pq.priorityQ).(pqNode)
		atomic.AddInt32(&pq.length, -1)
		return req.data
	}
	return nil
}

/* Deletes all elements with provided jobId in the
 * heap. Returns the number of elements deleted
 */
func (pq *PriorityQ) Delete(jobId uint32) (nDel int) {
	nDel = 0
	removed := 0
	pq.Lock()
	for {
		removed = 0
		for i, x := range pq.priorityQ {
			if x.jobId == jobId {
				heap.Remove(&pq.priorityQ, i)
				removed++
				nDel++
				atomic.AddInt32(&pq.length, -1)
				break
			}
		}
		if removed == 0 {
			break
		}
	}
	pq.Unlock()
	return nDel
}

/* Scavenges all elements with provided jobId in the
 * heap. Returns a slice containing the elements
 */
func (pq *PriorityQ) Scavenge(jobId uint32) (nDel int, elems []interface{}) {
	nDel = 0
	elems = make([]interface{}, 0, 10)
	removed := 0
	pq.Lock()
	for {
		removed = 0
		for i, x := range pq.priorityQ {
			if x.jobId == jobId {
				elems = append(elems, pq.priorityQ[i].data)
				heap.Remove(&pq.priorityQ, i)
				removed++
				nDel++
				atomic.AddInt32(&pq.length, -1)
				break
			}
		}
		if removed == 0 {
			break
		}
	}
	pq.Unlock()
	return nDel, elems
}
func (pq *PriorityQ) PeekTop() (prio float64) {
	pq.Lock()
	defer pq.Unlock()
	if len(pq.priorityQ) != 0 {
		prio = pq.priorityQ[0].priority
	} else {
		prio = -1
	}
	return prio
}

func (pq *PriorityQ) Length() (length int) {
	pq.Lock()
	defer pq.Unlock()
	return len(pq.priorityQ)
}

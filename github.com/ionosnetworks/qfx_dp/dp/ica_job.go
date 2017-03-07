package dp

import (
	"github.com/ionosnetworks/qfx_dp/infra"
)

const (
	SUCCESS          uint32 = 0
	FAILURE          uint32 = 1
	CHECK_SUM_ERR    uint32 = 2
	CHECK_UNSUM_ERR  uint32 = 3
	DISK_WRITE_ERR   uint32 = 4
	DISK_FULL_ERR    uint32 = 5
	NO_SRC_FILE_ERR  uint32 = 6
	FILE_CORRUPT_ERR uint32 = 7
	FILE_ACCESS_ERR  uint32 = 8
	CHUNK_IGNORE_CMD uint32 = 9
	SUCCESS_AND_MORE uint32 = 100
	FAILURE_AND_MORE uint32 = 200
)

type ChunkInfo struct {
	ChunkNum            uint32
	ChunkAdvertizedSize uint32
	TotalChunks         uint32
	ChunkPath           string
	ChunkActualSize     uint64
	WriteOffset         uint64
	BufferEndOffset     uint64
	StartJobletIndex    int32
	EndJobletIndex      int32
	UUID                infra.UUID
}

/**
 * An RxJobInfo struct flows through the pipeline
 * with an incoming chunk. It has all the required
 * information for each chunk to process the request
 * and send the chunk forward to the next stage
 * In future if some additional stage is added all the
 * information pertaining to chunk for the stage to
 * function should be added here
 */

type RxJobInfo struct {
	AbsoluteFileName string
	FileDir          string
	Chunksize        uint32
	FileSize         uint64
	Key              []byte
	JobPrio          int
	JobStrictPrio    int
}

/**
 * An TxJobInfo struct flows through the pipeline
 * with an incoming chunk. It has all the required
 * information for each chunk to process the request
 * and send the chunk forward to the next stage
 * In future if some additional stage is added all the
 * information pertaining to chunk for the stage to
 * function should be added here
 */

type TxJobInfo struct {
	Dest             []infra.CsID
	AbsoluteFileName string
	Chunksize        uint64
	HeaderSpace      uint64
	WriteOffset      uint64
	FileSize         uint64
	Key              []byte
	Chunkdir         string
	JobPrio          uint32
	JobStrictPrio    uint32
}

/**
 * An RxJobReqResp structure flows through the
 * pipeline.PrcessReqResp.data field through the pipeline
 * All stages perform its operation on the associated buffer
 * The result of the operation on the req should be propagated
 * through respChan interface. This should generally be done
 * by the final stage in case of a successful flow through
 * the pipeline or by a stage that has encountered an
 * error in processing this chunk
 */
type RxJobReqResp struct {
	JobInfo  *RxJobInfo
	cfo      ChunkInfo
	buffer   []byte
	respChan *RespChanPoolEntry
}

/**
 * An TxJobReqResp structure flows through the
 * pipeline.PrcessReqResp.data field through the pipeline
 * All stages perform its operation on the associated buffer
 * The result of the operation on the req should be propagated
 * through respChan interface. This should generally be done
 * by the final stage in case of a successful flow through
 * the pipeline or by a stage that has encountered an
 * error in processing this chunk
 */
type TxJobReqResp struct {
	JobInfo  *TxJobInfo
	cfo      ChunkInfo
	buffer   []byte
	respChan *RespChanPoolEntry
}

/*
 * For each chunk information received at thsi stage
 * from Ionos Forwarder, a response channel is allocated
 * from a response channel pool and passed along with the
 * chunk across the pipeline so that, any errors or success
 * can be conveyed by it
 */
type RespChanPoolEntry struct {
	channel      chan interface{}
	poolTableIdx uint32
	free         bool
	jobId        uint32
}

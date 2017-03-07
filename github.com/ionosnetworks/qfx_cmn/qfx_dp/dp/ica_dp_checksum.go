package dp

import (
	"encoding/binary"
	"fmt"
	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/infra"
	p "github.com/ionosnetworks/qfx_dp/pipeline"
	"hash/crc64"
	"os"
)

var crcTable *crc64.Table

type Checksum struct{}

const (
	ICA_DP_CHUNKSUM_HEADER_SIZE      = 17
	ICA_DP_CHUNKSUM_PAYLOAD_LEN_SIZE = 8
	ICA_DP_CHUNKSUM_VERSION_SIZE     = 1
)

func (f *Checksum) Init() bool {
	log.Info(ctx, "Checksum initialized, Table created", nil)
	return true
}

func calculateCrc64(buffer []byte) (cksum uint64) {
	crcTable := crc64.MakeTable(crc64.ECMA)
	cksum = crc64.Checksum(buffer, crcTable)
	return
}

func slapChecksumHeader(headroom []byte, sum64 uint64,
	payloadLength uint64, version byte) {
	headroom[0] = version
	log.Debug(ctx, "", blog.Fields{"payloadLength": payloadLength})
	var i uint32
	for i = 1; i < 9; i++ {
		headroom[i] = byte((payloadLength &
			(0xff << ((i - 1) * 8))) >> ((i - 1) * 8))
	}
	for i = 9; i < 17; i++ {
		headroom[i] = byte((sum64 &
			(0xff << ((i - 9) * 8))) >> ((i - 9) * 8))
	}

}

func readChunk(req *RxJobReqResp) (syncID uint32, err error) {
	os.Chmod(req.cfo.ChunkPath, 0777)
	fi, err := os.Open(req.cfo.ChunkPath)
	if err != nil {
		log.Err(ctx, "open failed", blog.Fields{"Error": err})
		return
	}
	f, _ := fi.Stat()
	size := f.Size()
	req.buffer = make([]byte, size)
	_, err = fi.Read(req.buffer)
	if err != nil {
		log.Err(ctx, "read failed", blog.Fields{"Error": err})
		return
	}
	fi.Close()

	// XXX Offset the outer Header
	headerLen := req.buffer[1:9]
	num := binary.LittleEndian.Uint64(headerLen)
	// fmt.Println("Bytes of header left to read: ", num)

	//Read Sync ID
	id := req.buffer[10:14]
	syncID = binary.LittleEndian.Uint32(id)
	// fmt.Println("Sync ID : ", syncID)

	// Read Chunk ID
	var cID [16]byte
	for i := 0; i < 16; i++ {
		cID[i] = req.buffer[14+i]
	}
	var chunkID infra.UUID = cID
	fmt.Println("ChunkID: ", chunkID.String())

	req.buffer = req.buffer[num+9:]
	req.cfo.UUID = chunkID

	/* See if file needs to be deleted here */
	return
}

func (f *Checksum) Process(name string, req *p.ProcessReqResp) bool {
	if req.Interests[req.CurrentInterest] == "chunksum" {
		/* Its a tx job, Create the crc64 */
		/* create the checksum and update the buffer */
		// txreq := req.Data.(*TxJobReqResp)
		txreq := req.Data.(*JobReq)
		offset := txreq.writeOffset
		buffer :=
			txreq.buffer[offset : offset+txreq.ChunkActualSize]
		sum64 := calculateCrc64(buffer)
		headroom := txreq.buffer[txreq.writeOffset-
			ICA_DP_CHUNKSUM_HEADER_SIZE:]
		var version byte = 1
		var payloadLength uint64 = txreq.ChunkActualSize
		slapChecksumHeader(headroom, sum64, payloadLength, version)
		txreq.ChunkActualSize += ICA_DP_CHUNKSUM_HEADER_SIZE
		txreq.writeOffset -= ICA_DP_CHUNKSUM_HEADER_SIZE
		return true
	} else if req.Interests[req.CurrentInterest] == "chunkunsum" {
		var rxcsum64 uint64 = 0
		rxreq := req.Data.(*RxJobReqResp)
		/* Read the chunk file */
		/* Read the file and pass on data */
		syncID, err := readChunk(rxreq)
		req.SyncID = syncID
		if err != nil {
			/* Send a failure on response channel */
			pr := new(p.PipelineResp)
			pr.SyncID = req.SyncID
			pr.Req = req
			// pr.UUID = req.UUID
			pr.MsgType = req.MsgType
			pr.Status = CHECK_UNSUM_ERR
			req.RespChan <- *pr
			/*
				pr := new(PipelineResult)
				pr.status = CHECK_UNSUM_ERR
				pr.chunkInfo = rxreq.cfo
				pr.jobId = req.JobId
				pr.poolTableIdx = rxreq.respChan.poolTableIdx
				rxreq.respChan.channel <- *pr
				log.Err(ctx, "Checksum validation failed for chunk",
					blog.Fields{"Chunk Path": rxreq.cfo.ChunkPath})
			*/
			return false
		}
		//XXX looks ugly
		req.UUID = rxreq.cfo.UUID

		/* Check sum header looks like -
		+-------------------------------------------+
		|ver | payload len | checksum | payload     |
		+-------------------------------------------+
		*/
		buffer := rxreq.buffer[ICA_DP_CHUNKSUM_HEADER_SIZE:]
		sum64 := calculateCrc64(buffer)
		cksumheader := rxreq.buffer[0:ICA_DP_CHUNKSUM_HEADER_SIZE]
		offset := ICA_DP_CHUNKSUM_PAYLOAD_LEN_SIZE + 1
		/* Check if we support the version */
		rxcsum64 = uint64(cksumheader[offset])
		rxcsum64 |= uint64(cksumheader[offset+1]) << 8
		rxcsum64 |= uint64(cksumheader[offset+2]) << 16
		rxcsum64 |= uint64(cksumheader[offset+3]) << 24
		rxcsum64 |= uint64(cksumheader[offset+4]) << 32
		rxcsum64 |= uint64(cksumheader[offset+5]) << 40
		rxcsum64 |= uint64(cksumheader[offset+6]) << 48
		rxcsum64 |= uint64(cksumheader[offset+7]) << 56

		if sum64 != rxcsum64 {
			/* Send a failure on response channel */
			pr := new(p.PipelineResp)
			pr.SyncID = req.SyncID
			pr.UUID = req.UUID
			pr.Req = req
			pr.MsgType = req.MsgType
			pr.Status = CHECK_UNSUM_ERR
			req.RespChan <- *pr
			/*
				pr := new(PipelineResult)
				pr.status = CHECK_UNSUM_ERR
				pr.chunkInfo = rxreq.cfo
				pr.jobId = req.JobId
				pr.poolTableIdx = rxreq.respChan.poolTableIdx
				rxreq.respChan.channel <- *pr
				log.Err(ctx, "Checksum validation failed for chunk",
					blog.Fields{"ChunkPath": rxreq.cfo.ChunkPath})
			*/
			return false
		} else {
			rxreq.buffer = rxreq.buffer[ICA_DP_CHUNKSUM_HEADER_SIZE:]
			return true
		}
	}
	return false
}

func (f *Checksum) HeaderSpace() uint64 {
	/* 1 byte version + 8 byte length + 8 byte checksum */
	return 17
}

func (f *Checksum) Exit() {
	log.Info(ctx, "Checksum state exited", nil)
}

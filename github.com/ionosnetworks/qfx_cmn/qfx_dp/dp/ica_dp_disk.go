/**
 * Disk serializer module -
 * Interests - "CREATE_CHUNK", "STICH_CHUNK"
 * Needs - Two input channels each for each interest
 **/

package dp

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path"
	"runtime/debug"

	"github.com/ionosnetworks/qfx_cmn/blog"
	p "github.com/ionosnetworks/qfx_dp/pipeline"
	"github.com/ionosnetworks/qfx_dp/qfsync"
)

const (
	ICA_DP_DISC_HEADER_SIZE = 13
	MAX_PAYLOAD_SIZE        = 0x2000000
)

type DiscSer struct{}

func stichChunk2(rxjobreq *RxJobReqResp, syncID uint32, decrypt bool) uint32 {
	sr := qfsync.GetSyncRel(syncID)
	// Remove the header; We dont need any of the header fields now.
	var payloadLength uint64 = 0
	var i uint64 = 0
	for i = 1; i < 9; i++ {
		payloadLength |= uint64(rxjobreq.buffer[i]) << ((i - 1) * 8)
	}

	// Guard check
	if payloadLength >= MAX_PAYLOAD_SIZE {
		log.Warn(ctx, "payload length exceeds Ignoring",
			blog.Fields{"Max": MAX_PAYLOAD_SIZE,
				"job ID": syncID})
		return FAILURE
	}

	//	log.Debug(ctx, "In Stitch payload length=",
	// payloadLength, "jobid:", jobId, "chunk:", chunkNum)
	jobletInfoLenSlice := rxjobreq.buffer[9:11]
	var jobletInfoLen uint16
	buf := bytes.NewReader(jobletInfoLenSlice)
	err := binary.Read(buf, binary.LittleEndian, &jobletInfoLen)
	if err != nil {
		log.Err(ctx, "error in binary read",
			blog.Fields{"jobid:": syncID}) //, "chunk:": chunkNum})
		return FAILURE
	}
	jobletCountSlice := rxjobreq.buffer[11:13]
	var jobletCountLen uint16
	buf1 := bytes.NewReader(jobletCountSlice)
	err2 := binary.Read(buf1, binary.LittleEndian, &jobletCountLen)

	if err2 != nil {
		log.Err(ctx, "error in binary read",
			blog.Fields{"jobid:": syncID}) //, "chunk:": chunkNum})
		return FAILURE
	}
	jcisSlice :=
		rxjobreq.buffer[ICA_DP_DISC_HEADER_SIZE : ICA_DP_DISC_HEADER_SIZE+
			jobletInfoLen]

	jcis := make([]JobletChunkInfo, jobletCountLen)
	// log.Debug(ctx, "jcis buffer=", jcisSlice, "jobid:", jobId, "chunk:", chunkNum)
	dec := gob.NewDecoder(bytes.NewBuffer(jcisSlice))
	err = dec.Decode(&jcis)

	var ii int
	var payLoadStartOffset uint64
	payLoadStartOffset = uint64(ICA_DP_DISC_HEADER_SIZE + jobletInfoLen)
	payloadLength = payloadLength - uint64(jobletInfoLen)
	buffer :=
		rxjobreq.buffer[payLoadStartOffset : uint64(payLoadStartOffset)+
			payloadLength]

	payLoadStartOffset = 0
	var nr int
	for ii = 0; ii < len(jcis); ii++ {
		if jcis[ii].JobletCorrupt {
			log.Info(ctx, "Ignoring corrupt joblet",
				blog.Fields{"JobletID": jcis[ii].JobletId,
					"jobid": syncID}) //, "chunk": chunkNum})
			continue
		}

		jobletTempName, err := sr.GetFileName(int64(jcis[ii].JobletId))
		if err != nil {
			log.Err(ctx, "Get File Name returned error",
				blog.Fields{"Err": err.Error()})
			return FAILURE
		}

		jobletDir := path.Dir(jobletTempName)
		if _, err = os.Stat(jobletDir); err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(jobletDir, 0777)
				if err != nil {
					log.Err(ctx, "Error when creating Dir",
						blog.Fields{"JobletDir": jobletDir,
							"jobid:": syncID}) //,
					//	"chunk:": chunkNum})
					return FAILURE
				}
			} else {
				log.Err(ctx, "Dir Exists",
					blog.Fields{"Joblet Dir": jobletDir})
			}
		}

		fo, err := os.OpenFile(jobletTempName, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Err(ctx, "couldn't open successfully",
				blog.Fields{"File": jobletTempName, "jobid:": syncID}) //,
			// "chunk:": chunkNum})
			return FILE_ACCESS_ERR
		} else {
			log.Debug(ctx, "file name opened",
				blog.Fields{
					"fname": jobletTempName,
				})
		}
		for k := 0; k < len(jcis[ii].Mod); k++ {
			fileDataLen := jcis[ii].Mod[k].EndOffset -
				jcis[ii].Mod[k].StartOffset
			bufferStartOffset := payLoadStartOffset
			bufferEndOffset := payLoadStartOffset + fileDataLen
			log.Debug(ctx, "Reading from joblet",
				blog.Fields{"Joblet": ii, "Mod": k,
					"BufferOffsetStart ": bufferStartOffset,
					"BufferEndOffset":    bufferEndOffset,
					"FileDatalen":        fileDataLen,
					"ModStartOff":        jcis[ii].Mod[k].StartOffset})
			fbuf := buffer[bufferStartOffset:bufferEndOffset]
			nw, err := fo.WriteAt(fbuf, int64(jcis[ii].Mod[k].StartOffset))
			if err != nil {
				log.Err(ctx, "Disk Write Error.", blog.Fields{"err=": err})
				fo.Close()
				return DISK_WRITE_ERR
			}
			payLoadStartOffset += fileDataLen
			nr += nw
			log.Debug(ctx, "Written to file", blog.Fields{"NumBytes": nw,
				"JobId": syncID, // "chunk": chunkNum,
				"Fname": jobletTempName})
		}
		fo.Close()
	}
	rxjobreq.buffer = nil
	debug.FreeOSMemory()
	return SUCCESS
}

func getStartingJobletIdAndOffset(js *JobReq,
	chunkNum uint32) (uint32, uint32, uint64, string) {

	var totalSizeSoFar uint64
	totalSizeSoFar = 0

	chunkSize := js.chunkSize
	file_offset := uint64(uint64(chunkNum-1) * uint64(chunkSize))

	var i uint32
	for i = 0; i < uint32(len(js.Joblets)); i++ {
		_, err := os.Stat(js.Joblets[i].JobletFileName)
		if js.Joblets[i].ActionType == "DEL" {
			continue
		}
		if err != nil && os.IsNotExist(err) {
			log.Err(ctx, "File Doesn't Exist",
				blog.Fields{"Joblet": js.Joblets[i].JobletFileName})
		}
		if js.Joblets[i].IsDir == true || js.Joblets[i].FileSize == 0 {
			//dirs and zero size files are just created on destination
			//they are not sent in a chunk
			continue
		}
		var k uint32
		var jobletSize uint64 = 0
		log.Debug(ctx, "Before entering jobletMods ",
			blog.Fields{"ChunkNum": chunkNum, "fileOffset": file_offset,
				"jobletSize": jobletSize, "SizeSoFar": totalSizeSoFar,
				"Current joblet":            i,
				"Len of the mods in joblet": len(js.Joblets[i].Mod)})
		for k = 0; k < uint32(len(js.Joblets[i].Mod)); k++ {
			jobletSize += js.Joblets[i].Mod[k].Size
			if (jobletSize + totalSizeSoFar) > file_offset {
				return i, k, file_offset - totalSizeSoFar, "nil"
			}
			totalSizeSoFar = totalSizeSoFar + jobletSize
			log.Info(ctx, "Adding one more joblet Mod",
				blog.Fields{"ChunkNum": chunkNum, "JobletNum": i,
					"JobletModNum": k})
		}
	}
	return 0, 0, 0, "Error"
}

type JobletChunkInfo struct {
	JobletId      uint32
	Mod           []JobletMod
	JobletCorrupt bool
	Forder        uint32
}

func buildChunkJobletHeader(js *JobReq,
	startingJobletId uint32,
	subId uint32,
	jobletOffset uint64,
	jcis []JobletChunkInfo,
	chunkNum uint32) []JobletChunkInfo {

	log.Debug(ctx, "In buildChunkJobletHeader", nil)
	chunkSize := js.chunkSize
	var i uint32
	var numJobletsInChunk uint32
	var numJobletModsInJoblets uint32
	numJobletsInChunk = 0
	remainingChunk := chunkSize
	for i = startingJobletId; i < uint32(len(js.Joblets)); i++ {
		if js.Joblets[i].ActionType == "DEL" {
			continue
		}

		jobletSize := js.Joblets[i].FileSize
		if js.Joblets[i].IsDir == true || jobletSize == 0 {
			//dirs and zero size files are just created on destination
			//they are not sent in a chunk
			continue
		}
		jobletName := js.Joblets[i].JobletFileName
		modTime := js.Joblets[i].ModTime

		jcis[numJobletsInChunk].JobletCorrupt = js.Joblets[i].Corrupt

		if js.Joblets[i].Corrupt ||
			!isFileGood(jobletName, false, jobletSize, modTime) {
			jcis[numJobletsInChunk].JobletCorrupt = true
		}
		jcis[numJobletsInChunk].JobletId = js.Joblets[i].JobletId
		jcis[numJobletsInChunk].Forder = i
		jcis[numJobletsInChunk].Mod = make([]JobletMod, len(js.Joblets[i].Mod))
		numJobletModsInJoblets = 0
		for k := subId; k < uint32(len(js.Joblets[i].Mod)); k++ {
			jobletModSize := js.Joblets[i].Mod[k].Size
			log.Debug(ctx, "Details", blog.Fields{"ChunkNum": chunkNum,
				"JobletID": i, "JobletModId": k, "modSize": jobletModSize})
			jcis[numJobletsInChunk].Mod[numJobletModsInJoblets].StartOffset =
				js.Joblets[i].Mod[k].StartOffset + jobletOffset
			if jobletModSize-jobletOffset >= uint64(remainingChunk) {
				jcis[numJobletsInChunk].Mod[numJobletModsInJoblets].EndOffset =
					jcis[numJobletsInChunk].Mod[numJobletModsInJoblets].StartOffset +
						uint64(remainingChunk)
				numJobletModsInJoblets++
				jcis[numJobletsInChunk].Mod = jcis[numJobletsInChunk].Mod[0:numJobletModsInJoblets]
				numJobletsInChunk++
				goto end
			} else {
				jcis[numJobletsInChunk].Mod[numJobletModsInJoblets].EndOffset =
					js.Joblets[i].Mod[k].StartOffset + jobletModSize
				remainingChunk = remainingChunk -
					(jobletModSize - jobletOffset)
				numJobletModsInJoblets++
				jobletOffset = 0
			}
		}
		subId = 0
		numJobletsInChunk++
	}
end:
	log.Debug(ctx, "End of build chunk header",
		blog.Fields{"chunkNum": chunkNum, "Num Joblet": numJobletsInChunk,
			"JobletModId": numJobletModsInJoblets})
	return jcis[0:numJobletsInChunk]
}

func readChunkBuffer2(txjobreq *JobReq, chunkNum uint32,
	jobId uint32, encrypt bool) (uint32, []JobletChunkInfo) {
	jobletId, subId, offsetInJoblet, errstr :=
		getStartingJobletIdAndOffset(txjobreq, chunkNum)
	if errstr != "nil" {
		log.Err(ctx, "Starting joblet id and offset not found",
			blog.Fields{"jobId=": jobId, " chunkNum=": chunkNum})
		return NO_SRC_FILE_ERR, nil
	}

	jcis := make([]JobletChunkInfo, len(txjobreq.Joblets))
	jcis = buildChunkJobletHeader(txjobreq, jobletId, subId,
		offsetInJoblet, jcis, chunkNum)
	if jcis == nil {
		return FAILURE_AND_MORE, nil
	}
	numJobletsInChunk := uint16(len(jcis))
	if numJobletsInChunk == 0 {
		return FAILURE_AND_MORE, nil
	}

	var ii int
	validJobletinChunk := false
	corruptJobletinChunk := false
	for ii = 0; ii < len(jcis); ii++ {
		if jcis[ii].JobletCorrupt == false {
			validJobletinChunk = true
		} else {
			if !txjobreq.Joblets[jcis[ii].JobletId].Corrupt {
				corruptJobletinChunk = true
				txjobreq.Joblets[jcis[ii].JobletId].Corrupt =
					jcis[ii].JobletCorrupt
			}
		}
	}
	if corruptJobletinChunk {
		log.Info(ctx, "Atleast one new joblet got corrupted",
			blog.Fields{"jobId": jobId, "chunkNum": chunkNum})
	}

	if !validJobletinChunk {
		log.Err(ctx, "Return point 1. No valid joblet in",
			blog.Fields{"chunkNum": chunkNum, "JobId": jobId})
		return FAILURE_AND_MORE, jcis
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(jcis)
	jobletInfoLen := buf.Len()

	var buf1 bytes.Buffer
	err = binary.Write(&buf1, binary.LittleEndian, uint16(jobletInfoLen))
	var buf2 bytes.Buffer
	err = binary.Write(&buf2, binary.LittleEndian, numJobletsInChunk)

	rem := (jobletInfoLen) % aes.BlockSize
	bufferSize := txjobreq.chunkSize +
		txjobreq.headerSpace +
		uint64(jobletInfoLen) + uint64(rem)
	txjobreq.buffer = make([]byte, bufferSize)

	endoffset := uint64(bufferSize) -
		(txjobreq.headerSpace - txjobreq.writeOffset)

	jobletInfoEndOffset := txjobreq.writeOffset + uint64(jobletInfoLen)
	log.Debug(ctx, "", blog.Fields{"BufferSize=": bufferSize,
		"endOFfset=": endoffset, "jlet endoffset=": jobletInfoEndOffset})

	jobletInfoBuffer := txjobreq.buffer[txjobreq.writeOffset:jobletInfoEndOffset]

	numCopied := copy(jobletInfoBuffer, buf.Bytes())
	log.Debug(ctx, "",
		blog.Fields{"bytes copied into jobletInfoBuffer=": numCopied,
			"JobId": jobId, "chunk": chunkNum})

	log.Debug(ctx, "", blog.Fields{"jobletInfoBuffer=": jobletInfoBuffer,
		"len=": len(jobletInfoBuffer), "JobId": jobId, "chunk:": chunkNum})

	WriteOffsetStart := txjobreq.writeOffset + uint64(jobletInfoLen)
	log.Debug(ctx, "Write offset start", blog.Fields{"Offset": WriteOffsetStart,
		"Jobid:": jobId, "chunk:": chunkNum})
	buffer := txjobreq.buffer[WriteOffsetStart:endoffset]
	var totalRead uint64
	totalRead = 0
	for ii = 0; ii < len(jcis); ii++ {
		if jcis[ii].JobletCorrupt {
			log.Info(ctx, "Ignoring corrupt joblet",
				blog.Fields{"Joblet ID": jcis[ii].JobletId,
					"jobid:": jobId, "chunk:": chunkNum})
			continue
		}
		jobletName := txjobreq.Joblets[jcis[ii].Forder].JobletFileName
		jobletSize := txjobreq.Joblets[jcis[ii].Forder].FileSize
		jobletModTime := txjobreq.Joblets[jcis[ii].Forder].ModTime
		if isFileGood(jobletName, false, jobletSize, jobletModTime) {
			log.Debug(ctx, "Trying to Open Joblet",
				blog.Fields{"Name": jobletName, "JobID:": jobId,
					"chunk:": chunkNum})
			fd, err := os.Open(jobletName)
			if err != nil {
				log.Debug(ctx, "Couldn't open file",
					blog.Fields{"Name": jobletName, "err=": err,
						"jobid:": jobId, "chunk:": chunkNum})
				jcis[ii].JobletCorrupt = true
				continue
			}
			for k := 0; k < len(jcis[ii].Mod); k++ {
				endoffset = WriteOffsetStart +
					(jcis[ii].Mod[k].EndOffset -
						jcis[ii].Mod[k].StartOffset)
				log.Debug(ctx, "Reading from joblet",
					blog.Fields{"Joblet": ii, "Mod": k,
						"WriteOffsetStart ": WriteOffsetStart,
						"EndOffset":         endoffset,
						"BufferSize":        bufferSize,
						"ModStartOff":       jcis[ii].Mod[k].StartOffset})
				buffer = txjobreq.buffer[WriteOffsetStart:endoffset]
				numRead, err := fd.ReadAt(buffer,
					int64(jcis[ii].Mod[k].StartOffset))
				if err != nil {
					if err != io.EOF {
						log.Err(ctx,
							"Couldn't read full chunk buffer",
							blog.Fields{"Error": err,
								"JobId": jobId,
								"Chunk": chunkNum})
						fd.Close()
						return FILE_CORRUPT_ERR, nil
					}
				}

				totalRead += uint64(numRead)
				log.Debug(ctx, "", blog.Fields{"totRead 1=": totalRead,
					"chunk":  chunkNum,
					"Joblet": ii, "Mod": k})
				WriteOffsetStart = endoffset
			}
			fd.Close()
		} else {
			log.Debug(ctx, "File found to be corrupt just before reading:",
				blog.Fields{"name": jobletName, "jobid:": jobId,
					"chunk:": chunkNum})
			jcis[ii].JobletCorrupt = true
			continue
		}
	}
	validJobletinChunk = false
	newCorruptJobletinChunk := false
	for ii = 0; ii < len(jcis); ii++ {
		if jcis[ii].JobletCorrupt == false {
			validJobletinChunk = true
		} else {
			if !txjobreq.Joblets[jcis[ii].Forder].Corrupt {
				newCorruptJobletinChunk = true
				txjobreq.Joblets[jcis[ii].Forder].Corrupt =
					jcis[ii].JobletCorrupt
			}
		}
	}
	err = enc.Encode(jcis)
	jobletInfoLen = buf.Len()
	numCopied = copy(jobletInfoBuffer, buf.Bytes())
	jobletInfoLen = numCopied

	if !validJobletinChunk {
		log.Err(ctx, "No valid joblet", blog.Fields{"chunkNum": chunkNum,
			"jobid":  jobId,
			"chunk:": chunkNum})
		return FAILURE_AND_MORE, jcis
	}
	/* Put in the header for this stage here */
	var version byte = 1
	var payloadLength uint64 = uint64(totalRead) + uint64(numCopied)
	// log.Debug(ctx, "payloadLength=", payloadLength)
	headroom := txjobreq.buffer[txjobreq.writeOffset-
		ICA_DP_DISC_HEADER_SIZE:]
	headroom[0] = version
	var i uint32
	for i = 1; i < 9; i++ {
		headroom[i] = byte((payloadLength &
			(0xff << ((i - 1) * 8))) >> ((i - 1) * 8))
	}
	jobletLengthRoom := headroom[9:11]
	buf1.Read(jobletLengthRoom)
	jobletCountRoom := headroom[11:13]
	buf2.Read(jobletCountRoom)

	txjobreq.ChunkActualSize = ICA_DP_DISC_HEADER_SIZE + payloadLength
	// log.Debug(ctx, "Chunk Actual Size=", txjobreq.cfo.ChunkActualSize)
	if err != nil {
		if err != io.EOF {
			log.Err(ctx, "", blog.Fields{"Err": err.Error()})
			return FAILURE, nil
		}
	}
	// log.Debug(ctx, "Read chunk ", txjobreq.cfo.ChunkPath)
	txjobreq.writeOffset -= ICA_DP_DISC_HEADER_SIZE
	//	log.Debug(ctx, "New WriteOffset after Disc Read Stage=",
	// txjobreq.cfo.WriteOffset, "jobid:", jobId, "chunk:", chunkNum)

	if corruptJobletinChunk || newCorruptJobletinChunk {
		return SUCCESS_AND_MORE, jcis
	}
	return SUCCESS, jcis
}

func (ds *DiscSer) Init() bool {
	log.Info(ctx, "Disc serializer initialized", nil)
	return true
}
func (ds *DiscSer) Process(name string, req *p.ProcessReqResp) bool {

	/* I can do a buffer read or a write -
	 * Let me ask the request what it wants
	 */
	var ok uint32
	if req.Interests[req.CurrentInterest] == "stich" {
		rxjobreq := req.Data.(*RxJobReqResp)
		log.Info(ctx, "",
			blog.Fields{"Prev stage:": req.Interests[req.CurrentInterest-1]})
		var decrypt bool
		if req.Interests[req.CurrentInterest-1] == "decrypt" {
			decrypt = true
		} else {
			decrypt = false
		}

		ok = stichChunk2(rxjobreq, uint32(req.SyncID), decrypt)

		/* I am supposed to inform the first sender of
		 * this chunk the status of the operation
		 */
		fmt.Println(" Stitch done status:", ok)
		pr := new(p.PipelineResp)
		pr.SyncID = req.SyncID
		pr.UUID = req.UUID
		pr.MsgType = req.MsgType
		pr.Status = ok
		pr.Req = req
		req.RespChan <- *pr
		/*
			pr := new(PipelineResult)
			pr.status = ok
			pr.chunkInfo = rxjobreq.cfo
			pr.jobId = req.JobId
			pr.poolTableIdx = rxjobreq.respChan.poolTableIdx
			rxjobreq.respChan.channel <- *pr
		*/
	} else if req.Interests[req.CurrentInterest] == "chunk" {
		// txJobReq := req.Data.(*TxJobReqResp)
		txJobReq := req.Data.(*JobReq)
		log.Debug(ctx, "In dp disc", nil)
		var encrypt bool
		log.Debug(ctx, "",
			blog.Fields{"Next stage:": req.Interests[req.CurrentInterest+1]})
		if req.Interests[req.CurrentInterest+1] == "encrypt" {
			encrypt = true
		} else {
			encrypt = false
		}
		ret, jcis := readChunkBuffer2(txJobReq, req.ChunkNum, req.SyncID, encrypt)
		// TODO Need to have some resiliency feature here
		// map[req.UUID] = jcis
		// rslnc.SaveState(req.SyncID, req.UUID, jcis)
		jcis = jcis
		ok = ret
		if ok != SUCCESS {
			//send failure msg
			pr := new(p.PipelineResp)
			pr.SyncID = req.SyncID
			pr.UUID = req.UUID
			pr.MsgType = req.MsgType
			pr.Status = ok
			pr.Req = req
			req.RespChan <- *pr
			/*
				pr := new(PipelineResult)
				pr.status = ok
				pr.chunkInfo = txJobReq.cfo
				pr.jobId = req.JobId
				pr.poolTableIdx = txJobReq.respChan.poolTableIdx
				if jcis != nil && len(jcis) >= 1 {
					pr.chunkInfo.StartJobletIndex =
					int32(jcis[0].JobletId)
					pr.chunkInfo.EndJobletIndex =
					int32(jcis[len(jcis)-1].JobletId)
				} else {
					pr.chunkInfo.StartJobletIndex = -1
					pr.chunkInfo.EndJobletIndex = -1
				}
				txJobReq.respChan.channel <- *pr
			*/
		}
	}
	if ok == SUCCESS || ok == SUCCESS_AND_MORE {
		log.Debug(ctx, "Returning true from readChunkBuffer2/ stitchchunk", nil)
		return true
	}
	log.Debug(ctx, "Returning false from readChunkBuffer2/ stitchchunk", nil)
	return false
}

func (ds *DiscSer) HeaderSpace() uint64 {
	/* 1 byte for version ,8 bytes for length and
	 * 2 bytes for jobletInfo length, 2 bytes for joblet count
	 */
	return 13
}

func (ds *DiscSer) Exit() {
	log.Info(ctx, "Disc serializer exited", nil)
}

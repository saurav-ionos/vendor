package dp

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/infra"
	p "github.com/ionosnetworks/qfx_dp/pipeline"
)

const (
	ICA_FWDER_SOCK_NAME = "/tmp/unix-test-socket"
)

type Forwarder struct{}

type TemporaryError interface {
	Temporary() bool
}

func (f *Forwarder) Init() bool {
	log.Info(ctx, "Forwarder initialized", nil)
	return true
}
func slapOuterHeader(headroom []byte, uuid infra.UUID,
	length uint64, byteSlice []byte,
	prio uint64,
	syncID uint32, currentHop byte,
	index uint64,
	keyLast bool) {
	var last byte
	var i uint32

	var offset uint32 = 0
	headroom[offset] = 1 //version: 1
	offset += 1

	var j uint32 = 0
	for i = offset; i < offset+8; i++ {
		headroom[i] = byte((length &
			(0xff << ((i - offset) * 8))) >> ((i - offset) * 8))
	}
	offset += 8

	headroom[offset] = currentHop
	offset += 1

	// fmt.Println("Sync ID : ", syncID)
	for i = offset; i < offset+4; i++ {
		headroom[i] = byte((syncID &
			(0xff << ((i - offset) * 8))) >> ((i - offset) * 8))
	}
	offset += 4

	j = 0
	for i = offset; i < offset+16; i++ {
		headroom[i] = uuid[j]
		j++
	}
	offset += 16

	j = 0
	for i = offset; i < offset+uint32(len(byteSlice)); i++ {
		headroom[i] = byteSlice[j]
		j++
	}
	offset += uint32(len(byteSlice))

	for i = offset; i < offset+8; i++ {
		headroom[i] = byte((prio &
			(0xff << ((i - offset) * 8))) >> ((i - offset) * 8))
	}
	offset += 8

	for i = offset; i < offset+8; i++ {
		headroom[i] = byte((index &
			(0xff << ((i - offset) * 8))) >> ((i - offset) * 8))
	}
	offset += 8

	if keyLast {
		last = 1
	} else {
		last = 0
	}
	headroom[offset] = last
}

func (f *Forwarder) Process(name string, req *p.ProcessReqResp) bool {
	// txreq := req.Data.(*TxJobReqResp)
	txreq := req.Data.(*JobReq)
	uuid := req.UUID
	tchunkpath := fmt.Sprintf("%s/.%s.partial",
		txreq.chunkDir,
		uuid.String())

	nr := 0
	var ok bool = true

	fo, err := os.OpenFile(tchunkpath,
		os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Err(ctx, "Error creating file", blog.Fields{"Err": err})
		ok = false
	}
	if ok {
		// offset := txreq.cfo.WriteOffset
		var prio uint64 = uint64(req.Prio)
		var keyIndex uint64 = 1
		keyLast := false
		buffer := &bytes.Buffer{}

		gob.NewEncoder(buffer).Encode(txreq.dest)
		byteSlice := buffer.Bytes()
		// fmt.Printf("%q\n", byteSlice)
		length := uint64(5 + 16 + len(byteSlice) + 8 + 8 + 1)
		/*
			fmt.Println("UUID : ", uuid)
			fmt.Println("Buffer length: ", len(txreq.buffer))
			fmt.Println("Length of header: ", length+8)
			fmt.Println("Diff in length: ", txreq.writeOffset-length-8)
			fmt.Println("Witeoffset ", txreq.writeOffset)
		*/
		headroom := txreq.buffer[txreq.writeOffset-length-9:]
		currentHop := byte(0)
		slapOuterHeader(headroom, uuid, length,
			byteSlice, prio,
			req.SyncID, currentHop,
			keyIndex, keyLast)
		txreq.ChunkActualSize += length + 8 + 1 //Field indicating oH len
		txreq.writeOffset -= length + 8 + 1
		backToStringSlice := []infra.CsID{}
		gob.NewDecoder(buffer).Decode(&backToStringSlice)
		// fmt.Printf("Destinations: %v\n", backToStringSlice)

		nr, err = fo.Write(txreq.buffer[0:txreq.ChunkActualSize])
		fo.Close()
		if uint64(nr) != txreq.ChunkActualSize {
			ok = false
			log.Err(ctx, "Short Write: ", blog.Fields{"Bytes": nr,
				" Error": err})
		}
		chunkpath := fmt.Sprintf("%s/%s.partial",
			txreq.chunkDir,
			uuid.String())
		if ok {
			e := os.Rename(tchunkpath, chunkpath)
			if e != nil {
				log.Err(ctx, "Error renaming ",
					blog.Fields{"Src": tchunkpath,
						"dst": chunkpath, "Err": e})
				ok = false
			} else {
				fmt.Println("Rename successful")
				// SendForwarderMessage(chunkpath)
			}
		}
	}
	//Send status back
	pr := new(p.PipelineResp)
	pr.SyncID = req.SyncID
	pr.UUID = req.UUID
	pr.Req = req
	pr.MsgType = req.MsgType
	if ok {
		pr.Status = SUCCESS
	} else {
		pr.Status = FAILURE
	}
	req.RespChan <- *pr
	txreq.buffer = nil
	debug.FreeOSMemory()
	return ok
}

func (f *Forwarder) HeaderSpace() uint64 {
	return 0
}

func (f *Forwarder) Exit() {
	log.Info(ctx, "Forwarder exited", nil)
}

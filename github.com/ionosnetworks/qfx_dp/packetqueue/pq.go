package pktqueue

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
)

const (
	dataTypeChunk uint32 = 0x00000001
	dataTypeMsg   uint32 = 0x00000002

	packetTypeData uint16 = 0xaaaa
	packetTypeAck  uint16 = 0xbbbb
)

var (
	localCpeId string
)

type qdata struct {
	cpackets map[string][]byte //The packet queue
	mpackets map[string][]byte //The packet queue
	cTokens  chan struct{}     // Chunk writers block when this channel is full
	mTokens  chan struct{}     // Message writers block when this channel is full
	cw       io.Writer         // Where to write chunk to
	mw       io.Writer         // Where to write message to
	r        io.Reader         // Where to read data from
	blocking bool              //Should the sender block on queue full?
	sync.Mutex
}

type PqCtx struct {
	packetType uint16
	uuid       string
	syncID     uint32
	srcCpeID   string
	destCpeID  string
}

type Queue interface {
	InsertChunk(data []byte, destCpeID string, syncID uint32) (*PqCtx, error)
	InsertMsg(data []byte, destCpeID string, syncID uint32) (*PqCtx, error)
	Ack(pqc *PqCtx) error
}

func init() {
	data, err := ioutil.ReadFile("/etc/ionos-cpeid.conf")
	if err == nil {
		localCpeId = string(data)
	} else {
		panic(err)
	}
}

func genUUID() string {
	f, _ := os.Open("/dev/urandom")
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func New(qSize uint32, chunkWriter io.Writer,
	msgWriter io.Writer, r io.Reader, blocking bool) Queue {
	p := new(qdata)
	p.cTokens = make(chan struct{}, qSize)
	p.mTokens = make(chan struct{}, qSize)
	p.cpackets = make(map[string][]byte, qSize)
	p.mpackets = make(map[string][]byte, qSize)
	p.cw = chunkWriter
	p.mw = msgWriter
	p.r = r
	p.blocking = blocking
	return p

}

func (q *qdata) insertToQueue(v []byte, typeOfData uint32,
	destCpeId string, syncID uint32) (*PqCtx, error) {

	var tokens chan struct{}
	var queue map[string][]byte
	var w io.Writer
	if typeOfData == dataTypeChunk {
		tokens = q.cTokens
		queue = q.cpackets
		w = q.cw
	} else {
		tokens = q.mTokens
		queue = q.mpackets
		w = q.mw
	}
	// Try writing to the chunk channel
	tokens <- struct{}{}
	// Write successful hence we would update the map
	pqctx := &PqCtx{
		packetType: packetTypeData,
		uuid:       genUUID(),
		syncID:     syncID,
		srcCpeID:   "",
		destCpeID:  destCpeId,
	}
	var b bytes.Buffer
	err := binary.Write(&b, binary.LittleEndian, pqctx)
	if err != nil {
		return nil, err
	}
	buf := append(b.Bytes(), v...)
	toSend := len(buf)
	written := 0
	for toSend > 0 {
		n, e := w.Write(buf[written:])
		if e != nil {
			return nil, e
		}
		written += n
		toSend -= n
	}
	q.Lock()
	queue[pqctx.uuid] = buf
	q.Unlock()
	return pqctx, nil
}

func (q *qdata) InsertChunk(v []byte, destCpeID string,
	syncID uint32) (*PqCtx, error) {

	return q.insertToQueue(v, dataTypeChunk, destCpeID, syncID)
}
func (q *qdata) InsertMsg(v []byte, destCpeID string,
	syncID uint32) (*PqCtx, error) {

	return q.insertToQueue(v, dataTypeMsg, destCpeID, syncID)
}

func (q *qdata) Ack(ctx *PqCtx) error {
	ackctx := &PqCtx{
		packetType: packetTypeAck,
		uuid:       ctx.uuid, //Holds the uuid which it is acking
		syncID:     ctx.syncID,
		srcCpeID:   localCpeId,
		destCpeID:  ctx.srcCpeID,
	}
	err := binary.Write(q.mw, binary.LittleEndian, ackctx)
	return err
}

func ParsePqCtx(b []byte) (*PqCtx, []byte) {
	ctx := new(PqCtx)
	buf := bytes.NewBuffer(b)
	_ = binary.Read(buf, binary.LittleEndian, ctx)
	return ctx, buf.Bytes()
}

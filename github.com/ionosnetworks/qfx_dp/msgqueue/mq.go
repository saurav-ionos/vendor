package msgqueue

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ionosnetworks/qfx_dp/infra"
)

const (
	packetTypeAck uint8 = 0x01
	packetTypeMsg uint8 = 0x02
)

type qEntry struct {
	data       []byte
	retryLimit int
	delay      int
	replyChan  chan error
	retry      bool
}

type Mq struct {
	txtoken   chan struct{}
	txbuf     map[infra.UUID]*qEntry
	notifyMap map[infra.UUID]chan struct{}
	txbuflock sync.Mutex
	rxbuf     chan []byte
	w         io.Writer // Where to write the msg to
	r         io.Reader // Where to read the msg from
	txChan    chan *qEntry
	rxChan    chan []byte
	delay     int
}

// The message queue header.
type MqCtx struct {
	SrcCsID    string // Source of the message
	PacketType uint8
	Uuid       infra.UUID // UUID of the packet
}

func New(txSize, rxSize int, w io.Writer, r io.Reader) *Mq {

	m := new(Mq)
	m.txtoken = make(chan struct{}, txSize)
	m.txbuf = make(map[infra.UUID]*qEntry, txSize)
	m.notifyMap = make(map[infra.UUID]chan struct{})
	m.rxbuf = make(chan []byte, rxSize)
	m.w = w
	m.r = r
	m.txChan = make(chan *qEntry)
	m.rxChan = make(chan []byte)
	go m.keepSending()
	go m.keepReceiving()
	//go m.keepResending()
	return m
}

func (m *Mq) Destroy() {
	close(m.rxbuf)
	close(m.rxChan)
}

func (m *Mq) keepResending() {
	for {
		time.Sleep(time.Minute * 2)
		for _, val := range m.txbuf {
			if val.retryLimit == 0 {
				// return error to the caller
				//val.replyChan <- errors.New("Retry limit reached")
			} else {
				val.retry = true
				m.txChan <- val
			}
		}
	}
}

func (m *Mq) keepSending() {
	for x := range m.txChan {
		_, err := m.w.Write(x.data)
		x.retryLimit--
		x.replyChan <- err
	}
}

func (m *Mq) Consume(buf []byte) {
	m.rxChan <- buf
}

func (m *Mq) keepReceiving() {

	var mc MqCtx
	for x := range m.rxChan {
		b := bytes.NewBuffer(x)
		dec := gob.NewDecoder(b)
		e := dec.Decode(&mc)
		if e != nil {
			fmt.Println("Error while decoding", e)
			continue
		}
		if mc.PacketType == packetTypeAck {
			// Look up the uuid in the txbuf
			var ch chan struct{}
			var ok bool
			m.txbuflock.Lock()
			_, ok = m.txbuf[mc.Uuid]
			if ok {
				// free up a token
				<-m.txtoken
				ch, ok = m.notifyMap[mc.Uuid]
				delete(m.txbuf, mc.Uuid)
			}
			m.txbuflock.Unlock()
			// notify the sender about the ack
			go func() {
				if ok && ch != nil {
					select {
					case ch <- struct{}{}:
					default:
					}
				}
			}()
		} else if mc.PacketType == packetTypeMsg {
			// Send an ack message.
			mc.PacketType = packetTypeAck
			ack := new(bytes.Buffer)
			enc := gob.NewEncoder(ack)
			_ = enc.Encode(mc)
			qe := &qEntry{
				data:  ack.Bytes(),
				delay: 0, retryLimit: 1,
				replyChan: make(chan error),
			}
			// Let us try sending data to the destination
			m.txChan <- qe
			go func() {
				// ignore if any error. packet will be resent
				<-qe.replyChan
			}()

			//	m.Write(ack.Bytes())
			// Also push the buffer for processing
			m.rxbuf <- b.Bytes()
		}
	}
}

func (m *Mq) Write(payload []byte) (int, error) {
	return m.WriteAndNotifyOnAck(payload, nil)
}

func (m *Mq) WriteAndNotifyOnAck(payload []byte, ch chan struct{}) (int, error) {
	// Define a new msg context
	mc := MqCtx{
		PacketType: packetTypeMsg,
		Uuid:       infra.GenUUID(),
		SrcCsID:    infra.GetLocalCsID().String(),
	}
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	e := enc.Encode(mc)
	//e := binary.Write(&b, binary.LittleEndian, mc)
	if e != nil {
		return 0, e
	}
	buf := b.Bytes()
	// buf has the final payload to send
	buf = append(buf, payload...)

	// Is there any space left in the txbuf?? It shall block
	// if there is no space
	m.txtoken <- struct{}{}

	// We have a token so we should update the map and
	// retransit in case of failure
	qe := &qEntry{
		data:  buf,
		delay: 0, retryLimit: 3,
		replyChan: make(chan error),
	}
	m.txbuflock.Lock()
	m.txbuf[mc.Uuid] = qe
	if ch != nil {
		m.notifyMap[mc.Uuid] = ch
	}
	m.txbuflock.Unlock()

	// Let us try sending data to the destination
	m.txChan <- qe
	return len(payload), <-qe.replyChan
}

func (m *Mq) Read(buf []byte) (int, error) {
	b := <-m.rxbuf
	copy(buf, b)
	return len(b), nil
}

func GetSourceCsID(payload []byte) {
}

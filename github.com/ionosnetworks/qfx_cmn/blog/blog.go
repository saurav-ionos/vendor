//go:generate stringer -type=LogLevel
package blog

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"sync"
	"time"

	fb "github.com/google/flatbuffers/go"
	lgcmn "github.com/ionosnetworks/qfx_cmn/logsvc/common"
	auth "github.com/ionosnetworks/qfx_cmn/logsvc/fb/auth"
)

type blogger struct {
	outChan   chan *Entry
	prevEntry *Entry
	L         LogLevel
	entryPool sync.Pool
	reconn    chan struct{}
	done      chan struct{}
	out       io.WriteCloser
	enc       *gob.Encoder
	Lock      *sync.Mutex
}
type Entry struct {
	logger *blogger
	Data   Fields
	T      time.Time
	L      LogLevel
	Count  int
}

type LogLevel int

const loggerMaxChanLen int = 1024

const (
	Crit  LogLevel = iota
	Warn  LogLevel = iota
	Err   LogLevel = iota
	Info  LogLevel = iota
	Debug LogLevel = iota
)

type Fields map[string]interface{}

type Logger interface {
	Crit(ctx string, msg string, f Fields)
	Warn(ctx string, msg string, f Fields)
	Info(ctx string, msg string, f Fields)
	Err(ctx string, msg string, f Fields)
	Debug(ctx string, msg string, f Fields)
	CritS(ctx string, msg string)
	WarnS(ctx string, msg string)
	InfoS(ctx string, msg string)
	ErrS(ctx string, msg string)
	DebugS(ctx string, msg string)
	SetLevel(level LogLevel)
	Close()
}

func (bl *blogger) blog(ctx string, msg string, f Fields) *Entry {
	bl.Lock.Lock()
	defer bl.Lock.Unlock()
	entry := bl.newEntry()
	entry.logger = bl
	if f != nil {
		entry.Data = f
	} else {
		entry.Data = make(Fields, 5)
	}
	entry.Data["ctx"] = ctx
	entry.Data["msg"] = msg
	entry.Count = 1
	entry.T = time.Now()
	if bl.prevEntry == nil {
		bl.prevEntry = entry
	} else if bl.isSameAsPrevEntry(entry) {
		(bl.prevEntry.Count)++
		return nil
	} else {
		if bl.prevEntry.Count > 1 {
			bl.outChan <- bl.prevEntry
		}
		bl.prevEntry = entry
	}
	return entry
}

func (bl *blogger) newEntry() *Entry {
	return bl.entryPool.Get().(*Entry)
}

func (bl *blogger) releaseEntry(entry *Entry) {
	bl.entryPool.Put(entry)
}

func authenticateSession(conn net.Conn, accesskey, secret string) error {

	msgHeader := lgcmn.MsgHdr{Version: 1, Magic: 0x10305, MetaSize: 0, MsgType: 1}
	builder := fb.NewBuilder(0)
	nodeId := builder.CreateString("123456")
	nodeName := builder.CreateString("1234567")
	aKey := builder.CreateString(accesskey)
	sec := builder.CreateString(secret)
	auth.AuthmesgStart(builder)
	auth.AuthmesgAddNodeID(builder, nodeId)
	auth.AuthmesgAddNodeName(builder, nodeName)
	auth.AuthmesgAddAccessKey(builder, aKey)
	auth.AuthmesgAddSecret(builder, sec)
	authFB := auth.AuthmesgEnd(builder)
	builder.Finish(authFB)
	buf := builder.FinishedBytes()
	msgHeader.MetaSize = int32(len(buf))
	msgHeadBuf := new(bytes.Buffer)

	err := binary.Write(msgHeadBuf, binary.LittleEndian, &msgHeader)
	if err != nil {
		fmt.Println("err = ", err)
		return err
	}

	//	_, _ = conn.Write(msgHeadBuf.Bytes())
	totalWritten := 0
	for {
		bytesWritten, err := conn.Write(msgHeadBuf.Bytes()[totalWritten:])
		totalWritten += bytesWritten

		if err != nil {
			return err
		}
		if totalWritten == len(msgHeadBuf.Bytes()) {
			break
		}
	}

	// _, _ = conn.Write(buf)
	totalWritten = 0
	for {
		bytesWritten, err := conn.Write(buf[totalWritten:])
		totalWritten += bytesWritten

		if err != nil {
			return err
		}
		if totalWritten == len(buf) {
			break
		}
	}

	// Wait for Auth Ok message from server.
	return nil
}

func connect(logSrvAddr, accesskey, secret string,
	retryCount int) (net.Conn, error) {
	count := 0
	config := &tls.Config{InsecureSkipVerify: true}

	for {
		conn, err := tls.Dial("tcp", logSrvAddr, config)
		if err == nil {
			return conn, err
		} else {
			count++
			if count == retryCount {
				return nil, err
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func logInit(out io.WriteCloser) Logger {
	bl := new(blogger)
	bl.outChan = make(chan *Entry, loggerMaxChanLen)
	bl.L = Debug
	bl.entryPool.New = func() interface{} {
		return &Entry{}
	}
	bl.out = out
	bl.done = make(chan struct{})
	bl.Lock = &sync.Mutex{}
	bl.enc = gob.NewEncoder(out)
	go bl.drainLogs()
	return bl
}

func (bl *blogger) isSameAsPrevEntry(curr *Entry) bool {
	if reflect.DeepEqual(bl.prevEntry.Data, curr.Data) {
		return true
	}
	return false
}

func (bl *blogger) Crit(ctx string, msg string, f Fields) {
	if bl.L >= Crit {
		entry := bl.blog(ctx, msg, f)
		if entry == nil {
			return
		}
		bl.prevEntry.L = Crit
		entry.L = Crit
		select {
		case bl.outChan <- entry:
		default:
		}
	}
}

func (bl *blogger) CritS(ctx string, msg string) {
	bl.Crit(ctx, msg, nil)
}

func (bl *blogger) Warn(ctx string, msg string, f Fields) {
	if bl.L >= Warn {
		entry := bl.blog(ctx, msg, f)
		if entry == nil {
			return
		}
		bl.prevEntry.L = Warn
		entry.L = Warn
		select {
		case bl.outChan <- entry:
		default:
		}
	}
}

func (bl *blogger) WarnS(ctx string, msg string) {
	bl.Warn(ctx, msg, nil)
}

func (bl *blogger) Info(ctx string, msg string, f Fields) {
	if bl.L >= Info {
		entry := bl.blog(ctx, msg, f)
		if entry == nil {
			return
		}
		bl.prevEntry.L = Info
		entry.L = Info
		select {
		case bl.outChan <- entry:
		default:
		}
	}
}

func (bl *blogger) InfoS(ctx string, msg string) {
	bl.Info(ctx, msg, nil)
}

func (bl *blogger) Err(ctx string, msg string, f Fields) {
	if bl.L >= Err {
		entry := bl.blog(ctx, msg, f)
		if entry == nil {
			return
		}
		bl.prevEntry.L = Err
		entry.L = Err
		select {
		case bl.outChan <- entry:
		default:
		}
	}
}

func (bl *blogger) ErrS(ctx string, msg string) {
	bl.Err(ctx, msg, nil)
}

func (bl *blogger) Debug(ctx string, msg string, f Fields) {
	if bl.L >= Debug {
		entry := bl.blog(ctx, msg, f)
		if entry == nil {
			return
		}
		bl.prevEntry.L = Debug
		entry.L = Debug
		select {
		case bl.outChan <- entry:
		default:
		}
	}
}

func (bl *blogger) DebugS(ctx string, msg string) {
	bl.Debug(ctx, msg, nil)
}

func (bl *blogger) SetLevel(level LogLevel) {
	bl.L = level
}

func (bl *blogger) Close() {
	close(bl.outChan)
	<-bl.done
}

func (bl *blogger) drainLogs() {
	//enc := gob.NewEncoder(bl.out)
	for x := range bl.outChan {
		if bl.enc != nil {
			err := bl.enc.Encode(&x)
			if err != nil {
				//bl.reconnect()
				//enc = gob.NewEncoder(bl.out)
				fmt.Fprintln(os.Stderr, "blogger: could not encode", x)
			}
			bl.releaseEntry(x)
		}
	}
	bl.done <- struct{}{}
	bl.out.Close()
}

func (bl *blogger) drainLog(data *Entry) {
	if bl.enc != nil {
		bl.Lock.Lock()
		err := bl.enc.Encode(&data)
		bl.Lock.Unlock()
		if err != nil {
			bl.outChan <- data
			fmt.Fprintln(os.Stderr, "blogger: could not encode", data)
		}
		bl.releaseEntry(data)
	} else {
		bl.outChan <- data
	}

}

func (bl *blogger) drainingLogs() {
	for {
		select {
		case <-bl.reconn:
			<-bl.reconn
			continue
		default:
			if entr, ok := <-bl.outChan; ok {
				bl.drainLog(entr)
			} else {
				bl.done <- struct{}{}
				bl.out.Close()
				break
			}
		}
	}
}

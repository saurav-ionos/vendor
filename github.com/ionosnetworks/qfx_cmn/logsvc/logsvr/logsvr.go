package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog/decode"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	cmn "github.com/ionosnetworks/qfx_cmn/logsvc/common"
	auth "github.com/ionosnetworks/qfx_cmn/logsvc/fb/auth"
	kafkapr "github.com/ionosnetworks/qfx_cmn/msgq/producer"
	kcli "github.com/ionosnetworks/qfx_cp/keysvc/keycli"
)

const (
	FIRST_PKT_EXPECTED_INTERVAL = 10 * time.Second
	SVC_ACCESS_FILE             = "/keys/keyfile"
	SVC_CERT_FILE               = "/keys/lftsvr.crt"
	SVC_KEY_FILE                = "/keys/lftsvr.key"
)

var (
	keyClient *kcli.KeyCli
)

func main() {

	var err error
	logsvr := New()

	key := kr.New(SVC_ACCESS_FILE)
	if keyClient, err = kcli.New(key.Key, key.Secret); err != nil {
		fmt.Println("Failed to intialyze key client")
	}
	fmt.Println("Starting Log Message loop")
	logsvr.Start()
}

func New() *LogSvr {

	var logsvr LogSvr

	logsvr.init()
	return &logsvr
}

func (logsvr *LogSvr) init() {

	logsvr.readConfig()
	logsvr.LogMsgQueue = make(chan LogMsgRequest, logsvr.maxLogQueueSize)
	logsvr.LogMsgCritQueue = make(chan LogMsgRequest, logsvr.maxLogQueueSize)

	cmn.Init()
}

func (logsvr *LogSvr) readConfig() {

	// Check if IP addresses are provided.
	logsvr.Iparray = cmn.GetIPListToBroadcast()

	if port := os.Getenv("LOGSVC_PORT"); port != "" {
		logsvr.port = port
	} else {
		logsvr.port = "8088"
	}

	if maxWorkers := os.Getenv("MAX_WORKERS"); maxWorkers != "" {
		n, err := strconv.Atoi(maxWorkers)
		if err == nil {
			logsvr.maxWorkers = n
		} else {
			logsvr.maxWorkers = 3
		}
	} else {
		logsvr.maxWorkers = 3
	}

	if maxQueueSize := os.Getenv("MAX_QUEUESIZE"); maxQueueSize != "" {
		n, err := strconv.Atoi(maxQueueSize)
		if err == nil {
			logsvr.maxLogQueueSize = n
		} else {
			logsvr.maxLogQueueSize = 10000
		}
	} else {
		logsvr.maxLogQueueSize = 10000
	}

	fmt.Println("LogSvc Listening on Ip : ", logsvr.Iparray)
	fmt.Println("LOGSVC_PORT=", logsvr.port)

	if logPipeLineIP := os.Getenv("LOG_PIPELINE_IP"); logPipeLineIP != "" {
		logsvr.LogPipeLineIP = logPipeLineIP
	} else {
		fmt.Println("LOG PIPELINE IP not available. Sending to localhost")
		logsvr.LogPipeLineIP = "127.0.0.1"
	}
	if logPipeLinePort := os.Getenv("LOG_PIPELINE_PORT"); logPipeLinePort != "" {
		logsvr.LogPipeLinePort = logPipeLinePort
	} else {
		logsvr.LogPipeLinePort = "5044"
	}
	if logPipeLineType := os.Getenv("LOG_PIPELINE_TYPE"); logPipeLineType != "" {
		logsvr.LogPipeLineType = logPipeLineType
	} else {
		logsvr.LogPipeLineType = KAFKA
		logsvr.LogPipeLinePort = "9092"
	}
	if logsvr.LogPipeLineType == KAFKA {
		err := kafkapr.Init("LogSvr_Client", []string{logsvr.LogPipeLineIP + ":" + logsvr.LogPipeLinePort})
		if err != nil {
			panic(err)
		}
	}
}

func GetConnectionToLogPipeLine(logPipeLineIP string) net.Conn {
	logPipeLineConn, err := net.Dial("tcp", logPipeLineIP)
	if err != nil {
		fmt.Println("Not able to connect to LogPipeLine", logPipeLineIP)
		return nil
	}
	return logPipeLineConn
}

func (logsvr *LogSvr) Start() {
	StartLogDispatcher(logsvr.maxWorkers, logsvr.LogMsgCritQueue,
		logsvr.LogPipeLineIP+":"+logsvr.LogPipeLinePort, logsvr.LogPipeLineType, HIGH_PRIORITY)
	StartLogDispatcher(logsvr.maxWorkers, logsvr.LogMsgQueue,
		logsvr.LogPipeLineIP+":"+logsvr.LogPipeLinePort, logsvr.LogPipeLineType, LOW_PRIORITY)

	config := cmn.ConfigTLS(SVC_CERT_FILE, SVC_KEY_FILE)

	ln, err := tls.Listen("tcp", ":"+logsvr.port, config)
	if err != nil {
		// handle error
		fmt.Println("Not able to listen on given port", err)
		os.Exit(1)
	}

	fmt.Println("Listening on ", ln.Addr())
	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			// TODO:: This is a serious error.
			fmt.Println("LogSvr Error while accepting socket connnection. err=", err)
		} else {
			go logsvr.handleConnection(conn)
		}
	}
}

func sendAuthOK(conn net.Conn) error {
	authOkMsgHeader := cmn.MsgHdr{Version: 1, Magic: 0x10305, MetaSize: 0, MsgType: 2}
	msgHeadBuf := new(bytes.Buffer)
	err := binary.Write(msgHeadBuf, binary.LittleEndian, &authOkMsgHeader)
	if err != nil {
		return err
	}
	_, err = conn.Write(msgHeadBuf.Bytes())

	return err
}

//The Listen function creates servers
func (logsvr *LogSvr) handleConnection(conn net.Conn) {

	var pkt auth.Authmesg

	fmt.Printf("Received conn from %q \n", conn.RemoteAddr())
	pktChan := make(chan auth.Authmesg, 2)

	//  We expect auth packet within an interval. if not close the
	//  Connection.

	go cmn.ReadMessagePacket(conn, pktChan)
	select {

	case <-time.After(FIRST_PKT_EXPECTED_INTERVAL):
		fmt.Println("Closing connection as first packet did not arrive in time interval of %d seconds",
			FIRST_PKT_EXPECTED_INTERVAL, conn.RemoteAddr())
		conn.Close()
		return
	case pkt = <-pktChan:
	}

	authOK := ValidateKeys(pkt.AccessKey(), pkt.Secret())
	if !authOK {
		fmt.Println("Closing connection with %s as auth didn't succeed", conn.RemoteAddr())
		conn.Close()
		return
	} else {
		err := sendAuthOK(conn)
		if err != nil {
			fmt.Println("Error while sending AuthOK Msg, err=", err)
			conn.Close()
			return
		}
	}

	var fi = conn
	var fo LogChan
	fo.channel = logsvr.LogMsgQueue
	var encoder decode.Encoder = decode.NewTextEncoder(fo)

	var fo2 LogChan
	fo2.channel = logsvr.LogMsgCritQueue
	var highPrioEncoder decode.Encoder = decode.NewTextEncoder(fo2)

	var emap map[string]decode.Encoder = make(map[string]decode.Encoder)

	emap["Debug"] = encoder
	emap["Info"] = encoder
	emap["Err"] = highPrioEncoder
	emap["Crit"] = highPrioEncoder
	emap["Warn"] = highPrioEncoder

	dec := decode.NewXcoder(fi, emap)

	for {
		err := dec.Xcode()
		if err != nil {
			fmt.Println("Error decoding msg: Breaking Now")
			break
		}
	}
	fmt.Println("LogSvr Closing Connection to", conn.RemoteAddr())
	conn.Close()
}

func ValidateKeys(pktAccessKey []byte, pktSecret []byte) bool {

	return keyClient.ValidateFeatureRequest(string(pktAccessKey), string(pktSecret), "log")
}

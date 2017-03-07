package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"time"

	fb "github.com/google/flatbuffers/go"
	cmn "github.com/ionosnetworks/qfx_cmn/logsvc/common"
	auth "github.com/ionosnetworks/qfx_cmn/logsvc/fb/auth"
)

func main() {

	accesskey := "test"
	secret := "test"
	serverport := "8080"
	serveraddress := "127.0.0.1"
	msgHeader := cmn.MsgHdr{Version: 1, Magic: 0x10305, MetaSize: 0, MsgType: 1}

	//client := os.Args[1]
	//glMsgSvr := os.Args[2]

	fmt.Println("Usage :: ClientId MsgSvrAddr DstID ")
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
	config := &tls.Config{InsecureSkipVerify: true}

	conn, err := tls.Dial("tcp", serveraddress+":"+serverport, config)
	msgHeadBuf := new(bytes.Buffer)
	err = binary.Write(msgHeadBuf, binary.LittleEndian, &msgHeader)
	if err != nil {
		fmt.Println("err = ", err)
	}
	n, err := conn.Write(msgHeadBuf.Bytes())
	fmt.Println("Msg Header bytes written: %d", n)
	n, err = conn.Write(buf)
	fmt.Println("FB payload bytes written: %d", n)
	time.Sleep(2 * time.Second)
}

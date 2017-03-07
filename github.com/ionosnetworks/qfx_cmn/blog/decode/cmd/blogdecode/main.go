package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog/decode"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	cmn "github.com/ionosnetworks/qfx_cmn/logsvc/common"
	"github.com/ionosnetworks/qfx_cmn/logsvc/fb/auth"
	kcli "github.com/ionosnetworks/qfx_cp/keysvc/keycli"
)

const (
	FIRST_PKT_EXPECTED_INTERVAL = 10 * time.Second
	SVC_KEY_FILE                = "./keyfile"
)

var (
	keycli *kcli.KeyCli
	key    kr.AccessKey
)

/*
   Usage :: ./blogdecode port
*/

func main() {

	var fo io.WriteCloser = os.Stdout
	var err error
	var l net.Listener

	port := 0
	key = kr.New(SVC_KEY_FILE)

	if keycli, err = kcli.New(key.Key, key.Secret); err != nil {
		fmt.Println("Failed to intialize key server ", err.Error())
	}

	for {
		if len(os.Args) > 1 {
			port, err = strconv.Atoi(os.Args[1])
			if err == nil {
				// We will listen on port specified and drain
				// logs on the accepted socket

				config := cmn.ConfigTLS("logsvr.crt", "logsvr.key")

				l, err = tls.Listen("tcp4", "0.0.0.0:"+os.Args[1], config)
				fmt.Println("Listening on port ", port, " for logs")

				if err != nil {
					panic(err)
				}
			}
		} else {
			fmt.Println("Specify port")
		}

		for {
			if port > 0 {
				conn, err := l.Accept()
				if err != nil {
					panic(err)
				}
				go handleConnection(conn, fo)
			}

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
func handleConnection(conn net.Conn, fo io.WriteCloser) {

	var pkt auth.Authmesg

	fmt.Printf("Received conn from %q \n", conn.RemoteAddr())
	pktChan := make(chan auth.Authmesg, 2)

	//  We expect auth packet within an interval. if not close the
	//  Connection.
	cmn.Init()
	go cmn.ReadMessagePacket(conn, pktChan)
	select {

	case <-time.After(FIRST_PKT_EXPECTED_INTERVAL):
		fmt.Println("Closing connection as first packet did not arrive in time interval of %d seconds",
			FIRST_PKT_EXPECTED_INTERVAL, conn.RemoteAddr())
		conn.Close()
		return
	case pkt = <-pktChan:
	}

	authOK := ValidateKeys(pkt.Secret(), pkt.AccessKey())
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
	var encoder decode.Encoder = decode.NewTextEncoder(fo)
	var emap map[string]decode.Encoder = make(map[string]decode.Encoder)

	emap["Debug"] = encoder
	emap["Info"] = encoder
	emap["Err"] = encoder
	emap["Crit"] = encoder
	emap["Warn"] = encoder

	dec := decode.NewXcoder(fi, emap)

	for {
		err := dec.Xcode()
		if err != nil {
			fmt.Println("Error decoding msg: Breaking Now", err.Error())
			break
		}
	}
	fmt.Println("LogSvr Closing Connection to", conn.RemoteAddr())
	conn.Close()
}

//TODO
func ValidateKeys(pktSecret []byte, pktAccessKey []byte) bool {

	// fmt.Println("Key received ", string(pktAccessKey), string(pktSecret))

	if keycli.ValidateFeatureRequest(string(pktAccessKey), string(pktSecret), "log") {
		return true
	}

	return true
}

package common

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	cnts "github.com/ionosnetworks/qfx_cp/qfxConsts"
)

var (
	msgHeader  MsgHdr
	msgHdrSize int
	ctx        string
	logger     blog.Logger
)

func SetLoggerParams(context string, log blog.Logger) {
	ctx = context
	logger = log
}

func SetGlobalHeader(header MsgHdr) {
	msgHeader = header

	bin_buf := &bytes.Buffer{}
	err := binary.Write(bin_buf, binary.LittleEndian, msgHeader)
	if err != nil {
		panic(err)
	}
	msgHdrSize = len(bin_buf.Bytes())
	fmt.Println("Msg svc Header size ", msgHdrSize)
}

func ConfigTLS(capath, key string) *tls.Config {

	cer, err := tls.LoadX509KeyPair(capath, key)
	if err != nil {
		logger.Crit(ctx, "Failed to load certificates", blog.Fields{"err": err.Error()})
		return nil
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}

	config.Rand = rand.Reader
	config.MinVersion = tls.VersionTLS10
	config.SessionTicketsDisabled = false
	config.InsecureSkipVerify = false
	config.ClientAuth = tls.NoClientCert
	config.PreferServerCipherSuites = true
	config.ClientSessionCache = tls.NewLRUClientSessionCache(1000)

	return config
}

func TcpConnect(accessPoint string, retryCount int) (error, net.Conn) {

	count := 0
	config := &tls.Config{InsecureSkipVerify: true}

	for {

		conn, err := tls.Dial("tcp", accessPoint, config)
		if err == nil {
			return err, conn
		} else {
			count++
			if count == retryCount {
				return err, nil
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func ReadMessagePacket(netconn MsgNetConnection, ch chan MsgPkt) {

	var MesgHeader MsgHdr
	var MesgMetaHeader MsgMetaHdr

	if netconn.Conn == nil {
		ch <- MsgPkt{AppErr: errors.New("use of closed network connection"),
			Err: errors.New("use of closed network connection")}
		return
	}

	msgHeadBytes := make([]byte, msgHdrSize)

	_, err := io.ReadFull(netconn.Conn, msgHeadBytes)

	if err != nil {
		ch <- MsgPkt{AppErr: errors.New("Header read fail"), Err: err}
		return
	}
	bufreader := bytes.NewReader(msgHeadBytes)
	err = binary.Read(bufreader, binary.LittleEndian, &MesgHeader)
	if err != nil {
		panic(err)
	}

	if MesgHeader.MetaSize == 0 {
		ch <- MsgPkt{AppErr: errors.New("Received invalid Meta header size "), Err: err}
		return
	}
	msgMetaBytes := make([]byte, MesgHeader.MetaSize)
	_, err = io.ReadFull(netconn.Conn, msgMetaBytes)

	if err != nil {
		ch <- MsgPkt{AppErr: errors.New("Meta Header read "), Err: err}
		return
	}

	var byteArr bytes.Buffer
	byteArr.Write(msgMetaBytes)
	dec := gob.NewDecoder(&byteArr) // Will read from network.

	// Decode (receive) the value.
	err = dec.Decode(&MesgMetaHeader)
	if err != nil {
		logger.Crit(ctx, "Decode error", blog.Fields{"err": err.Error()})
		ch <- MsgPkt{AppErr: errors.New("Decode error"), Err: err}
		return
	}

	Payload := make([]byte, MesgMetaHeader.PayloadSz)

	if MesgMetaHeader.PayloadSz > cnts.MAX_PAYLOAD_ALLOWED {
		ch <- MsgPkt{AppErr: errors.New("Received invalid Meta header size  "), Err: err}
		return
	}

	sz, err := io.ReadFull(netconn.Conn, Payload)
	if sz != int(MesgMetaHeader.PayloadSz) || err != nil {
		ch <- MsgPkt{AppErr: errors.New("Received incomplete payload"), Err: err}
		return
	}

	ch <- MsgPkt{MesgHeader: &MesgHeader,
		MesgMetaHeader: &MesgMetaHeader, Payload: &Payload, Err: nil, AppErr: nil}
}

func CreateAndSendMessage(netconn MsgNetConnection, source, dest string,
	msgtype, subtype int32, payload []byte) error {

	if netconn.Conn == nil {
		return errors.New("Invalid connection")
	}
	lmsgHeader := msgHeader
	lmsgHeader.ClientTs = time.Now().UTC().UnixNano()
	payloadsz := int32(len(payload))
	metaheader := MsgMetaHdr{Src: source, Dst: dest, MsgType: msgtype,
		MsgSubType: subtype, PayloadSz: payloadsz}

	pkt := MsgPkt{MesgHeader: &lmsgHeader, MesgMetaHeader: &metaheader, Payload: &payload}
	return SendMessagePacket(pkt, netconn)
}

func SendMessagePacket(pkt MsgPkt, netconn MsgNetConnection) error {

	if netconn.Conn == nil {
		return errors.New("Invalid connection")
	}
	bin_buf := &bytes.Buffer{}

	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.

	// Encode (send) the value.
	err := enc.Encode(pkt.MesgMetaHeader)
	if err != nil {
		logger.Crit(ctx, "encode error:", blog.Fields{"err": err.Error()})
		return errors.New("encode error " + err.Error())
	}

	// Assign meta header size to main message header.
	pkt.MesgHeader.MetaSize = int32(len(network.Bytes()))

	// Add Message header
	err = binary.Write(bin_buf, binary.LittleEndian, pkt.MesgHeader)
	if err != nil {
		panic(err)
	}

	// Add Message meta header
	err = binary.Write(bin_buf, binary.LittleEndian, network.Bytes())
	if err != nil {
		panic(err)
	}

	// Add the payload
	_, err = bin_buf.Write([]byte(*(pkt.Payload)))

	totalWritten := 0

	netconn.ConnLock.Lock()
	defer netconn.ConnLock.Unlock()
	for {
		bytesWritten, err := netconn.Conn.Write(bin_buf.Bytes()[totalWritten:])
		totalWritten += bytesWritten

		if err != nil {
			return err
		}
		if totalWritten == len(bin_buf.Bytes()) {
			break
		}
	}
	return nil
}

func GetIntfLocalIP(name string) string {

	intf, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}

	addrs, err := intf.Addrs()
	if err != nil || name != intf.Name {
		return ""
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it

		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {

				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func GetIPListToListen() []string {
	var iparray = make([]string, 0)

	if ipstr := os.Getenv("MSGSVC_IP_TO_USE"); ipstr != "" {
		iparray = strings.Split(ipstr, ",")
	} else {
		ifarray := make([]string, 0)
		// Check if interfaces are provided.

		if iflist := os.Getenv("MSGSVC_IF_TO_USE"); iflist != "" {

			ifarray = strings.Split(iflist, ",")
		} else {
			// Both are not set. Using eth0
			ifarray = append(ifarray, "eth0")
		}
		for i := 0; i < len(ifarray); i++ {

			if ip := GetIntfLocalIP(ifarray[i]); ip != "" {
				iparray = append(iparray, ip)
			}
		}
	}
	return iparray
}

func GetPortForMsgSvc() string {
	if port := os.Getenv("MSGSVC_PORT"); port != "" {
		return port
	}
	return "8080"
}

func GetMsgLBAddress() string {
	if lbAddr := os.Getenv("MSGSVC_LB_ADDR"); lbAddr != "" {
		return lbAddr
	}
	return ""
}

func VerifyAuthInternal(accessKey, accessSecret, source, msgCtrlUUID string, netconn MsgNetConnection) error {

	// Send the Auth packet.
	authpkt := MsgAuthPkt{Key: accessKey, Secret: accessSecret}

	var bin_buf bytes.Buffer

	enc := gob.NewEncoder(&bin_buf)

	enc.Encode(authpkt)

	// Send the packet.
	CreateAndSendMessage(netconn, source, msgCtrlUUID,
		CLIENT_INFO_RESP, CLIENT_INFO_RESP, bin_buf.Bytes())

	var pkt MsgPkt
	pktChan := make(chan MsgPkt, 2)

	ReadMessagePacket(netconn, pktChan)
	pkt = <-pktChan

	return pkt.Err
}

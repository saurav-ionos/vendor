package common

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	auth "github.com/ionosnetworks/qfx_cmn/logsvc/fb/auth"
)

var msgHdrSize = 0

func Init() {

	var msgHeader MsgHdr

	bin_buf := &bytes.Buffer{}
	err := binary.Write(bin_buf, binary.LittleEndian, msgHeader)
	if err != nil {
		panic(err)
	}
	msgHdrSize = len(bin_buf.Bytes())
	fmt.Println("Log service  Header size ", msgHdrSize)
}

func ReadMessagePacket(conn net.Conn, ch chan auth.Authmesg) {

	var MesgHeader MsgHdr

	if conn == nil {
		return
	}

	// msgHeadBytes := make([]byte, unsafe.Sizeof(MesgHeader))
	msgHeadBytes := make([]byte, msgHdrSize)

	_, err := io.ReadFull(conn, msgHeadBytes)

	if err != nil {
		fmt.Println("Header read failed :", err.Error())
		return
	}
	bufreader := bytes.NewReader(msgHeadBytes)
	err = binary.Read(bufreader, binary.LittleEndian, &MesgHeader)
	if err != nil {
		fmt.Println("Header conversion to struct failed err=", err)
		return
	}
	fmt.Println("Printing Struct %+v", MesgHeader)
	payloadBytes := make([]byte, MesgHeader.MetaSize)
	_, err = io.ReadFull(conn, payloadBytes)

	if err != nil {
		fmt.Println("Meta Header read failed :", err.Error())
		return
	}
	authMsg := auth.GetRootAsAuthmesg(payloadBytes, 0)
	fmt.Println("authMsg: ", authMsg.NodeID(), authMsg.NodeName(),
		string(authMsg.Secret()[:]), string(authMsg.AccessKey()[:]))
	ch <- *authMsg
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

func GetIPListToBroadcast() []string {
	var iparray = make([]string, 0)

	if ipstr := os.Getenv("LOGSVC_IP_TO_USE"); ipstr != "" {
		iparray = strings.Split(ipstr, ",")
		fmt.Println("LOGSVC_IP_TO_USE=", iparray)

	} else {
		ifarray := make([]string, 0)
		// Check if interfaces are provided.

		if iflist := os.Getenv("LOGSVC_IF_TO_USE"); iflist != "" {

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
	if port := os.Getenv("LOGSVC_PORT"); port != "" {
		return port
	}
	return "8080"
}

func GetMsgLBAddress() string {
	if lbAddr := os.Getenv("LOGSVC_LB_ADDR"); lbAddr != "" {
		return lbAddr
	}
	return ""
}

func TcpConnect(accessPoint string, retryCount int) (error, net.Conn) {

	count := 0

	config := &tls.Config{InsecureSkipVerify: true}
	for {

		//conn, err := net.Dial("tcp", accessPoint)
		conn, err := tls.Dial("tcp", accessPoint, config)
		if err != nil {
			fmt.Println("Connection failed to " + accessPoint) // handle error
			count++
			if count == retryCount {
				return err, nil
			}
		} else {
			return nil, conn
		}
		time.Sleep(5 * time.Second)
	}
}

func ConfigTLS(capath, key string) *tls.Config {

	cer, err := tls.LoadX509KeyPair(capath, key)
	if err != nil {
		fmt.Println("Failed to load certificates", err)
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

// All the object defentions for Message Server, Client and API
package common

import (
	"net"
	"sync"
)

const (
	ClientLoginTimeout = 60
)

const (
	GLOBAL_CONTROLLER = iota
	LOCAL_CONTROLLER  = iota
	ION_CONTROLLER    = iota
)
const (
	LOCAL_NETWORK_IP   = iota
	PUBLIC_IP          = iota //
	GLOBAL_MSG_CTRL_IP = iota
	LOCAL_MSG_CTRL_IP  = iota
	NAT_IP             = iota // Filled by controller
)

// Message types
const (
	CLIENT_LOGIN       = iota
	CLIENT_LOGOUT      = iota
	CLIENT_MSG         = iota
	CLIENT_INFO_REQ    = iota
	CLIENT_INFO_RESP   = iota
	CLIENT_INFO_UPDATE = iota
	CLIENT_AUTH        = iota
	MSG_CTRL_LOGIN     = iota
	MSG_CTRL_LOGOUT    = iota
	MSG_CTRL_MESSAGE   = iota
)

type MsgHdr struct {
	Version  int32
	Magic    int32
	MetaSize int32
	Filler   int32

	// For debugging....
	ClientTs   int64 // Client sending
	CntrlRcvTs int64 // First controller switching...
	CntrlDstTs int64 // Controller writing to client..
	DestTs     int64 // Client Recv..
}

// This will be encoded using gbuf
type MsgMetaHdr struct {
	Src        string
	Dst        string
	PayloadSz  int32
	MsgType    int32
	MsgSubType int32
}

type MsgAuthPkt struct {
	Key    string
	Secret string
}

type MsgPkt struct {
	MesgHeader     *MsgHdr
	MesgMetaHeader *MsgMetaHdr
	Payload        *[]byte
	Err            error
	AppErr         error
}
type MsgNetConnection struct {
	ConnLock *sync.Mutex
	Conn     net.Conn
}

type ClientAccessPoint struct {
	AccessPoint string // Could be DNS:port or IPaddr:port
	Type        int    // Type of access point. Could be, Client Local address,
	// NAT/PAT address of client, Local MsgServer Address, Global MsgServer.
}

type ClientAPList struct {
	ApList []ClientAccessPoint
}

type ClientIPResponse struct {
	Access []ClientAccessPoint
	Status error
}

type MsgCtrlLoginReq struct {
	MsgCtrlType int
	IpAddr      string
	Port        string
	MsgCtrlUUID string
}

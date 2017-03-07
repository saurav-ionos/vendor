package main

import (
	"sync"

	cmn "github.com/ionosnetworks/qfx_cmn/msgsvc/common"
)

// Structure to keep track of all client connection
type MsgClients struct {
	netConn cmn.MsgNetConnection
	valid   bool
}

// Structure to keep track of all Controller connections.
type MsgController struct {
	IpAddr      string
	msgCtrlUUID string
	netConn     cmn.MsgNetConnection
	valid       bool
}

type MsgSvr struct {
	// This list is for all the controllers that have logged in.
	msgRxCtrlList     map[string]MsgController
	msgRxCtrlListLock *sync.RWMutex

	// This list is for all the controllers that this controller has logged into.
	msgTxCtrlList     map[string]MsgController
	msgTxCtrlListLock *sync.RWMutex

	// This list is to cache all the clients and its controller information.
	msgCliCtrlList     map[string]string
	msgCliCtrlListLock *sync.RWMutex

	// This list is for all the clients that are logged into this controller
	msgCliList     map[string]MsgClients
	msgCliListLock *sync.RWMutex

	msgCtrlUUID        string
	port               string
	msgCtrlListenIp    []string
	msgCtrlBroadcastIp string

	msgCtrlRole    string
	parentCtrlAP   string
	parentCtrlUUID string
	msgLbAddr      string

	EtcdIP   string
	authPack cmn.MsgAuthPkt
}

type MsgCtrlStats struct {
	clientCount      int
	authFailSessions int
}

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	cmn "github.com/ionosnetworks/qfx_cmn/msgsvc/common"
	kcli "github.com/ionosnetworks/qfx_cp/keysvc/keycli"
	cnts "github.com/ionosnetworks/qfx_cp/qfxConsts"
)

const (
	SVC_ACCESS_FILE  = "/keys/keyfile"
	MSGSVR_CERT_FILE = "/keys/lftsvr.crt"
	MSGSVR_KEY_FILE  = "/keys/lftsvr.key"
	HEADER_VERSION   = 1
	HEADER_MAGIC     = 0x10135

	FIRST_PKT_EXPECTED_INTERVAL = 1 * time.Minute
	CTRL_STATS_UPDATE_INTERVAL  = 5 * 60 // In Seconds
	MSG_CTRL_HB_INTERVAL        = 60     // In Seconds
	MSGSVR_DEFAULT_PORT         = "8080"
	MSGSVR_CTRL_ROLE_MASTER     = "Master"
	MSGSVR_CTRL_ROLE_SECONDARY  = "Secondary"
	SVC_KEY_FILE                = "/keys/keyfile"
	CPUPROFILE                  = "msgsvrprofile"
)

var (
	key       kr.AccessKey
	ctx       = ""
	logger    blog.Logger
	keyClient *kcli.KeyCli
	profw     io.Writer
)

func main() {
	var err error

	profw, err = os.Create(CPUPROFILE)
	if err != nil {
		fmt.Println("Failed to create profile file", err)
	} else {
		pprof.StartCPUProfile(profw)
		defer pprof.StopCPUProfile()
	}

	key := kr.New(SVC_ACCESS_FILE)
	InitLogger(key.Key, key.Secret)

	if logger != nil {
		defer logger.Close()
	}

	if keyClient, err = kcli.New(key.Key, key.Secret); err != nil {
		fmt.Println("Failed to intialyze key client")
	}

	msgsvr, err := New()
	if err != nil {
		logger.Info(ctx, "Failed to initialise Messag server", blog.Fields{"Err": err.Error()})
		return
	}

	sigUserChan := make(chan os.Signal, 1)
	signal.Notify(sigUserChan, syscall.SIGUSR1)
	go handleSigUser1(sigUserChan)

	fmt.Println("Staring msg server ")
	// Start the message server
	msgsvr.Start()
}

func handleSigUser1(sch chan os.Signal) {
	stop := true
	for {
		<-sch
		if stop {
			logger.Info(ctx, "Stopping CPU profiling", nil)
			pprof.StopCPUProfile()
			stop = false
		} else {
			stop = true
			logger.Info(ctx, "Starting CPU profiling", nil)
			pprof.StartCPUProfile(profw)
		}

	}
	return
}

func New() (*MsgSvr, error) {

	msgsvr := MsgSvr{port: MSGSVR_DEFAULT_PORT,
		msgTxCtrlList:     make(map[string]MsgController),
		msgTxCtrlListLock: &sync.RWMutex{},
		msgRxCtrlList:     make(map[string]MsgController),
		msgRxCtrlListLock: &sync.RWMutex{},

		msgCliCtrlList:     make(map[string]string),
		msgCliCtrlListLock: &sync.RWMutex{},

		msgCliList:         make(map[string]MsgClients),
		msgCliListLock:     &sync.RWMutex{},
		authPack:           cmn.MsgAuthPkt{Key: key.Key, Secret: key.Secret},
		msgCtrlUUID:        "",
		msgCtrlBroadcastIp: "",
		EtcdIP:             "",
		msgCtrlRole:        MSGSVR_CTRL_ROLE_MASTER}

	msgsvr.readConfig()
	ctx += msgsvr.msgCtrlUUID

	cmn.SetLoggerParams(ctx, logger)
	cmn.SetGlobalHeader(cmn.MsgHdr{Version: HEADER_VERSION, Magic: HEADER_MAGIC, MetaSize: 0})

	// if the role is not master, we expect parent not to be NULL.
	if msgsvr.msgCtrlRole != MSGSVR_CTRL_ROLE_MASTER {

		if msgsvr.parentCtrlAP == "" {
			return nil, errors.New("Primary Controller is missing")
		}
		// This is a secondary controller. Login to main controller and keep sending HB
		tgtCtrl := MsgController{IpAddr: msgsvr.parentCtrlAP, msgCtrlUUID: msgsvr.parentCtrlUUID}
		srcCtlr := MsgController{msgCtrlUUID: msgsvr.msgCtrlUUID}

		if err := tgtCtrl.init(&msgsvr); err != nil {
			logger.Crit(ctx, "Main controller Login failed", blog.Fields{"Err": err.Error()})
			return nil, err
		}
		msgsvr.msgTxCtrlList[tgtCtrl.msgCtrlUUID] = tgtCtrl
		srcCtlr.sendLoginData(&msgsvr, &tgtCtrl)
		go srcCtlr.sendHB(&msgsvr, &tgtCtrl, true)
	}
	SetControllerIp(msgsvr.msgCtrlUUID, msgsvr.msgCtrlListenIp, msgsvr.port, MSG_CTRL_HB_INTERVAL*2)
	logger.Info(ctx, "CTRL IP", blog.Fields{"ID": msgsvr.msgCtrlUUID, "ip": GetControllerIp(msgsvr.msgCtrlUUID)})
	go msgsvr.updateConfigAndStats()

	logger.Info(ctx, "MsgSvr running", blog.Fields{"ID": msgsvr.msgCtrlUUID,
		"IP": msgsvr.msgCtrlListenIp[0], "Port": msgsvr.port})

	return &msgsvr, nil
}

/*
  For each periodic task, add a ticker and its action.
*/
func (msgsvr *MsgSvr) updateConfigAndStats() {

	ipUpdateTicker := time.NewTicker(MSG_CTRL_HB_INTERVAL * time.Second)
	statsUpdateTicker := time.NewTicker(CTRL_STATS_UPDATE_INTERVAL * time.Second)

	for {
		select {
		case <-ipUpdateTicker.C:
			// Share the msg controller IP list with others. Lease is set to twice of HB interval.
			SetControllerIp(msgsvr.msgCtrlUUID, msgsvr.msgCtrlListenIp, msgsvr.port, MSG_CTRL_HB_INTERVAL*2)
			logger.Info(ctx, "CTRL IP", blog.Fields{"ID": msgsvr.msgCtrlUUID, "ip": GetControllerIp(msgsvr.msgCtrlUUID)})

		case <-statsUpdateTicker.C:
			//Call the periodic function here.
			logger.Info(ctx, "Sending Stats update to controller ", nil)

			logger.Info(ctx, "Stats ", blog.Fields{"Clients": len(msgsvr.msgCliList),
				"Contrl": len(msgsvr.msgTxCtrlList)})
		}
	}
}

func (msgsvr *MsgSvr) readConfig() {

	// Check if IP addresses are provided.
	msgsvr.msgCtrlListenIp = cmn.GetIPListToListen()

	if val := os.Getenv("MSGSVC_PORT"); val != "" {
		msgsvr.port = val
	}

	if val := os.Getenv("MSGSVC_CTRL_UUID"); val != "" {
		msgsvr.msgCtrlUUID = val
	} else {
		msgsvr.msgCtrlUUID, _ = os.Hostname()
	}

	if val := os.Getenv("MSGSVC_CTRL_ROLE"); val != "" {
		msgsvr.msgCtrlRole = val
	}

	if val := os.Getenv("MSGSVC_CTRL_BROADCAST_IP"); val != "" {
		msgsvr.msgCtrlBroadcastIp = val
	}

	if val := os.Getenv("MSGSVC_CTRL_PARENT"); val != "" {
		msgsvr.parentCtrlAP = val
	}

	if val := os.Getenv("MSGSVC_CTRL_PARENT_ID"); val != "" {
		msgsvr.parentCtrlUUID = val
	}

	if val := os.Getenv("ETCD_CLUSTER_IP"); val != "" {
		msgsvr.EtcdIP = val
	} else {
		logger.Warn(ctx, "Running in single mode", nil)
	}
	InitEtcd(msgsvr.EtcdIP, handleEtcdEvt, msgsvr)

	if lbAddr := os.Getenv("MSGSVC_LB_ADDR"); lbAddr != "" {
		// Default Lb Address.
		msgsvr.msgLbAddr = lbAddr
	}

}

func (msgsvr *MsgSvr) Start() {

	config := cmn.ConfigTLS(MSGSVR_CERT_FILE, MSGSVR_KEY_FILE)

	ln, err := tls.Listen("tcp", ":"+msgsvr.port, config)
	if err != nil {
		// handle error
		logger.Crit(ctx, "Not able to listen on port", blog.Fields{"port": msgsvr.port, "err": err.Error()})
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	// Start the client for Controller messages only if controller is writing messages to kafka.
	if msgsvr.msgCtrlRole == MSGSVR_CTRL_ROLE_MASTER {
		if msgqAddr := os.Getenv("MSGSVC_MSGQ_ADDR"); msgqAddr != "" {
			go msgsvr.ControllerMessageClient(msgqAddr)
		}
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			logger.Crit(ctx, "Server could not accept connection", blog.Fields{"Err": err.Error()})
			return
		}
		netconn := cmn.MsgNetConnection{Conn: conn, ConnLock: &sync.Mutex{}}
		go msgsvr.handleConnection(netconn)
	}
}

func (msgsvr *MsgSvr) handleConnection(netConn cmn.MsgNetConnection) {

	var pkt cmn.MsgPkt

	// logger.Info(ctx, "Connected to ", blog.Fields{"Addr": netConn.Conn.RemoteAddr().String()})

	pktChan := make(chan cmn.MsgPkt, 2)

	//  We expect authorized packet within an interval. if not close the
	// Connection.

	go cmn.ReadMessagePacket(netConn, pktChan)
	select {

	case <-time.After(FIRST_PKT_EXPECTED_INTERVAL):
		logger.Warn(ctx, "Auth Packet timedout. Closing", blog.Fields{"Addr": netConn.Conn.RemoteAddr().String()})
		netConn.Conn.Close()
		return
	case pkt = <-pktChan:
	}

	err := pkt.Err
	if pkt.Err == nil {
		err = pkt.AppErr
	}

	if err != nil {
		logger.Warn(ctx, "Auth Packet Recv Failed", blog.Fields{"Addr": netConn.Conn.RemoteAddr().String(),
			"err": pkt.Err.Error()})
		netConn.Conn.Close()
		return
	}

	if msgsvr.handleClientAuth(pkt, netConn) == false {
		logger.Warn(ctx, "Invalid Auth Packet. Closing",
			blog.Fields{"id": pkt.MesgMetaHeader.Src, "Addr": netConn.Conn.RemoteAddr().String()})
		netConn.Conn.Close()
		return
	}

	msgsvr.readPacketFromClient(netConn)
}

func (msgsvr *MsgSvr) readPacketFromClient(netConn cmn.MsgNetConnection) {
	client := ""
	ctrlr := ""
	dest := ""
	pktCount := 0
	t1 := time.Now()

	pktChan := make(chan cmn.MsgPkt, 2)
	for {

		cmn.ReadMessagePacket(netConn, pktChan)
		pkt := <-pktChan

		err := pkt.Err

		if pkt.Err == nil {
			err = pkt.AppErr
		}

		if err != nil {
			/*
				switch err {

				case io.EOF:
					if client != "" {
						msgsvr.handleClientLogout(client)
					} else if ctrlr != "" {
						msgsvr.handleControllerLogout(ctrlr)
					} else if dest != "" {
						// We would have cached the controller data for this client in the beginning.
						// This connection was for sending data. Cleanup the controller cache.
						msgsvr.deleteClientCtrlCache(dest)
					}

				default:
					if !strings.Contains(pkt.Err.Error(), "use of closed network connection") {

						logger.Debug(ctx, "Incomplete message", blog.Fields{"err": pkt.Err.Error()})
					}
				}
			*/
			if client != "" {
				msgsvr.handleClientLogout(client)
			} else if ctrlr != "" {
				msgsvr.handleControllerLogout(ctrlr)
			} else if dest != "" {
				// We would have cached the controller data for this client in the beginning.
				// This connection was for sending data. Cleanup the controller cache.
				msgsvr.deleteClientCtrlCache(dest)
			}
			logger.Debug(ctx, "Packet Count", blog.Fields{"ID": client, "Count": pktCount})
			netConn.Conn.Close()
			return
		}

		// Log the number of packets exchaged between controllers.
		t2 := time.Now()
		if ctrlr != "" && t2.After(t1.Add(time.Minute)) {

			logger.Debug(ctx, "Packet Count", blog.Fields{"ID": ctrlr, "Count": pktCount})
			t1 = time.Now()
		}

		switch pkt.MesgMetaHeader.MsgType {

		case cmn.CLIENT_MSG:
			dest = pkt.MesgMetaHeader.Dst
			pktCount++
			msgsvr.handleClientMessage(pkt)

		case cmn.CLIENT_LOGIN:
			client = pkt.MesgMetaHeader.Src
			msgsvr.handleClientLogin(pkt.MesgMetaHeader.Src, netConn)

		case cmn.CLIENT_INFO_REQ:
			msgsvr.handleClientInfoReq(pkt, netConn)

		case cmn.CLIENT_INFO_UPDATE:
			msgsvr.handleClientInfoUpdate(pkt, netConn)

		case cmn.CLIENT_LOGOUT:
			msgsvr.handleClientLogout(pkt.MesgMetaHeader.Src)

		case cmn.MSG_CTRL_LOGOUT:
			msgsvr.handleControllerLogout(pkt.MesgMetaHeader.Src)

		case cmn.MSG_CTRL_LOGIN:
			ctrlr = pkt.MesgMetaHeader.Src
			msgsvr.handleControllerLogin(pkt.MesgMetaHeader.Src, netConn)
		}
	}
}

func (msgsvr *MsgSvr) handleClientInfoReq(pkt cmn.MsgPkt, netconn cmn.MsgNetConnection) {

	var apdlist []cmn.ClientAccessPoint
	var bin_buf bytes.Buffer

	dest := string(*pkt.Payload)

	jsonstr := GetClientIp(dest)

	logger.Debug(ctx, "Destination ", blog.Fields{"Dest": dest, "Value": jsonstr})

	jdec := json.NewDecoder(bytes.NewBufferString(jsonstr))
	jdec.Decode(&apdlist)

	enc := gob.NewEncoder(&bin_buf)

	enc.Encode(apdlist)

	// Send the client info response packet.
	cmn.CreateAndSendMessage(netconn, msgsvr.msgCtrlUUID, dest,
		cmn.CLIENT_INFO_RESP, cmn.CLIENT_INFO_RESP, bin_buf.Bytes())
}

func (msgsvr *MsgSvr) handleClientAuth(pkt cmn.MsgPkt, netconn cmn.MsgNetConnection) bool {

	var authpkt cmn.MsgAuthPkt

	dec := gob.NewDecoder(bytes.NewReader(*pkt.Payload))

	// Decode (receive) the value.
	err := dec.Decode(&authpkt)
	if err != nil {
		logger.Info(ctx, "Invalid Auth Packet ", blog.Fields{"Data": authpkt, "err": err.Error()})
		return false
	}
	logger.Debug(ctx, "Auth Packet ", blog.Fields{"Key": authpkt.Key, "Secret": authpkt.Secret})

	if keyClient.ValidateFeatureRequest(authpkt.Key, authpkt.Secret, "msg") == false {
		return false
	}

	// Send the AuthOK packet.

	logger.Debug(ctx, "Auth Packet Sent ", blog.Fields{"Dst": pkt.MesgMetaHeader.Src})
	cmn.CreateAndSendMessage(netconn, msgsvr.msgCtrlUUID, pkt.MesgMetaHeader.Src,
		cmn.CLIENT_AUTH, cmn.CLIENT_AUTH, make([]byte, 0))

	return true
}

func (msgsvr *MsgSvr) handleClientInfoUpdate(pkt cmn.MsgPkt, netConn cmn.MsgNetConnection) {

	var aplist []cmn.ClientAccessPoint

	dec := gob.NewDecoder(bytes.NewReader(*pkt.Payload))

	// Decode (receive) the value.
	err := dec.Decode(&aplist)
	if err != nil {
		logger.Err(ctx, "handleClientInfoUpdate:: decode error:", blog.Fields{"err": err.Error()})
	}

	// Add the IP address of remote connection which can be used if
	// NAT/PAT hole punching works.

	aplist = append(aplist, cmn.ClientAccessPoint{AccessPoint: netConn.Conn.RemoteAddr().String(),
		Type: cmn.NAT_IP})

	var bin_buf bytes.Buffer
	enc := json.NewEncoder(&bin_buf)
	enc.Encode(aplist)

	jsonstr := string(bin_buf.Bytes())

	SetClientIp(pkt.MesgMetaHeader.Src, jsonstr)
}

/*
  Move this to read loop.
	Get the handle for the first packet. Cache the connection. This will reduce ETCD calls.
	if write fails, try getting the connection again.

	Clients can ask for revalidating connections in case of missing ACKS.
	API will try to get new coordinates of client and connect to it. If client has gone offline,
	we will declare node dead.

*/
func (msgsvr *MsgSvr) handleClientMessage(pkt cmn.MsgPkt) {

	// Check whether the packet is for the Management controller.
	// if yes then we need to send this message to kafka bus.

	// Packet is for some other client. Identify the client or corresponding
	// controller for it and send it.

	if pkt.MesgMetaHeader == nil {
		return
	}
	if pkt.MesgMetaHeader.Dst == cnts.GLOBAL_CONTROLLER_UUID {
		SendMessageToController(pkt)
		return
	}

	if netconn, entity := msgsvr.getConnForClient(pkt.MesgMetaHeader.Dst); netconn == nil {
		logger.Info(ctx, "Client not logged in ", blog.Fields{"ID": pkt.MesgMetaHeader.Dst})
	} else {
		if entity == 1 {
			pkt.MesgHeader.CntrlRcvTs = time.Now().UTC().UnixNano()
			pkt.MesgHeader.CntrlDstTs = time.Now().UTC().UnixNano()
		} else { // 0 for clients
			pkt.MesgHeader.CntrlDstTs = time.Now().UTC().UnixNano()
		}

		err := cmn.SendMessagePacket(pkt, *netconn)
		if err != nil {
			// Need to re-evaluate path.
		}
	}
}

func (srcCtlr *MsgController) sendLoginData(msgsvr *MsgSvr, tgtCtlr *MsgController) error {

	var ap cmn.ClientAccessPoint
	var aplist []cmn.ClientAccessPoint

	logger.Debug(ctx, "Sending IP Data", blog.Fields{"Id": srcCtlr.msgCtrlUUID})

	// Add Global Controller ID
	ap.AccessPoint = msgsvr.msgCtrlBroadcastIp + ":" + msgsvr.port
	ap.Type = cmn.LOCAL_CONTROLLER

	aplist = append(aplist, ap)

	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.

	// Encode (send) the value.
	err := enc.Encode(aplist)
	if err != nil {
		logger.Crit(ctx, "Encode Error ", blog.Fields{"err": err.Error()})
		return err
	}

	if err = cmn.CreateAndSendMessage(tgtCtlr.netConn, tgtCtlr.msgCtrlUUID, srcCtlr.msgCtrlUUID,
		cmn.CLIENT_INFO_UPDATE, cmn.CLIENT_INFO_UPDATE, network.Bytes()); err != nil {

		logger.Warn(ctx, "Send Login Info Failed  ", blog.Fields{"err": err.Error()})
	}
	return err
}

func (srcCtlr *MsgController) reconnect(msgsvr *MsgSvr, tgtCtrl *MsgController) error {

	if err := tgtCtrl.init(msgsvr); err != nil {
		logger.Crit(ctx, "Reconnect failed", blog.Fields{"Err": err.Error()})
		return err
	}

	srcCtlr.sendLoginData(msgsvr, tgtCtrl)

	return nil
}

func (srcCtlr *MsgController) sendHB(msgsvr *MsgSvr, tgtCtlr *MsgController, reconnect bool) error {

	payload := make([]byte, 0)

	for {
		logger.Debug(ctx, "Sending HB Data", blog.Fields{"Id": tgtCtlr.IpAddr})
		err := cmn.CreateAndSendMessage(tgtCtlr.netConn, srcCtlr.msgCtrlUUID,
			tgtCtlr.msgCtrlUUID, cmn.MSG_CTRL_LOGIN, cmn.MSG_CTRL_LOGIN, payload)

		if err != nil {

			logger.Warn(ctx, "Send HB Failed  ", blog.Fields{"err": err.Error()})

			if reconnect == true {
				// We connect to controller only if we are secondary.
				srcCtlr.reconnect(msgsvr, tgtCtlr)
			} else {
				delete(msgsvr.msgTxCtrlList, tgtCtlr.msgCtrlUUID)
			}

		}
		time.Sleep(time.Second * MSG_CTRL_HB_INTERVAL)
	}
}

func (msgsvr *MsgSvr) getConnForClient(destid string) (*cmn.MsgNetConnection, int) {

	// Check if this controller has connection with requested destination.

	msgsvr.msgCliListLock.RLock()
	client, found := msgsvr.msgCliList[destid]
	msgsvr.msgCliListLock.RUnlock()

	if found == true {
		return &client.netConn, 0 // "Client"
	}

	// Check if the requested client has logged into any other controller

	// Check if client-controller map has client's controller information.
	destMsgCtrl := ""
	msgsvr.msgCliCtrlListLock.RLock()
	destMsgCtrl, found = msgsvr.msgCliCtrlList[destid]
	msgsvr.msgCliCtrlListLock.RUnlock()

	if found != true {
		if destMsgCtrl = GetControllerForClient(destid); destMsgCtrl == "" {
			// No such client logged in yet.
			return nil, -1 // ""
		} else {
			logger.Debug(ctx, "Caching CTRL", blog.Fields{"CTRL": destMsgCtrl, "Client": destid})
			msgsvr.msgCliCtrlListLock.Lock()
			msgsvr.msgCliCtrlList[destid] = destMsgCtrl
			msgsvr.msgCliCtrlListLock.Unlock()
		}
	}

	//Connection could have been established in another context.
	msgsvr.msgTxCtrlListLock.RLock()
	msgCont, found := msgsvr.msgTxCtrlList[destMsgCtrl]
	msgsvr.msgTxCtrlListLock.RUnlock()

	if found == true {
		return &msgCont.netConn, 1 //  "Cntrl"
	}

	// We need to establish connection with the controller
	msgsvr.msgTxCtrlListLock.Lock()
	defer msgsvr.msgTxCtrlListLock.Unlock()

	msgCont, found = msgsvr.msgTxCtrlList[destMsgCtrl]

	if found == true {
		return &msgCont.netConn, 1 // "Cntrl"
	}

	// Get the IP address of the controller.
	msgCtrlIP := GetControllerIp(destMsgCtrl)

	logger.Debug(ctx, "Logging to controller", blog.Fields{"Id": destMsgCtrl, "IP": msgCtrlIP})

	msgCtrlIpList := strings.Split(msgCtrlIP, " ")

	for i := 0; i < len(msgCtrlIpList); i++ {

		if strings.TrimSpace(msgCtrlIpList[i]) == "" {
			continue
		}
		tgtCtlr := MsgController{IpAddr: msgCtrlIpList[i], msgCtrlUUID: destMsgCtrl}
		srcCtlr := MsgController{msgCtrlUUID: msgsvr.msgCtrlUUID}

		if tgtCtlr.init(msgsvr) == nil {
			msgsvr.msgTxCtrlList[destMsgCtrl] = tgtCtlr

			go srcCtlr.sendHB(msgsvr, &tgtCtlr, false)
			return &tgtCtlr.netConn, 1 //  "Cntrl"
		}
	}

	// Failed to connect to required controller. Return nil.
	return nil, -1 // ""
}

func (msgCtrl *MsgController) init(msgsvr *MsgSvr) error {

	// Connect to the controller
	err, conn := cmn.TcpConnect(msgCtrl.IpAddr, 5)
	if err != nil {
		return err
	}

	// Add to the list of controller
	netconn := cmn.MsgNetConnection{Conn: conn, ConnLock: &sync.Mutex{}}

	err = cmn.VerifyAuthInternal(msgsvr.authPack.Key, msgsvr.authPack.Secret,
		msgsvr.msgCtrlUUID, msgCtrl.IpAddr, netconn)
	if err != nil {
		logger.Err(ctx, "Controller login Auth fail", blog.Fields{"Id": msgCtrl.msgCtrlUUID})
		conn.Close()
		return err
	}
	msgCtrl.netConn = netconn
	msgCtrl.valid = true

	return nil
}

// Client Login
func (msgsvr *MsgSvr) handleClientLogin(clientid string, netConn cmn.MsgNetConnection) {

	msgsvr.msgCliListLock.RLock()
	client, found := msgsvr.msgCliList[clientid]
	msgsvr.msgCliListLock.RUnlock()
	if found == false || client.valid == false {

		logger.Info(ctx, "Client Logged in", blog.Fields{"Id": clientid})

		client.valid = true
		client.netConn = netConn
		msgsvr.msgCliListLock.Lock()
		msgsvr.msgCliList[clientid] = client
		msgsvr.msgCliListLock.Unlock()

		// TestCode :: Send the secondary login controller details.
		// msgsvr.sendSecondaryCtrlDetails(clientid, conn)
	} else {
		logger.Debug(ctx, "Client HB recieved", blog.Fields{"Id": clientid})
	}
	SetControllerForClient(clientid, msgsvr.msgCtrlUUID, cmn.ClientLoginTimeout)
}

//TestCode :: This function will mimic secondary controller login request.
func (msgsvr *MsgSvr) sendSecondaryCtrlDetails(clientid string, netConn cmn.MsgNetConnection) {

	ctrlInfo := cmn.MsgCtrlLoginReq{
		MsgCtrlType: cmn.LOCAL_CONTROLLER, IpAddr: msgsvr.msgCtrlListenIp[0],
		Port: msgsvr.port, MsgCtrlUUID: msgsvr.msgCtrlUUID}

	var bin_buf bytes.Buffer
	enc := gob.NewEncoder(&bin_buf)
	enc.Encode(ctrlInfo)

	cmn.CreateAndSendMessage(netConn, msgsvr.msgCtrlUUID, clientid,
		cmn.MSG_CTRL_LOGIN, cmn.MSG_CTRL_LOGIN, bin_buf.Bytes())
}

func (msgsvr *MsgSvr) handleClientLogout(clientid string) {

	msgsvr.msgCliListLock.Lock()
	_, found := msgsvr.msgCliList[clientid]

	if found == false {
		logger.Warn(ctx, "Invalid client logout received", blog.Fields{"Id": clientid})
	}
	logger.Info(ctx, "Client Logged out", blog.Fields{"Id": clientid})

	DelClientEntry(clientid)
	delete(msgsvr.msgCliList, clientid)
	msgsvr.msgCliListLock.Unlock()

	// Connection from client is lost. We have to close sockets.
	// Mark client as down.
	sendLogoutMessagetoController(clientid)
}

func (msgsvr *MsgSvr) handleControllerLogout(ctrlId string) {

	msgsvr.msgRxCtrlListLock.Lock()
	defer msgsvr.msgRxCtrlListLock.Unlock()

	_, found := msgsvr.msgRxCtrlList[ctrlId]

	if found == false {
		logger.Warn(ctx, "Invalid controller logout received", blog.Fields{"Id": ctrlId})
	}
	logger.Info(ctx, "Controller Logged out", blog.Fields{"Id": ctrlId})

	delete(msgsvr.msgRxCtrlList, ctrlId)
}

func (msgsvr *MsgSvr) handleControllerLogin(controllerid string, netConn cmn.MsgNetConnection) {
	// Other Controller is trying to connect

	msgsvr.msgRxCtrlListLock.RLock()
	controller, found := msgsvr.msgRxCtrlList[controllerid]
	msgsvr.msgRxCtrlListLock.RUnlock()
	if found == false || controller.valid == false {

		logger.Info(ctx, "Controller Logged in", blog.Fields{"Id": controllerid})

		controller.valid = true
		controller.netConn = netConn
		msgsvr.msgRxCtrlListLock.Lock()
		msgsvr.msgRxCtrlList[controllerid] = controller
		msgsvr.msgRxCtrlListLock.Unlock()

	} else {
		logger.Debug(ctx, "Controller HB received", blog.Fields{"Id": controllerid})
	}
}

func handleEtcdEvt(cbkctx interface{}, bdelete bool, key, value string) {

	logger.Debug(ctx, "handleEtcdEvt", blog.Fields{"Del": bdelete, "key": key, "value": value})
	// Check if we have cached.
	// We are taking write lock itself as we expect less frequent changes.
	if msgsvr, ok := cbkctx.(*MsgSvr); ok {
		msgsvr.msgCliCtrlListLock.Lock()

		if _, found := msgsvr.msgCliCtrlList[key]; found == true {

			if bdelete {
				delete(msgsvr.msgCliCtrlList, key)
			} else {
				msgsvr.msgCliCtrlList[key] = value
			}
		}
		msgsvr.msgCliCtrlListLock.Unlock()

	}
}

func (msgsvr *MsgSvr) deleteClientCtrlCache(destid string) {

	logger.Debug(ctx, "Cleaning up CNTLR cache", blog.Fields{"Client": destid})
	msgsvr.msgCliCtrlListLock.Lock()

	delete(msgsvr.msgCliCtrlList, destid)

	msgsvr.msgCliCtrlListLock.Unlock()
}

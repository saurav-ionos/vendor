package msgcli

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	cmn "github.com/ionosnetworks/qfx_cmn/msgsvc/common"
	cnts "github.com/ionosnetworks/qfx_cp/qfxConsts"
)

// Structure to keep track of all Controller connections.
type MsgController struct {
	ipAddr         string
	port           string
	netConn        cmn.MsgNetConnection
	valid          bool
	msgCtrlType    int
	msgcli         *MsgCli
	msgCtrlUUID    string
	source         string
	retryCount     int
	clientInfoChan map[string]chan []cmn.ClientAccessPoint
}

type MsgPkt struct {
	PayLoad    []byte
	Source     string
	ClientTs   int64 // Client sending
	CntrlRcvTs int64 // First controller switching...
	CntrlDstTs int64 // Controller writing to client..
	DestTs     int64 // Client Recv..
}

type MsgCli struct {
	cliUUID         string
	globalMsgCtrl   *MsgController
	localMsgCtrl    map[string]*MsgController
	msgCtrlListLock *sync.RWMutex

	// Keys and secret
	accessKey    string
	accessSecret string
	//MsgRecv      chan *[]byte
	MsgRecv chan MsgPkt
}

type MsgXferClient struct {
	destUUID string
	netConn  cmn.MsgNetConnection
}

type MsgXfer struct {
	msgcli      *MsgCli
	destClients []MsgXferClient
	cliUUID     string
}

const (
	HEADER_VERSION           = 1
	HEADER_MAGIC             = 0x10135
	CLIENTHBINTERVAL         = (cmn.ClientLoginTimeout / 2) * time.Second
	MAXUNREADMESSAGES        = 20
	CTRL_CONNECT_RETRY_COUNT = 5
)

var (
	ctx       = ""
	logger    blog.Logger
	accessKey = ""
	secret    = ""
)

func initLogger() {

	logSvr := "127.0.0.1:2000"
	if val := os.Getenv("LOG_SERVER"); val != "" {
		logSvr = val
	}

	if logger = blog.New(logSvr, accessKey, secret); logger == nil {
		fmt.Println("Failed to initialize logger")
	}

	logLevel := "Debug"
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		logLevel = level
	}

	switch logLevel {
	case "Debug":
		logger.SetLevel(blog.Debug)
	case "Info":
		logger.SetLevel(blog.Info)
	case "Warn":
		logger.SetLevel(blog.Warn)
	case "Err":
		logger.SetLevel(blog.Err)
	case "Crit":
		logger.SetLevel(blog.Crit)
	}
}

// MsgCli related functions.
func New(serverAddr, cliUUID, serverport, accesskey, secret, ctx string, log *blog.Logger) (*MsgCli, error) {

	var mcli MsgCli
	err := mcli.init(serverAddr, cliUUID, serverport, accesskey, secret, ctx, log)

	return &mcli, err
}

func (msgcli *MsgCli) init(serverAddr, cliUUID, serverport, accesskey, secret, contxt string, log *blog.Logger) error {

	msgcli.cliUUID = cliUUID
	msgcli.accessKey = accesskey
	msgcli.accessSecret = secret
	msgcli.localMsgCtrl = make(map[string]*MsgController)
	msgcli.MsgRecv = make(chan MsgPkt, MAXUNREADMESSAGES)
	msgcli.msgCtrlListLock = &sync.RWMutex{}

	if log != nil {
		logger = *log
	}
	initLogger()
	if contxt == "" {
		ctx = os.Args[0]
	} else {
		ctx = contxt
	}

	gctrl := MsgController{msgcli: msgcli, ipAddr: serverAddr, port: serverport,
		source: cliUUID, msgCtrlUUID: "",
		retryCount: CTRL_CONNECT_RETRY_COUNT, msgCtrlType: cmn.GLOBAL_MSG_CTRL_IP}

	msgcli.globalMsgCtrl = &gctrl

	cmn.SetGlobalHeader(cmn.MsgHdr{Version: 1, Magic: 0x10305, MetaSize: 0})

	return msgcli.globalMsgCtrl.init()
}

func (msgcli *MsgCli) Write(ctrlMsgType int32, payload []byte, dstlist ...string) {

	// If the destination list is null, send it to global controller.
	if len(dstlist) == 0 {

		err := cmn.CreateAndSendMessage(msgcli.globalMsgCtrl.netConn, msgcli.cliUUID,
			cnts.GLOBAL_CONTROLLER_UUID, cmn.CLIENT_MSG, ctrlMsgType, payload)

		if err != nil {
			logger.Err(ctx, "Data send failed", blog.Fields{"src": msgcli.cliUUID,
				"dst": cnts.GLOBAL_CONTROLLER_UUID, "err": err.Error()})
		}
		return
	}

	// If the destination is entry is a controller and this client has logged in,
	// send the message to it. if not, send the message to the global controller.
	for _, entry := range dstlist {

		conn := msgcli.globalMsgCtrl.netConn
		if msgctrl, found := msgcli.localMsgCtrl[entry]; found == true {
			conn = msgctrl.netConn
		}
		if err := cmn.CreateAndSendMessage(conn, msgcli.cliUUID,
			cnts.GLOBAL_CONTROLLER_UUID, cmn.CLIENT_MSG, ctrlMsgType, payload); err != nil {
			logger.Err(ctx, "Data send failed", blog.Fields{"src": msgcli.cliUUID,
				"dst": cnts.GLOBAL_CONTROLLER_UUID, "err": err.Error()})
		}
	}
}

func (msgcli *MsgCli) Close() {

	// For all controllers send logout message and close the connection.
	msgcli.msgCtrlListLock.Lock()
	for _, ctrl := range msgcli.localMsgCtrl {
		ctrl.close()
	}
	msgcli.msgCtrlListLock.Unlock()

	msgcli.globalMsgCtrl.close()
}

func (msgcli *MsgCli) InitXfer(dstlist ...string) (*MsgXfer, map[string]error) {
	msgx := MsgXfer{}

	clcount := 0

	dst := make([]string, 0)

	for _, entry := range dstlist {

		if entry = strings.Trim(entry, " "); entry != "" {
			dst = append(dst, entry)
			clcount++
		}
	}

	if msgcli == nil {
		logger.Err(ctx, "InitXfer invoked on NULL ", nil)
		return nil, nil
	}

	if clcount == 0 {
		logger.Err(ctx, "InitXfer invoked with zero destinations ", nil)
		return nil, nil
	}
	errMap := msgx.init(msgcli, dst)
	return &msgx, errMap
}

func (msgcli *MsgCli) GetClientStatus(dstlist ...string) (error, map[string]bool) {

	clcount := 0
	status := false

	statusMap := make(map[string]bool, 0)

	if msgcli == nil {
		logger.Err(ctx, "GetClientStatus invoked on NULL ", nil)
		return errors.New("Client interface NULL"), nil
	}

	for _, cliUUID := range dstlist {

		if cliUUID = strings.Trim(cliUUID, " "); cliUUID != "" {

			msgcli.msgCtrlListLock.RLock()
			status = false
			logger.Debug(ctx, "Using local controllers", nil)
			for _, ctrl := range msgcli.localMsgCtrl {

				status = ctrl.getClientStatus(cliUUID)

				// If error is auth, return here itself.
				if status == true {
					break
				}
			}

			msgcli.msgCtrlListLock.RUnlock()

			if status == true {
				statusMap[cliUUID] = true
				break
			}

			// Check with Global Controller
			status := msgcli.globalMsgCtrl.getClientStatus(cliUUID)
			statusMap[cliUUID] = status
			clcount++
		}
	}

	if clcount == 0 {
		logger.Err(ctx, "GetClientStatus invoked with zero destinations ", nil)
		return errors.New("No destinations"), nil
	}
	return nil, statusMap
}

func (msgxfer *MsgXfer) Revalidate() {
	msgxfer.Close()

	for _, dstentry := range msgxfer.destClients {
		dstentry.init(msgxfer.msgcli)
	}
}

func (msgcli *MsgCli) AddLocalController(ipAddr, port, uuid string, ctrlType int) {
	msgCtrlInfo := cmn.MsgCtrlLoginReq{IpAddr: ipAddr, Port: port,
		MsgCtrlUUID: uuid, MsgCtrlType: ctrlType}

	msgcli.addLocalController(msgCtrlInfo)
}

func (msgcli *MsgCli) addLocalController(msgCtrlInfo cmn.MsgCtrlLoginReq) {

	localCtrl := MsgController{msgcli: msgcli,
		ipAddr: msgCtrlInfo.IpAddr, port: msgCtrlInfo.Port,
		source: msgcli.cliUUID, msgCtrlUUID: msgCtrlInfo.MsgCtrlUUID,
		retryCount: CTRL_CONNECT_RETRY_COUNT, msgCtrlType: msgCtrlInfo.MsgCtrlType}

	localCtrl.init()
	msgcli.msgCtrlListLock.Lock()
	msgcli.localMsgCtrl[msgCtrlInfo.MsgCtrlUUID] = &localCtrl
	msgcli.msgCtrlListLock.Unlock()
}

// MsgXfer related functions.

func (msgx *MsgXfer) init(msgcli *MsgCli, dst []string) map[string]error {

	msgx.destClients = make([]MsgXferClient, len(dst))
	msgx.cliUUID = msgcli.cliUUID
	msgx.msgcli = msgcli

	errMap := make(map[string]error)

	for i := 0; i < len(msgx.destClients); i++ {

		msgx.destClients[i].destUUID = dst[i]
		err := msgx.destClients[i].init(msgcli)
		errMap[dst[i]] = err
	}

	return errMap
}

func (msgx *MsgXfer) Write(payload []byte) {

	for i := 0; i < len(msgx.destClients); i++ {

		if msgx.destClients[i].netConn.Conn == nil {
			continue
		}
		err := cmn.CreateAndSendMessage(msgx.destClients[i].netConn, msgx.cliUUID,
			msgx.destClients[i].destUUID, cmn.CLIENT_MSG, cmn.CLIENT_MSG, payload)

		if err != nil {
			logger.Err(ctx, "Data send failed", blog.Fields{"src": msgx.cliUUID,
				"dst": msgx.destClients[i].destUUID, "err": err.Error()})

			// Try reconnecting once.
			if err = msgx.destClients[i].init(msgx.msgcli); err != nil {
				logger.Err(ctx, "Reconnect failed", blog.Fields{"src": msgx.cliUUID,
					"dst": msgx.destClients[i].destUUID, "err": err.Error()})
			}
		}
	}
}

func (msgx *MsgXfer) Close() {

	for i := 0; i < len(msgx.destClients); i++ {
		msgx.destClients[i].close()
	}
}

func (msgctrl *MsgController) getClientStatus(cliUUID string) bool {
	aplist := msgctrl.getClientInfo(cliUUID)

	if len(aplist) > 0 {
		return true
	} else {
		return false
	}
}

func (msgctrl *MsgController) clientConnect(msgXferCli *MsgXferClient) error {

	logger.Debug(ctx, "Using controller", blog.Fields{"id": msgctrl.msgCtrlUUID})

	aplist := msgctrl.getClientInfo(msgXferCli.destUUID)

	for i := 0; i < len(aplist); i++ {

		if aplist[i].Type == cmn.NAT_IP {
			logger.Debug(ctx, "Skipping NAT IP as of now", nil)
			continue
		}
		err, conn := cmn.TcpConnect(aplist[i].AccessPoint, CTRL_CONNECT_RETRY_COUNT)
		if err == nil {
			// Connected.
			netconn := cmn.MsgNetConnection{Conn: conn, ConnLock: &sync.Mutex{}}

			err = cmn.VerifyAuthInternal(msgctrl.msgcli.accessKey,
				msgctrl.msgcli.accessSecret, msgctrl.source, msgctrl.msgCtrlUUID, netconn)

			if err == nil {
				msgXferCli.netConn = netconn
				return err
			}

			logger.Debug(ctx, "Authorization failed", blog.Fields{"key": msgctrl.msgcli.accessKey,
				"secret": msgctrl.msgcli.accessSecret})
			conn.Close()
		}
	}
	return errors.New("Connection failed")
}

//TODO:: Prioritize the IP address used for connecting to the client.
//       validate destination Client UUID if the connection is direct.

func (msgXferCli *MsgXferClient) init(msgcli *MsgCli) error {

	// Check with all the local controllers.
	msgcli.msgCtrlListLock.RLock()

	logger.Debug(ctx, "Using local controllers", nil)
	for _, ctrl := range msgcli.localMsgCtrl {

		err := ctrl.clientConnect(msgXferCli)

		// If error is auth, return here itself.
		if err == nil {
			return err
		}
	}
	msgcli.msgCtrlListLock.RUnlock()

	// Check with Global Controller
	err := msgcli.globalMsgCtrl.clientConnect(msgXferCli)
	return err

	/*
				// TestCode :: This code is to verify message flows via another controller.
			var aplisttweak []cmn.ClientAccessPoint
			aplisttweak = append(aplisttweak, cmn.ClientAccessPoint{AccessPoint: msgcli.globalMsgCtrl.netConn.Conn.RemoteAddr().String(),
				Type: cmn.GLOBAL_MSG_CTRL_IP})
			aplist = aplisttweak
		// TestCode :: end
	*/

}

func (msgcli *MsgXferClient) close() {
	if msgcli.netConn.Conn != nil {
		msgcli.netConn.Conn.Close()
	}
}

// MsgController related functions.

func (msgctrl *MsgController) init() error {

	msgctrl.clientInfoChan = make(map[string]chan []cmn.ClientAccessPoint)

	err := msgctrl.connect()

	if err != nil {
		return err
	}

	go msgctrl.readLoop()
	go msgctrl.sendHB()

	msgctrl.sendLoginData()

	return nil
}

func (msgctrl *MsgController) AddController(pkt cmn.MsgPkt) {
	var msgCtrlInfo cmn.MsgCtrlLoginReq
	dec := gob.NewDecoder(bytes.NewReader(*pkt.Payload))
	// Decode (receive) the value.
	err := dec.Decode(&msgCtrlInfo)

	if err == nil {
		logger.Info(ctx, "MSG_CTRL_LOGIN ", blog.Fields{"Id": msgCtrlInfo.MsgCtrlUUID})

		msgctrl.msgcli.addLocalController(msgCtrlInfo)
		msgctrl.sendLoginData()
	} else {
		logger.Crit(ctx, "Decode Error ", blog.Fields{"err": err.Error()})
	}
}

func (msgctrl *MsgController) deleteController(pkt cmn.MsgPkt) {
	var msgCtrlInfo cmn.MsgCtrlLoginReq
	dec := gob.NewDecoder(bytes.NewReader(*pkt.Payload))
	// Decode (receive) the value.
	err := dec.Decode(&msgCtrlInfo)
	if err == nil {

		logger.Info(ctx, "MSG_CTRL_LOGOUT ", blog.Fields{"Id": msgCtrlInfo.MsgCtrlUUID})

		msgctrl.msgcli.msgCtrlListLock.Lock()
		delete(msgctrl.msgcli.localMsgCtrl, msgCtrlInfo.MsgCtrlUUID)
		msgctrl.msgcli.msgCtrlListLock.Unlock()

		msgctrl.sendLoginData()
	} else {
		logger.Crit(ctx, "Decode Error ", blog.Fields{"err": err.Error()})
	}
}

func (msgctrl *MsgController) close() {

	if msgctrl.netConn.Conn != nil {
		logger.Info(ctx, "Closing controller session", blog.Fields{"Id": msgctrl.msgCtrlUUID})

		msgctrl.netConn.Conn.Close()
	}
}

func (msgctrl *MsgController) sendLoginData() error {

	var aplist []cmn.ClientAccessPoint
	var ap cmn.ClientAccessPoint

	logger.Debug(ctx, "Sending IP Data", blog.Fields{"Id": msgctrl.msgcli.cliUUID})

	// Add all the network intefaces and port that we are listening
	/*
		iparray := cmn.GetIPListToListen()
		_, localport, err := net.SplitHostPort(msgctrl.conn.LocalAddr().String())

		for i := 0; i < len(iparray); i++ {
			ap.AccessPoint = iparray[i] + ":" + localport
			ap.Type = cmn.LOCAL_NETWORK_IP
			aplist = append(aplist, ap)
		}
	*/

	// Add additional controller that this client has logged in.
	msgctrl.msgcli.msgCtrlListLock.RLock()
	for _, contrl := range msgctrl.msgcli.localMsgCtrl {

		if contrl.valid == false {
			continue
		}
		ap.AccessPoint = contrl.ipAddr + ":" + contrl.port
		ap.Type = contrl.msgCtrlType
		aplist = append(aplist, ap)
	}
	msgctrl.msgcli.msgCtrlListLock.RUnlock()

	// Add Global Controller ID
	ap.AccessPoint = msgctrl.msgcli.globalMsgCtrl.ipAddr + ":" +
		msgctrl.msgcli.globalMsgCtrl.port
	ap.Type = msgctrl.msgcli.globalMsgCtrl.msgCtrlType
	aplist = append(aplist, ap)

	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.

	// Encode (send) the value.
	err := enc.Encode(aplist)
	if err != nil {
		logger.Crit(ctx, "Encode Error ", blog.Fields{"err": err.Error()})
		return err
	}

	err = cmn.CreateAndSendMessage(msgctrl.netConn, msgctrl.msgcli.cliUUID,
		msgctrl.msgCtrlUUID, cmn.CLIENT_INFO_UPDATE, cmn.CLIENT_INFO_UPDATE, network.Bytes())

	if err != nil {
		logger.Warn(ctx, "Send Login Info Failed  ", blog.Fields{"err": err.Error()})
	}
	return err
}

func (msgctrl *MsgController) sendHB() error {

	payload := make([]byte, 0)
	for {

		logger.Debug(ctx, "Sending HB Data", blog.Fields{"Id": msgctrl.msgcli.cliUUID})
		err := cmn.CreateAndSendMessage(msgctrl.netConn, msgctrl.msgcli.cliUUID,
			msgctrl.msgCtrlUUID, cmn.CLIENT_LOGIN, cmn.CLIENT_LOGIN, payload)

		if err != nil {

			logger.Warn(ctx, "Send HB Failed  ", blog.Fields{"err": err.Error()})

			// Connection is reset. Try to reconnect again..
			msgctrl.reconnect(5)
		}

		time.Sleep(CLIENTHBINTERVAL)
	}
}

func (msgctrl *MsgController) getClientInfo(dest string) []cmn.ClientAccessPoint {

	ch, ok := msgctrl.clientInfoChan[dest]

	if ok == false {
		ch = make(chan []cmn.ClientAccessPoint)
		msgctrl.clientInfoChan[dest] = ch
		defer delete(msgctrl.clientInfoChan, dest)
	}

	payload := []byte(dest)
	err := cmn.CreateAndSendMessage(msgctrl.netConn, msgctrl.msgcli.cliUUID,
		dest, cmn.CLIENT_INFO_REQ, cmn.CLIENT_INFO_REQ, payload)

	if err != nil {
		logger.Warn(ctx, "Request ClientInfo Failed ", blog.Fields{"err": err.Error()})

	} else {
		aplist := <-ch

		//logger.Debug(ctx, "Destination info ", blog.Fields{"iplist": aplist})
		return aplist
	}

	return nil
}

func (msgctrl *MsgController) reconnect(retryCount int) {

	msgctrl.connect()
	msgctrl.sendLoginData()
}

func (msgctrl *MsgController) connect() error {

	accessPoint := msgctrl.ipAddr + ":" + msgctrl.port

	if err, conn := cmn.TcpConnect(accessPoint, msgctrl.retryCount); err == nil {

		netconn := cmn.MsgNetConnection{Conn: conn, ConnLock: &sync.Mutex{}}
		msgctrl.netConn = netconn
	}
	return msgctrl.verifyAuth()
}

func (msgctrl *MsgController) verifyAuth() error {
	return cmn.VerifyAuthInternal(msgctrl.msgcli.accessKey,
		msgctrl.msgcli.accessSecret, msgctrl.source, msgctrl.msgCtrlUUID, msgctrl.netConn)
}

func (msgctrl *MsgController) readLoop() error {

	var pkt cmn.MsgPkt
	pktChan := make(chan cmn.MsgPkt, 2)

	logger.Debug(ctx, "Started listen thread", blog.Fields{"Id": msgctrl.msgCtrlUUID})

	for {

		cmn.ReadMessagePacket(msgctrl.netConn, pktChan)
		pkt = <-pktChan

		err := pkt.Err

		if pkt.Err == nil {
			err = pkt.AppErr
		}

		if err != nil {

			switch err {

			case io.EOF:
				// Controller connection is broken. Return the error back to the caller.
				// Caller can decide what next action to perform
				return err

			default:
				if !strings.Contains(pkt.Err.Error(), "use of closed network connection") {

					logger.Debug(ctx, "Incomplete message", blog.Fields{"Id": msgctrl.msgCtrlUUID, "err": pkt.Err.Error()})
				}
				return err
			}
		}

		switch pkt.MesgMetaHeader.MsgType {

		case cmn.CLIENT_MSG:
			//	msgctrl.msgcli.MsgRecv <- MsgPkt{PayLoad: *pkt.Payload, Source: pkt.MesgMetaHeader.Src}
			msgctrl.msgcli.MsgRecv <- MsgPkt{PayLoad: *pkt.Payload, Source: pkt.MesgMetaHeader.Src,
				ClientTs: pkt.MesgHeader.ClientTs, CntrlRcvTs: pkt.MesgHeader.CntrlRcvTs,
				CntrlDstTs: pkt.MesgHeader.CntrlDstTs, DestTs: time.Now().UTC().UnixNano()}

		case cmn.MSG_CTRL_MESSAGE:
			logger.Debug(ctx, "CTRL_MSG ", blog.Fields{"Id": pkt.MesgMetaHeader.Dst})

		case cmn.MSG_CTRL_LOGIN:
			msgctrl.AddController(pkt)

		case cmn.MSG_CTRL_LOGOUT:
			msgctrl.deleteController(pkt)

		case cmn.CLIENT_INFO_RESP:
			handleInfoResponse(msgctrl, pkt)

		}
	}
}

func handleInfoResponse(msgctrl *MsgController, pkt cmn.MsgPkt) {
	var aplist []cmn.ClientAccessPoint

	ch := msgctrl.clientInfoChan[pkt.MesgMetaHeader.Dst]

	if ch == nil {
		logger.Info(ctx, "Did not expect", blog.Fields{"dst": pkt.MesgMetaHeader.Dst})
		return
	}
	dec := gob.NewDecoder(bytes.NewReader(*pkt.Payload))

	// Decode (receive) the value.
	err := dec.Decode(&aplist)
	if err != nil {
		logger.Crit(ctx, "Decode Error ", blog.Fields{"err": err.Error()})
	}
	// Notify this
	ch <- aplist
}

package qfsync

import (
	"sync"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/chreader"
	"github.com/ionosnetworks/qfx_dp/infra"
	"github.com/ionosnetworks/qfx_dp/msgqueue"
)

// A node's representation at any other
// node
var csCtx string = "CLOUDSEED"

type cloudSeed struct {
	csid     infra.CsID
	mq       *msgqueue.Mq
	alive    bool
	htbtCtrl chan struct{}
	sync.Mutex
	lastHeartBeat time.Time
}

var csTable struct {
	m map[infra.CsID]*cloudSeed
	sync.Mutex
}

func init() {
	csTable.m = make(map[infra.CsID]*cloudSeed)
}

func createCloudSeed(csid infra.CsID) (*cloudSeed, error) {
	defer csTable.Unlock()
	csTable.Lock()
	if cs, ok := csTable.m[csid]; ok {
		return cs, nil
	}
	xfer, err := glMsgCli.InitXfer(csid.String())
	if err[csid.String()] != nil {
		return nil, err[csid.String()]
	}
	// Try to get a message handle to the Cloud Seed device
	mq := msgqueue.New(5, 5,
		chreader.NewMsgWriter(xfer), nil)
	//TODO: put an error check here
	cs := &cloudSeed{
		mq:       mq,
		htbtCtrl: make(chan struct{}),
	}
	copy(cs.csid[:], csid[:])
	cs.heartBeat()
	cs.ListenForMessages()
	csTable.m[csid] = cs
	return cs, nil
}

func (cs *cloudSeed) heartBeat() {
	go func() {
		hb := SyncHeartBeat{
			CsID: infra.GetLocalCsID(),
		}
		msg := MsgWrapper{
			MsgID: MsgIDSyncHeartBeat,
			Data:  hb,
		}
		for {
			select {
			case <-time.After(heartBeatInterval):
				b, e := msg.Encode()
				if e != nil {
					logger.ErrS(csCtx, e.Error())
					continue
				}
				_, e = cs.mq.Write(b)
				if e != nil {
					logger.ErrS(csCtx, e.Error())
				}
				if time.Since(cs.lastHeartBeat) >
					disableCsTimeout {
					cs.Lock()
					cs.alive = false
					cs.Unlock()
				}
			case <-cs.htbtCtrl:
				logger.Info(csCtx, "Exiting htbt thread for",
					blog.Fields{
						"csid": cs.csid.String()})
			}
		}
	}()
	logger.Info(csCtx, "heart beat thread started for ", blog.Fields{
		"csid": cs.csid.String()})
}

func (cs *cloudSeed) ListenForMessages() {
	logger.InfoS(csCtx, "Listening for messages")
	go PrepareListener(cs.mq)
	RegisterConsumer(cs.csid.String(), cs.mq)
}

func (cs *cloudSeed) SendMsg(b []byte) error {
	_, err := cs.mq.Write(b)
	return err
}

func (cs *cloudSeed) Alive() bool {
	defer cs.Unlock()
	cs.Lock()
	return cs.alive
}

func updateOnlineStatus(hb *SyncHeartBeat) {
	csTable.Lock()
	cs, ok := csTable.m[hb.CsID]
	csTable.Unlock()
	if ok {
		cs.lastHeartBeat = time.Now()
		cs.Lock()
		cs.alive = true
		cs.Unlock()
	} else {
		logger.WarnS(csCtx, "Heartbeat for invalid cpe")
	}
}

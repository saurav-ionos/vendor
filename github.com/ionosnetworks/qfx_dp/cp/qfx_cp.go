package cp

import (
	"fmt"
	"os"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_cmn/msgsvc/msgcli"
	"github.com/ionosnetworks/qfx_cmn/qfxMsgs/dp2ctrl"
	"github.com/ionosnetworks/qfx_dp/infra"
	"github.com/ionosnetworks/qfx_dp/qfsync"
)

var localCsID infra.CsID
var logger blog.Logger
var msgCli *msgcli.MsgCli

func Start() {

	var err error
	var accesskey string = "0123456789"
	var secret string = "abcdefghijklmnopqrstuvwxyz"
	localCsID = infra.GetLocalCsID()
	// Create a connection to the global message server
	uuid := fmt.Sprintf("%s-%s", localCsID.String(), "cp")

	msgServer := os.Getenv("MSG_SERVER")
	if msgServer == "" {
		panic("MSG_SERVER not specified")
	}
	for {
		fmt.Printf("connecting to msg server @%s:%s with uuid %s\n", msgServer, "8080", uuid)
		msgCli, err = msgcli.New(msgServer, uuid, "8080", accesskey,
			secret, "msgcli-cp", nil)
		if err == nil {
			break
		} else {
			fmt.Println(err)
		}
		time.Sleep(time.Second)
	}

	go processControlMessages(msgCli.MsgRecv)
}

func processControlMessages(ch chan msgcli.MsgPkt) {

	fmt.Println("waiting for messages")
	for x := range ch {
		fmt.Println("packet received from", x.Source)
		// Decode the flat buffer

		msg := dp2ctrl.GetRootAsQfxCtrlMsg(x.PayLoad, 0)
		unionTable := new(flatbuffers.Table)
		if msg.Msg(unionTable) {
			unionType := msg.MsgType()
			if unionType == dp2ctrl.QfxMsgQfxSyncRel {

				// I need to create a new sync relation
				s := qfsync.CreateSyncRel(100)
				// I have a sync relation message
				syncRel := new(dp2ctrl.QfxSyncRel)
				syncRel.Init(unionTable.Bytes, unionTable.Pos)
				s.SetSrc(string(syncRel.SrcCSId()), string(syncRel.SrcCSDir()))
				// Create the sync entry here
				dstLen := syncRel.DstCSsLength()
				dst := new(dp2ctrl.DstCSSyncInfo)

				for i := 0; i < dstLen; i++ {
					if syncRel.DstCSs(dst, i) {
						s.SetDest(string(dst.DstCSId()), string(dst.DstCSDir()))

					}
				}
				go func() {
					s.Init()
					s.Sync()
				}()

			}
		}

	}

}

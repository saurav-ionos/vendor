package qfxObjects

import (
	msg "github.com/ionosnetworks/qfx_cmn/qfxMsgs/dp2ctrl"
)

type CpeInfo struct {
	CpeId         string `json:"cpeid"`
	PhysicalCpeId string `json:"physicalcpeid"`
	HwVersion     string `json:"hwversion"`
	ModelNumber   string `json:"modelnumber"`
	SerialNumber  string `json:"serialnumber"`

	//For Add/modify CPE
	Tenant string `json:"tenant"`
	User   string `json:"user"`
	Region string `json:"region"`

	UploadBW    int64 `json:"uploadbW"`
	DownloadBW  int64 `json:"downloadbW"`
	AllocatedBW int64 `json:"allocatedbW"`

	FwVersion string `json:"version"`
	Status    int    `json:"status"`
}

// For all entities like, CPE, IONS, SLC, Controller
const (
	ENTITY_CPE = msg.EntityTypeCPE
	ENTITY_ION = msg.EntityTypeION
	ENTITY_SLC = msg.EntityTypeSLC
)

// For All message types
const (
	SITE_UP     = msg.MsgTypeSITE_UP
	SITE_DOWN   = msg.MsgTypeSITE_DOWN
	QFX_MSG_ACK = msg.MsgTypeQFX_MSG_ACK
)

type MsgWrapper struct {
	MsgId   int32
	NeedAck int32
	MsgType int32
	Data    interface{}
}

type Siteup struct {
	Id     string `json:"id"`
	Entity int    `json:"entity"`
}

type Sitedown struct {
	Id     string `json:"id"`
	Entity int    `json:"entity"`
}

type QfxMsgAck struct {
}

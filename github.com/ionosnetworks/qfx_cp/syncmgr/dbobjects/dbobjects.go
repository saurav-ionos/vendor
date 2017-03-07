//go:generate stringer -type=ApiReturnStatus
package dbobjects

import (
	//"gopkg.in/mgo.v2/bson"
)

type ApiReturnStatus int

const (
	SUCCESS ApiReturnStatus = iota
	FAILURE ApiReturnStatus = iota
)

type DbResp struct {
	Status    string `json:"status"`
	ErrorCode string `json:"errorcode"`
	ErrorDesc string `json:"errordesc"`
}

type DbSyncReln struct {
	Name       string        `json:"name",        bson:"name"`
	SyncId     string        `json:"syncid",      bson:"syncid"`
	TenantName string        `json:"tenantname",  bson:"tenantname"`
	UserName   string        `json:"clientname",  bson:"clientname"`
	SrcCpeId   string        `json:"srccpeid",    bson:"srccpeid"`
	DstCpeIds  []string      `json:"dstcpeidlist",bson:"dstcpeidlist"`
}

type DbCreateSyncRelnInput struct {
	DbSyncReln
}

type DbCreateSyncRelnOutput struct {
	DbResp
}

type DbEditSyncRelnInput struct {
    DbSyncReln
}

type DbEditSyncRelnOutput struct {
    DbResp
}

type DbGetSyncRelnInput struct {
	SyncId string `json:"syncid", bson:"syncid"`
	TenantName string        `json:"tenantname",  bson:"tenantname"`
}

type DbGetSyncRelnOutput struct {
	DbSyncReln
	DbResp
}

package dao

import (
	"time"

	blog "github.com/ionosnetworks/qfx_cmn/blog"
	ecli "github.com/ionosnetworks/qfx_cmn/etcdclipool"
	o "github.com/ionosnetworks/qfx_cp/apisvr/objects"
)

type (
	DaoApiIn  o.ApiIn
	DaoApiOut o.ApiOut
	DaoApiKey o.ApiKey
	DAOErr    int
)

const (
	//EtcdAddress      = "http://etcd:2379"
	ETCDCLIENTTIMEOUT        = 30 * time.Second
	DAO_OK            DAOErr = 1
	DAO_ERR           DAOErr = 0
	STRLEN_APISVC            = len("/apisvc/")
)

var (
	// keyValMap   map[string]string
	EtcdAddress string
	ctx         = ""
	logger      blog.Logger
)

func SetLogger(context string, log blog.Logger) {

	ctx = context
	logger = log
}

func (api DaoApiIn) DAOSave() DAOErr {

	key := "/apisvc/" + api.AccessPoint
	ecli.SetKeyVal(key, api.Uri, 0)
	return DAO_OK
}

func (api DaoApiKey) DAOLoad() (DaoApiOut, DAOErr) {

	key := "/apisvc/" + api.AccessPoint
	uri := ecli.GetKey(key)
	keyout := DaoApiOut{AccessPoint: api.AccessPoint[STRLEN_APISVC:], Uri: uri, ErrCode: "OK"}

	return keyout, DAO_OK
}

func DAOGetAllRegistrations() ([]DaoApiOut, DAOErr) {

	apilist := make([]DaoApiOut, 0)

	//TODO
	/*
		if EtcdAddress == "" {

			for key, val := range keyValMap {
				apilist = append(apilist, DaoApiOut{AccessPoint: key[STRLEN_APISVC:], Uri: val})
			}

		} else {
			// TODO for ETCD.

		}
	*/

	return apilist, DAO_OK
}

func (api DaoApiKey) DAODelete() (o.ObjectDeleteOut, DAOErr) {

	key := "/apisvc/" + api.AccessPoint

	ecli.EtcdV3Del(key)

	return o.ObjectDeleteOut{"OK"}, DAO_OK
}

func SetEtcdAddress(etcdAP string) {
	ecli.Init(etcdAP, ctx, logger)
}

package main

import (
	"time"

	ecli "github.com/ionosnetworks/qfx_cmn/etcdclipool"
)

const (
	ETCDCLIENTTIMEOUT      = 30 * time.Second
	ETCD_CLIENT_POOL_COUNT = 10
	CLIENT_CONTRL_DIR      = "/CLIENTS/Controller/"
	CONTROLLER_DIR         = "/CONTROLLER/"
	CLIENT_DIR             = "/CLIENTS/"
)

/*
 All the controller information is stored under /CONTROLLER.
 Client information is split under /CLIENTS and /CLIENTS/CONTROLLER

 /CLIENTS - Will have client Id and its IP address

*/
func InitEtcd(etcdAP string, callbk func(interface{}, bool, string, string), cbkctx interface{}) {

	ecli.Init(etcdAP, ctx, logger)
	ecli.RegisterCallBk(etcdAP, CLIENT_CONTRL_DIR, callbk, cbkctx)
}

func GetControllerForClient(clientid string) string {

	key := CLIENT_CONTRL_DIR + clientid
	return ecli.GetKey(key)
}

func SetControllerForClient(clientid, controllerid string, leasetime int) {

	key := CLIENT_CONTRL_DIR + clientid
	ecli.SetKeyVal(key, controllerid, leasetime)
}

func DelClientEntry(clientid string) {

	ecli.DeleteKey(CLIENT_DIR + clientid)
	ecli.DeleteKey(CLIENT_CONTRL_DIR + clientid)
	ecli.DeleteKey(CLIENT_DIR + clientid + "/ip")
}

func SetClientIp(clientid, ipstr string) {

	key := CLIENT_DIR + clientid + "/ip"
	ecli.SetKeyVal(key, ipstr, 0)
}

func GetClientIp(clientid string) string {

	key := CLIENT_DIR + clientid + "/ip"
	return ecli.GetKey(key)
}

func SetControllerIp(controllerid string, iplist []string, port string, leasetime int) {

	accessPoints := ""
	key := CONTROLLER_DIR + controllerid + "/ip"

	for i := 0; i < len(iplist); i++ {
		accessPoints += iplist[i] + ":" + port + " "
	}
	ecli.SetKeyVal(key, accessPoints, leasetime)
}

func GetControllerIp(controllerid string) string {

	key := CONTROLLER_DIR + controllerid + "/ip"
	return ecli.GetKey(key)
}

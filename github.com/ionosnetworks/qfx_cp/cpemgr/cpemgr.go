package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	ihttp "github.com/ionosnetworks/qfx_cmn/httplib/svr"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	hnd "github.com/ionosnetworks/qfx_cp/cpemgr/handler"
)

const (
	SVC_ACCESS_FILE = "/keys/keyfile"
	SVR_CERT_FILE   = "/keys/lftsvr.crt"
	SVR_KEY_FILE    = "/keys/lftsvr.key"
)

var (
	ctx    = "cpesvc"
	logger blog.Logger

	globalApisvc = ""

	// Keys used for sending logs to log server.
	key kr.AccessKey

	svcName     = "cpesvc"
	svcPort     = "9090"
	accessPoint = ""

	routes = ihttp.Routes{
		// GET methods
		ihttp.Route{"CPE", "GET", "/cpe/allcpe", hnd.GetAllCpe},
		ihttp.Route{"CPE", "GET", "/cpe/usercpe", hnd.TODO},
		ihttp.Route{"CPE", "GET", "/cpe/tenantcpe", hnd.TODO},
		ihttp.Route{"CPE", "GET", "/cpe/unassigned", hnd.TODO},

		// Add
		ihttp.Route{"CPE", "POST", "/cpe/add", hnd.AddCpe},
		// Delete
		ihttp.Route{"CPE", "DELETE", "/cpe/delete", hnd.TODO},

		// Modify
		ihttp.Route{"CPE", "POST", "/cpe/assign", hnd.TODO},
		ihttp.Route{"CPE", "POST", "/cpe/unassign", hnd.TODO},

		//Login
		ihttp.Route{"CPE", "POST", "/cpe/login", hnd.Login},

		//Regions
		ihttp.Route{"CPE", "GET", "/cpe/getProviders", hnd.GetAllProvider},
		ihttp.Route{"CPE", "GET", "/cpe/getSites", hnd.GetAllSite},
		ihttp.Route{"CPE", "GET", "/cpe/getZones", hnd.GetAllZone},

		// Add
		ihttp.Route{"CPE", "POST", "/cpe/addRegion", hnd.AddRegion},
	}
)

func main() {

	readConfig()
	initLogger(key)

	hnd.SetLogger(ctx, logger)

	// Start reading CPE related messages from the bus.
	go processCPEMessages()

	svc := ihttp.New(ctx, svcPort, SVR_CERT_FILE, SVR_KEY_FILE, routes)
	svc.SetLogParams(ctx, logger)

	// This needs to be done periodically.
	accessPoint = svcName + ":" + svcPort
	if globalApisvc != "" {
		svc.RegisterAccessPoint(globalApisvc, key.Key, key.Secret, "/cpe", accessPoint)
	}

	fmt.Println("Starting cpemgr service ", accessPoint)
	svc.Start()

	time.Sleep(5 * time.Second)
	os.Exit(-1)
}

func readConfig() {

	if val := os.Getenv("GLOBAL_API_SVR"); val != "" {
		globalApisvc = val
	}

	if val := os.Getenv("CPESVC_PORT"); val != "" {
		svcPort = val
	}

	if val := os.Getenv("CPESVC_NAME"); val != "" {
		svcName = val
	}
	key = kr.New(SVC_ACCESS_FILE)

}

func initLogger(key kr.AccessKey) {

	logSvr := "127.0.0.1:2000"
	if val := os.Getenv("LOG_SERVER"); val != "" {
		logSvr = val
	}

	logger = blog.New(logSvr, key.Key, key.Secret)

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

func processCPEMessages() {

}

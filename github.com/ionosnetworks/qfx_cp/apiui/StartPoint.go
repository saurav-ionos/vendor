package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	ihttp "github.com/ionosnetworks/qfx_cmn/httplib/svr"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	hnd "github.com/ionosnetworks/qfx_cp/apiui/Handler"
	rt "github.com/ionosnetworks/qfx_cp/apiui/Routing"
)

const (
	SVC_ACCESS_FILE = "/keys/keyfile"
	SVR_CERT_FILE   = "/keys/lftsvr.crt"
	SVR_KEY_FILE    = "/keys/lftsvr.key"
)

var (
	ctx    = "apisvc1"
	logger blog.Logger

	globalApisvc = "127.0.0.1:8080"

	// Keys used for sending logs to log server.
	key kr.AccessKey

	svcName = "127.0.0.1"
	svcPort = "8088"
)

func main() {

	readConfig()
	initLogger(key.Key, key.Secret)

	svc := ihttp.New(ctx, svcPort, SVR_CERT_FILE, SVR_KEY_FILE, rt.ApiRoutes)
	svc.SetLogParams(ctx, logger)
	//svc.DefaultHandler = hnd.DefaultHandler
	// This needs to be done periodically.
	if globalApisvc != "" {
		svc.RegisterAccessPoint(globalApisvc, key.Key, key.Secret, "/api1", svcName+":"+svcPort)
	}

	hnd.SetLogger(ctx, logger)
	//db.InitDao(ctx, logger)

	fmt.Println("Starting key service")
	svc.Start()

	time.Sleep(5 * time.Second)
	os.Exit(-1)
}

func readConfig() {

	if val := os.Getenv("GLOBAL_API_SERVER"); val != "" {
		globalApisvc = val
	}

	if val := os.Getenv("APISVC_PORT"); val != "" {
		svcPort = val
	}

	if val := os.Getenv("APISVC_NAME"); val != "" {
		svcName = val
	}
	key = kr.New(SVC_ACCESS_FILE)
}

func initLogger(accessKey, secret string) {

	logSvr := "127.0.0.1:2000"
	if val := os.Getenv("LOG_SERVER"); val != "" {
		logSvr = val
	}

	logger = blog.New(logSvr, accessKey, secret)

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

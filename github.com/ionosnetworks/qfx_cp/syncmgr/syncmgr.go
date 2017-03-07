package main

import (
	_ "fmt"
	"github.com/ionosnetworks/qfx_cmn/blog"
	ihttp "github.com/ionosnetworks/qfx_cmn/httplib/svr"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	dbhndlr "github.com/ionosnetworks/qfx_cp/syncmgr/dbhandler"
	hndlr "github.com/ionosnetworks/qfx_cp/syncmgr/handler"
	"os"
	"time"
)

const (
	SVC_ACCESS_FILE = "/keys/keyfile"
	SVR_CERT_FILE   = "/keys/lftsvr.crt"
	SVR_KEY_FILE    = "/keys/lftsvr.key"
)

var (
	ctx          = "syncmgr"
	logger       blog.Logger
	globalApisvc = "127.0.0.1:8080"

	key         kr.AccessKey
	svcName     = "127.0.0.1"
	svcPort     = "1282"
	accessPoint = ""
)

func readConfig() {

	if val := os.Getenv("GLOBAL_API_SERVER"); val != "" {
		globalApisvc = val
	}

	if val := os.Getenv("SYNCSVC_PORT"); val != "" {
		svcPort = val
	}

	if val := os.Getenv("SYNCSVC_NAME"); val != "" {
		svcName = val
	}
	key = kr.New(SVC_ACCESS_FILE)
}

func initLogger(accessKey, secret string) {

	logSvr := "127.0.0.1:2000"
	if val := os.Getenv("SYNCSVC_LOG_SERVER"); val != "" {
		logSvr = val
	}

	logger = blog.LazyLog(logSvr, accessKey, secret)

	logLevel := "Debug"
	if level := os.Getenv("KEYSVC_LOG_LEVEL"); level != "" {
		logLevel = level
	}
	setLogLevel(logLevel)
}

func setLogLevel(logLevel string) {
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

func main() {
	readConfig()
	initLogger(key.Key, key.Secret)
	svc := ihttp.New(ctx, svcPort, SVR_CERT_FILE, SVR_KEY_FILE, routes)
	svc.SetLogParams(ctx, logger)
	accessPoint = svcName + ":" + svcPort
	if globalApisvc != "" {
		svc.RegisterAccessPoint(globalApisvc, key.Key, key.Secret, "/sync", accessPoint)
	}
	logger.Debug(ctx, "Starting SyncMgr Svc", blog.Fields{"accessPoint": accessPoint})
	hndlr.SetLogger(ctx, logger)
	dbhndlr.SetLogger(ctx, logger)
	dbhndlr.Init()
	svc.Start()
	time.Sleep(5 * time.Second)
	os.Exit(-1)
}

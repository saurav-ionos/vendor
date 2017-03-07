package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	ihttp "github.com/ionosnetworks/qfx_cmn/httplib/svr"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	icore "github.com/ionosnetworks/qfx_cp/apisvr/core"
	"github.com/ionosnetworks/qfx_cp/apisvr/dao"
	hnd "github.com/ionosnetworks/qfx_cp/apisvr/handler"
)

const (
	SVC_ACCESS_FILE = "/keys/keyfile"
	SVR_CERT_FILE   = "/keys/lftsvr.crt"
	SVR_KEY_FILE    = "/keys/lftsvr.key"
)

var (
	ctx    = "apisvc"
	logger blog.Logger

	etcdIP = ""

	// Keys used for sending logs to log server.
	key kr.AccessKey

	port = "8080"

	routes = ihttp.Routes{

		ihttp.Route{"Api", "POST", "/api", hnd.ApiRegisterCreate},
		ihttp.Route{"Api", "DELETE", "/api", hnd.ApiRegisterDelete},
		ihttp.Route{"Api", "GET", "/api", hnd.ApiRegisterGet},
	}
)

func initialize() {

	key = kr.New(SVC_ACCESS_FILE)
	initLogger(key.Key, key.Secret)
	hnd.SetLogger(ctx, logger)

	if val := os.Getenv("APISVR_PORT"); val != "" {
		port = val
	}

	if val := os.Getenv("ETCD_CLUSTER_IP"); val != "" {
		etcdIP = val
	} else {
		logger.Info(ctx, "Running in single mode", nil)
	}
	icore.Initialyze(key.Key, key.Secret, ctx, logger)
	dao.SetEtcdAddress(etcdIP)

	sigUserChan := make(chan os.Signal, 1)
	signal.Notify(sigUserChan, syscall.SIGUSR1)
	signal.Notify(sigUserChan, syscall.SIGUSR2)
	go handleSigUser1(sigUserChan)

	icore.PopulateHandlers()

}

func main() {

	initialize()
	svc := ihttp.New(ctx, port, SVR_CERT_FILE, SVR_KEY_FILE, routes)

	svc.SetLogParams(ctx, logger)

	svc.DefaultHandler = hnd.DefaultHandler

	fmt.Println("Starting API service")
	svc.Start()

	time.Sleep(5 * time.Second)
	os.Exit(-1)
}

func handleSigUser1(sch chan os.Signal) {

	for {
		signal := <-sch

		switch signal {
		case syscall.SIGUSR1:
			logger.Info(ctx, "Received USR1 ", nil)
		case syscall.SIGUSR2:
			logger.Info(ctx, "Received USR2 ", nil)
		}
	}
	return
}

func initLogger(key, secret string) {

	logSvr := "127.0.0.1:2000"
	if val := os.Getenv("LOG_SERVER"); val != "" {
		logSvr = val
	}

	logger = blog.New(logSvr, key, secret)

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

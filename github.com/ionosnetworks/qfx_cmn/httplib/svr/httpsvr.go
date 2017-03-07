package svr

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ionosnetworks/qfx_cmn/blog"
	hcli "github.com/ionosnetworks/qfx_cmn/httplib/client"
	apio "github.com/ionosnetworks/qfx_cp/apisvr/objects"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

type HttpSvc struct {
	name     string
	port     string
	certFile string
	keyFile  string

	readtimeout  int
	writetimeout int

	logger blog.Logger
	ctx    string
	routes Routes

	DefaultHandler http.HandlerFunc
}

func New(name, port, certfile, keyfile string, routes Routes) *HttpSvc {

	svc := HttpSvc{name: name, port: port, routes: routes,
		readtimeout: 10, writetimeout: 10,
		certFile: certfile, keyFile: keyfile, DefaultHandler: defHandler}

	return &svc
}

func (svc *HttpSvc) SetLogParams(ctx string, logger blog.Logger) {
	svc.ctx = ctx
	svc.logger = logger
}

func (svc *HttpSvc) RegisterAccessPoint(globalApisvc, key, secret, accessPoint, url string) {

	operation := "POST"
	command := "api"

	query, _ := json.Marshal(apio.ApiIn{AccessPoint: accessPoint, Uri: url})

	cli := hcli.New(globalApisvc)

	result, err := cli.RunCommand(operation, command, bytes.NewReader(query))

	if err != nil {
		svc.logger.Info(svc.ctx, "Registering with Api server", blog.Fields{"AP": accessPoint,
			"URI": url, "result": result, "err": err.Error()})
	} else {
		svc.logger.Info(svc.ctx, "Registering with Api server", blog.Fields{"AP": accessPoint,
			"URI": url, "result": string(result)})
	}

}

func (svc *HttpSvc) UnregisterAccessPoint(globalApisvc, key, secret, accessPoint string) {

	operation := "DELETE"
	command := "api"

	query, _ := json.Marshal(apio.ApiKey{AccessPoint: accessPoint})

	cli := hcli.New(globalApisvc)

	result, err := cli.RunCommand(operation, command, bytes.NewReader(query))

	svc.logger.Info(svc.ctx, "Unregistering with Api server", blog.Fields{"AP": accessPoint,
		"URI": accessPoint, "result": result, "err": err.Error()})
}

func (svc *HttpSvc) Logger(inner http.Handler, name string) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Verify request.
		inner.ServeHTTP(w, r)

		svc.logger.Info(svc.ctx, "Logger::", blog.Fields{"Method": r.Method, "URI": r.RequestURI,
			"Name": name, "Time": time.Since(start).String()})

	})
}

func (svc *HttpSvc) NewRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range svc.routes {
		var handler http.Handler

		handler = route.HandlerFunc
		handler = svc.Logger(handler, route.Name)

		router.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(handler)

	}
	router.NotFoundHandler = http.HandlerFunc(svc.DefaultHandler)
	return router
}

func (svc *HttpSvc) Start() {

	router := svc.NewRouter()

	server := &http.Server{
		Addr:         ":" + svc.port,
		Handler:      router,
		ReadTimeout:  time.Duration(svc.readtimeout) * time.Second,
		WriteTimeout: time.Duration(svc.writetimeout) * time.Second,
	}
	server.TLSConfig = svc.ConfigTLS()

	svc.logger.Info(svc.ctx, "Server started ", blog.Fields{"Svc": svc.name, "Addr": server.Addr})
	err := server.ListenAndServeTLS(svc.certFile, svc.keyFile)

	svc.logger.Crit(svc.ctx, "Server Stopped ", blog.Fields{"Svc": svc.name, "err": err.Error()})

}

func (svc *HttpSvc) ConfigTLS() *tls.Config {

	cer, err := tls.LoadX509KeyPair(svc.certFile, svc.keyFile)
	if err != nil {
		svc.logger.Crit(svc.ctx, "Failed to load certificates", blog.Fields{"err": err.Error()})
		return nil
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}

	config.Rand = rand.Reader
	config.MinVersion = tls.VersionTLS10
	config.SessionTicketsDisabled = false
	config.InsecureSkipVerify = false
	config.ClientAuth = tls.NoClientCert
	config.PreferServerCipherSuites = true
	config.ClientSessionCache = tls.NewLRUClientSessionCache(1000)

	return config
}

func defHandler(w http.ResponseWriter, r *http.Request) {

	_, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	command := r.URL.String()
	method := r.Method
	token := r.Header.Get("ACCESS_KEY")
	secret := r.Header.Get("ACCESS_SECRET")

	fmt.Println("DefaultHandler:: command", command,
		"Method", method, "Token", token, "Secret", secret)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// No handler at this path. Return error 404.
	w.WriteHeader(http.StatusNotFound)
	io.WriteString(w, "NO HANDLERS FOR THE REQUESTED PATH "+string(command))

}

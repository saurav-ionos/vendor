package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	icore "github.com/ionosnetworks/qfx_cp/apisvr/core"
)

const (
	CaPath     = "/home/chandra/work/CloudVault/test/chandra/ca.crt"
	ClientCert = "key.crt"
	ClientKey  = "/home/chandra/work/CloudVault/test/chandra/CloudVault-chandra.key"
)

var (
	HttpClient *http.Client
	ClientInit = false
)

func DefaultHandler(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	method := r.Method
	command := r.URL.String()
	token := r.Header.Get("ACCESS_KEY")
	secret := r.Header.Get("ACCESS_SECRET")

	logger.Debug(ctx, "DefaultHandler::", blog.Fields{"command": command, "Method": method,
		"Token": token, "Secret": secret})

	if ret, featureList := icore.ValidateRequest(token, secret, command); ret == false {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "Access Denied")
		return
	} else if featureList != nil && len(featureList) > 0 {

		w.Header().Set("QFX-Features", strings.Join(featureList, " "))
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	ap := icore.GetHandler(string(command))

	if ap == "" {
		// No handler at this path. Return error 404.
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "NO HANDLERS FOR THE REQUESTED PATH "+string(command))
	} else {

		url := "https://" + ap + command
		logger.Debug(ctx, "DefaultHandler::", blog.Fields{"URL": url})

		req, err := http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			panic(err)
		}

		copyHeader(req.Header, r.Header)

		client := GetHttpClient()
		response, err := client.Do(req)

		if err != nil {
			logger.Debug(ctx, "DefaultHandler:: Clientcall ", blog.Fields{"Err": err.Error()})
			w.WriteHeader(500)
		} else {
			defer response.Body.Close()
			contents, err2 := ioutil.ReadAll(response.Body)
			if err2 != nil {
				logger.Debug(ctx, "DefaultHandler::Response", blog.Fields{"Err": err2.Error()})
				w.WriteHeader(500)
			} else {
				copyHeader(w.Header(), response.Header)

				w.WriteHeader(response.StatusCode)
				w.Write(contents)
			}
		}
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func ConfigTLS() *tls.Config {

	TLSConfig := &tls.Config{}

	TLSConfig.Rand = rand.Reader
	TLSConfig.MinVersion = tls.VersionTLS10
	TLSConfig.SessionTicketsDisabled = false
	TLSConfig.InsecureSkipVerify = true

	return TLSConfig
}

func GetHttpClient() *http.Client {

	if ClientInit == false {
		tr := &http.Transport{
			DisableCompression: true,
		}
		tr.TLSClientConfig = ConfigTLS()
		HttpClient = &http.Client{Transport: tr}
	}
	return HttpClient
}

func Logger(inner http.Handler, name string) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		method := r.Method
		command := r.URL.String()
		token := r.Header.Get("ACCESS_KEY")
		secret := r.Header.Get("ACCESS_SECRET")

		logger.Debug(ctx, "DefaultHandler::", blog.Fields{"command": command, "Method": method,
			"Token": token, "Secret": secret})

		if ret, _ := icore.ValidateRequest(token, secret, command); ret == false {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, "Access Denied")
			return
		}
		inner.ServeHTTP(w, r)

		logger.Debug(ctx, "Logger::", blog.Fields{"Method": r.Method, "URI": r.RequestURI,
			"Name": name, "Time": time.Since(start)})

	})
}

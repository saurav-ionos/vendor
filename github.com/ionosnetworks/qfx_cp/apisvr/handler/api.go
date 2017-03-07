package handler

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ionosnetworks/qfx_cmn/blog"
	icore "github.com/ionosnetworks/qfx_cp/apisvr/core"
	iobjects "github.com/ionosnetworks/qfx_cp/apisvr/objects"
)

var (
	ctx    = ""
	logger blog.Logger
)

func SetLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
}

func ApiRegisterCreate(w http.ResponseWriter, r *http.Request) {
	var api iobjects.ApiIn
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &api); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	apiout := (icore.CoreApiIn(api)).Process()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiout); err != nil {
		panic(err)
	}

}

func ApiRegisterDelete(w http.ResponseWriter, r *http.Request) {
	var api iobjects.ApiKey
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &api); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	apiout := (icore.CoreApiKey(api)).Delete()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiout); err != nil {
		panic(err)
	}

}

func ApiRegisterGet(w http.ResponseWriter, r *http.Request) {
	var api iobjects.ApiKey
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &api); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	apiout := (icore.CoreApiKey(api)).Load()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiout); err != nil {
		panic(err)
	}
}

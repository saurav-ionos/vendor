package handler

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ionosnetworks/qfx_cmn/blog"
	kcore "github.com/ionosnetworks/qfx_cp/keysvc/core"
	kobjects "github.com/ionosnetworks/qfx_cp/keysvc/objects"
)

var (
	ctx                = ""
	logger blog.Logger = nil
)

func SetLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
	kcore.SetLogger(context, log)
}

func KeyCreate(w http.ResponseWriter, r *http.Request) {
	var key kobjects.KeyCreateIn
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &key); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	token := r.Header.Get("ACCESS_KEY")
	keyout := (kcore.CoreKeyCreateIn(key)).Process(token)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(keyout); err != nil {
		panic(err)
	}
}

func KeyDelete(w http.ResponseWriter, r *http.Request) {
	var key kobjects.KeyIn
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &key); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	out := (kcore.CoreKeyIn(key)).Delete()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		panic(err)
	}
}

func KeyGet(w http.ResponseWriter, r *http.Request) {
	var key kobjects.KeyIn
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &key); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	keyout := (kcore.CoreKeyIn(key)).Load()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(keyout); err != nil {
		panic(err)
	}
}

func ClientKeyGet(w http.ResponseWriter, r *http.Request) {
	var key kobjects.ClientKeyIn
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &key); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	keyout := (kcore.CoreClientKeyIn(key)).Load()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(keyout); err != nil {
		panic(err)
	}
}

func DefaultHandler(w http.ResponseWriter, r *http.Request) {

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

	if logger != nil {
		logger.Debug(ctx, "DefaultHandler::", blog.Fields{"command": command,
			"Method": method, "Token": token, "Secret": secret})
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// No handler at this path. Return error 404.
	w.WriteHeader(http.StatusNotFound)
	io.WriteString(w, "NO HANDLERS FOR THE REQUESTED PATH "+string(command))

}

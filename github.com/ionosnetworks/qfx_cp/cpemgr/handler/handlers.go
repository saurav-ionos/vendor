package handler

import (
	"encoding/json"
	"fmt"
	"github.com/ionosnetworks/qfx_cmn/blog"
	"io"
	"io/ioutil"
	"net/http"

	cmnCpeo "github.com/ionosnetworks/qfx_cmn/qfxObjects"
	core "github.com/ionosnetworks/qfx_cp/cpemgr/core"
)

var (
	ctx    = ""
	logger blog.Logger
)

func SetLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
}

func Login(w http.ResponseWriter, r *http.Request) {

	logger.Info(ctx, "Processing Login", nil)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)

	fmt.Println("Processing Login")
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	fmt.Println("Request Param ::", string(body))
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode("Niti"); err != nil {
		panic(err)
	}
}

func GetAllCpe(w http.ResponseWriter, r *http.Request) {

	logger.Info(ctx, "Processing Get All Cpes", nil)
	sessionId := r.Header.Get("SessionId")
	logger.Info(ctx, "SessionId: ", blog.Fields{"sessionId": sessionId})
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)
	_, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	out := core.GetAllCpe()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		panic(err)
	}
}

func TODO(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	fmt.Println("Request Param ::", string(body))
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("To be implemented"))

}

func AddCpe(w http.ResponseWriter, r *http.Request) {
	logger.Info(ctx, "Processing Add Cpe", nil)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)
	var cpe cmnCpeo.CpeInfo
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logger.Err(ctx, "Error reading request body", blog.Fields{"Err": err.Error()})
	} else {
		err := json.Unmarshal(body, &cpe)
		if err != nil {
			logger.Err(ctx, "Error in Processing Add CPE", blog.Fields{"Err": err.Error()})
		} else {
			logger.Info(ctx, "Cpe to be Added: ", blog.Fields{"cpe": cpe.CpeId})
			out := core.AddCpe(cpe)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(out); err != nil {
				panic(err)
			}
		}
	}
}

func AddRegion(w http.ResponseWriter, r *http.Request) {
	logger.Info(ctx, "Processing Add Region", nil)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)
	var region cmnCpeo.Region
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logger.Err(ctx, "Error reading request body", blog.Fields{"Err": err.Error()})
	} else {
		err := json.Unmarshal(body, &region)
		if err != nil {
			logger.Err(ctx, "Error in Processing Add Region", blog.Fields{"Err": err.Error()})
		} else {
			logger.Info(ctx, "Region to be Added: ", blog.Fields{"region": region.Provider})
			out := core.AddRegion(region)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(out); err != nil {
				panic(err)
			}
		}
	}
}

func GetAllProvider(w http.ResponseWriter, r *http.Request) {
	logger.Info(ctx, "Processing Get All Provider", nil)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)
	_, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	out := core.GetAllProvider()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		panic(err)
	}
}

func GetAllSite(w http.ResponseWriter, r *http.Request) {
	logger.Info(ctx, "Processing Get All Site", nil)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)
	if r.URL.Query().Get("provider") == "" {
		logger.Err(ctx, "Provider passed is empty", nil)
	} else {
		out := core.GetAllSite(r.URL.Query().Get("provider"))
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(out); err != nil {
			panic(err)
		}
	}
}

func GetAllZone(w http.ResponseWriter, r *http.Request) {
	logger.Info(ctx, "Processing Get All Zone", nil)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)
	if r.URL.Query().Get("site") == "" {
		logger.Err(ctx, "Site passed is empty", nil)
	} else {
		out := core.GetAllZone(r.URL.Query().Get("site"))
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(out); err != nil {
			panic(err)
		}
	}
}

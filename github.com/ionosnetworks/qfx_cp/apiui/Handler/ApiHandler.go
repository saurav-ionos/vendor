package Handler

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_cmn/msgq/producer"
	Constants "github.com/ionosnetworks/qfx_cp/apiui/Constants"
	DataObjects "github.com/ionosnetworks/qfx_cp/apiui/DataObjects"
	gorequest "github.com/parnurzeal/gorequest"
)

var (
	ctx    = ""
	logger blog.Logger
)

func SetLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
}

func CheckLogin(w http.ResponseWriter, r *http.Request) {

	logger.Info(ctx, "Processing Login", nil)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)

	var usrInfo DataObjects.LoginInfoObj
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logger.Err(ctx, "Error reading request body", blog.Fields{"Err": err.Error()})
	} else {
		err := json.Unmarshal(body, &usrInfo)
		if err != nil {
			logger.Err(ctx, "Error in Processing Login", blog.Fields{"Err": err.Error()})
		} else {
			var loginInputObj DataObjects.LoginInputObj
			loginInputObj.Input = usrInfo
			url := Constants.IC2URLUTM + "login"
			request := gorequest.New()
			_, body, errs := request.Post(url).
				Send(loginInputObj).
				End()
			if errs != nil {
				logger.Err(ctx, "Error in Posting request", blog.Fields{"Err": errs})
			} else {
				fmt.Println("*****", body)
				w.Write([]byte(body))
			}
		}
	}
}

func GetCpe(w http.ResponseWriter, r *http.Request) {
	logger.Info(ctx, "Get Cpes", nil)
	vars := mux.Vars(r)
	category := vars["category"]
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(200)
	var cpeInfo DataObjects.CpeInfoObj
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logger.Err(ctx, "Error reading request body", blog.Fields{"Err": err.Error()})
	} else {
		err := json.Unmarshal(body, &cpeInfo)
		if err != nil {
			logger.Err(ctx, "Error in getting Cpes Info", blog.Fields{"Err": err.Error()})
		} else {
			url := ""
			if category == "all" {
				var CpeInputObj DataObjects.CpeInputObj
				CpeInputObj.Input = cpeInfo
				url = Constants.IC2URLCPE + "ui-get-all-cpes"
				request := gorequest.New()
				_, body, errs := request.Post(url).
					Send(CpeInputObj).
					End()
				if errs != nil {
					logger.Err(ctx, "Error in Posting request", blog.Fields{"Err": errs})
				} else {
					w.Write([]byte(body))
				}
			} else {
				var cpeDetailObj DataObjects.CpeDetailObj
				var cpeDetailObjInputObj DataObjects.CpeDetailObjInputObj
				cpeDetailObj.SessionId = cpeInfo.SessionId
				cpeDetailObj.CpeId = category
				cpeDetailObjInputObj.Input = cpeDetailObj
				url = Constants.IC2URLCPE + "ui-get-cpe"
				request := gorequest.New()
				_, body, errs := request.Post(url).
					Send(cpeDetailObjInputObj).
					End()
				if errs != nil {
					logger.Err(ctx, "Error in Posting request", blog.Fields{"Err": errs})
				} else {
					w.Write([]byte(body))
				}
			}

		}
	}
}

func UpdateCpeLogLevel(w http.ResponseWriter, r *http.Request) {
	logger.Info(ctx, "Updating Log Level", nil)
	response := make(map[string]DataObjects.Response)
	var cpeLogLevelInfo DataObjects.CpeLogLevelInfo
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logger.Err(ctx, "Error reading request body", blog.Fields{"Err": err.Error()})
	} else {
		err := json.Unmarshal(body, &cpeLogLevelInfo)
		if err != nil {
			logger.Err(ctx, "Error in Processing CPE Log Level Info", blog.Fields{"Err": err.Error()})
		} else {
			logger.Info(ctx, "CpeLogLevelInfo: ", blog.Fields{"cpeLogLevelInfo": cpeLogLevelInfo})
			response["output"] = DataObjects.Response{Result: "SUCCESS", ErrorRes: Constants.NoError}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				logger.Err(ctx, "Error encoding json", blog.Fields{"Err": err.Error()})
			}
			logger.Info(ctx, "Sending request to MsgServ: ", nil)
			broker := "192.168.1.173:9092"
			brokers := []string{broker}
			producer.Init("testNitiProducer", brokers)
			producer.SendSyncMessage("topic4", "12345", []byte("Log level of CPE X changed to"))
			logger.Info(ctx, "Sent request to MsgServ ", nil)
		}
	}
}

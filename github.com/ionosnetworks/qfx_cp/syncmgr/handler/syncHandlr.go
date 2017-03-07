package handler

import (
	"encoding/json"
	"github.com/ionosnetworks/qfx_cmn/blog"
	dbhndlr "github.com/ionosnetworks/qfx_cp/syncmgr/dbhandler"
	syncDbObjects "github.com/ionosnetworks/qfx_cp/syncmgr/dbobjects"
	syncObjects "github.com/ionosnetworks/qfx_cp/syncmgr/objects"
	"github.com/ionosnetworks/qfx_cp/syncmgr/validator"
	"io"
	"io/ioutil"
	"net/http"
)

var (
	ctx                = ""
	logger blog.Logger = nil
)

func SetLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
}

func CreateSyncReln(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logger.Err(ctx, "In CreateSyncReln Error in Reading request Body", blog.Fields{"err": err.Error()})
		return
	}
	if err := r.Body.Close(); err != nil {
		logger.Err(ctx, "In CreateSyncReln Error in Closing body", blog.Fields{"err": err.Error()})
		return
	}

	logger.Debug(ctx, "In CreateSyncReln", blog.Fields{"Body": string(body[:])})

	var output syncObjects.CreateSyncRelnOutput
	var syncReln syncObjects.CreateSyncRelnInput
	output.Status = syncObjects.FAILURE.String()

	if err := json.Unmarshal(body, &syncReln); err != nil {
		logger.Err(ctx, "Unmarshalling Error In CreateSyncReln.", blog.Fields{"err": err.Error()})
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		output.ErrorCode = "INVALIDINPUT"
		output.ErrorDesc = err.Error()
		if err := json.NewEncoder(w).Encode(output); err != nil {
			logger.Err(ctx, "Json Encoding Error In CreateSyncReln.", blog.Fields{"err": err.Error()})
		}
		return
	}

	err = validator.ValidateCreateSyncRelnUiInput(&syncReln)
	if err == nil {
		dhndlr := dbhndlr.NewSyncDbHandler()
		dbOutput, _ := dhndlr.CreateSyncReln(&syncReln)
		createSyncRelnUiOutput(&output, dbOutput)
	} else {
		output.Status = syncObjects.FAILURE.String()
		output.ErrorCode = "INVALIDINPUT"
		output.ErrorDesc = err.Error()
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if output.Status == "FAILURE" {
		w.WriteHeader(422)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(output); err != nil {
		logger.Err(ctx, "Encoding Error while returning in CreateSyncReln.", blog.Fields{"err": err.Error()})
	}
    logger.Debug(ctx, "Returning from CreateSyncReln", nil)
}

func createSyncRelnUiOutput(output *syncObjects.CreateSyncRelnOutput, dbOutput *syncDbObjects.DbCreateSyncRelnOutput) {
	output.Status = dbOutput.Status
	output.ErrorCode = dbOutput.ErrorCode
	output.ErrorDesc = dbOutput.ErrorDesc
    //copy more fields if present
}

func editSyncRelnUiOutput(output *syncObjects.EditSyncRelnOutput, dbOutput *syncDbObjects.DbEditSyncRelnOutput) {
	output.Status = dbOutput.Status
	output.ErrorCode = dbOutput.ErrorCode
	output.ErrorDesc = dbOutput.ErrorDesc
    //copy more fields if present
}

func getSyncRelnUiOutput(output *syncObjects.GetSyncRelnOutput, dbOutput *syncDbObjects.DbGetSyncRelnOutput) {
     out, err := json.Marshal(*dbOutput)
     if err != nil {
        logger.Err(ctx,"Error in Marshalling in getSyncRelnUiOutput", nil)
     }
     err = json.Unmarshal(out, &output)
     if err != nil {
        logger.Err(ctx,"Error in UnMarshalling in getSyncRelnUiOutput", nil)
     }
}

func DeleteSyncReln(w http.ResponseWriter, r *http.Request) {
}

func GetSyncReln(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logger.Err(ctx, "In GetSyncReln Error in Reading request Body", blog.Fields{"err": err.Error()})
		return
	}
	if err := r.Body.Close(); err != nil {
		logger.Err(ctx, "In GetSyncReln Error in Closing body", blog.Fields{"err": err.Error()})
		return
	}

	logger.Debug(ctx, "In GetSyncReln", blog.Fields{"Body": string(body[:])})

	var output syncObjects.GetSyncRelnOutput
	var syncReln syncObjects.GetSyncRelnInput
	output.Status = syncObjects.FAILURE.String()

	if err := json.Unmarshal(body, &syncReln); err != nil {
		logger.Err(ctx, "Unmarshalling Error In GetSyncReln.", blog.Fields{"err": err.Error()})
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		output.ErrorCode = "INVALIDINPUT"
		output.ErrorDesc = err.Error()
		if err := json.NewEncoder(w).Encode(output); err != nil {
			logger.Err(ctx, "Json Encoding Error In GetSyncReln.", blog.Fields{"err": err.Error()})
		}
		return
	}

	err = validator.ValidateGetSyncRelnUiInput(&syncReln)
	if err == nil {
		dhndlr := dbhndlr.NewSyncDbHandler()
		dbOutput, err := dhndlr.GetSyncRelnDetails(&syncReln)
        if err != nil {
           logger.Err(ctx, "Error while Getting SyncReln Details", blog.Fields{"err":err.Error()})
        }
		getSyncRelnUiOutput(&output, dbOutput)
	} else {
		output.Status = syncObjects.FAILURE.String()
		output.ErrorCode = "INVALIDINPUT"
		output.ErrorDesc = err.Error()
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if output.Status == "FAILURE" {
		w.WriteHeader(422)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(output); err != nil {
		logger.Err(ctx, "Encoding Error while returning in CreateSyncReln.", blog.Fields{"err": err.Error()})
	}
    logger.Debug(ctx, "Returning from CreateSyncReln", nil)
}

func EditSyncReln(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logger.Err(ctx, "In EditSyncReln Error in Reading request Body", blog.Fields{"err": err.Error()})
		return
	}
	if err := r.Body.Close(); err != nil {
		logger.Err(ctx, "In EditSyncReln Error in Closing body", blog.Fields{"err": err.Error()})
		return
	}

	logger.Debug(ctx, "In EditSyncReln", blog.Fields{"Body": string(body[:])})

	var output syncObjects.EditSyncRelnOutput
	var syncReln syncObjects.EditSyncRelnInput
	output.Status = syncObjects.FAILURE.String()

	if err := json.Unmarshal(body, &syncReln); err != nil {
		logger.Err(ctx, "Unmarshalling Error In EditSyncReln.", blog.Fields{"err": err.Error()})
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		output.ErrorCode = "INVALIDINPUT"
		output.ErrorDesc = err.Error()
		if err := json.NewEncoder(w).Encode(output); err != nil {
			logger.Err(ctx, "Json Response Encoding Error In EditSyncReln.", blog.Fields{"err": err.Error()})
		}
		return
	}

	err = validator.ValidateEditSyncRelnUiInput(&syncReln)
	if err == nil {
		dhndlr := dbhndlr.NewSyncDbHandler()
		dbOutput, _ := dhndlr.EditSyncReln(&syncReln)
		editSyncRelnUiOutput(&output, dbOutput)
	} else {
		output.Status = syncObjects.FAILURE.String()
		output.ErrorCode = "INVALIDINPUT"
		output.ErrorDesc = err.Error()
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if output.Status == "FAILURE" {
		w.WriteHeader(422)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(output); err != nil {
		logger.Err(ctx, "Encoding Error while returning in EditSyncReln.", blog.Fields{"err": err.Error()})
	}
    logger.Debug(ctx, "Returning from EditSyncReln", nil)
}

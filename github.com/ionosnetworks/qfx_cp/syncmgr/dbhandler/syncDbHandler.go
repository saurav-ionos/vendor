package dbhandler

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_cmn/db/mongodb"
	"github.com/ionosnetworks/qfx_cmn/utils"
	syncDbObjects "github.com/ionosnetworks/qfx_cp/syncmgr/dbobjects"
	syncObjects "github.com/ionosnetworks/qfx_cp/syncmgr/objects"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	ctx     string
	logger  blog.Logger
	session *mgo.Session
)

const (
	SYNC_RELATION_TABLE = "syncrelations"
    FAILURE = "FAILURE"
    SUCCESS = "SUCCESS"
    INTERNAL_ERROR = "INTERNAL_ERROR"
)

func SetLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
}

type (
	SyncDbHandler struct {
		session *mgo.Session
		isMongo bool
	}
)

func Init() {
	logger.Info(ctx, "Initializing DB", nil)
	dbHost := os.Getenv("DB_SVC")
	dbPort := os.Getenv("DB_PORT")
	var err error
	session, err = mongodb.GetClusterSession([]string{dbHost}, dbPort)
	if err != nil {
		panic(err)
	}
}

func NewSyncDbHandler() *SyncDbHandler {
	return &SyncDbHandler{session: session, isMongo: true}
}

func (hndlr *SyncDbHandler) CreateSyncReln(syncRelnInput *syncObjects.CreateSyncRelnInput) (*syncDbObjects.DbCreateSyncRelnOutput, error) {
	out, err := json.Marshal(*syncRelnInput)
	if err != nil {
		return nil, err
	}
	var dbInput syncDbObjects.DbCreateSyncRelnInput
	err = json.Unmarshal(out, &dbInput)
	dbInput.SyncId = utils.GetGuidStr()
	//dbInput.Id = bson.ObjectId(dbInput.SyncId)
	//dbInput.Id = bson.NewObjectId()
	dbInput.Name = dbInput.SyncId
	var output syncDbObjects.DbCreateSyncRelnOutput

	exists, err := hndlr.doesSyncRelnExists(dbInput.Name, dbInput.TenantName)
	if err == nil {
		if exists == false {
			err = hndlr.createNewSyncReln(&dbInput)
			if err == nil {
				output.Status = "SUCCESS"
				output.ErrorCode = "NONE"
				output.ErrorDesc = "NONE"
			} else {
				output.Status = "FAILURE"
				output.ErrorCode = "InternalError"
				output.ErrorDesc = "Error while Writing Sync Relation into DB"
			}
		} else {
			err = errors.New("Sync Reln Already Exists")
			output.Status = "FAILURE"
			output.ErrorCode = "ALREADYEXISTS"
			output.ErrorDesc = "This Sync Relation Already exists"
		}
	} else {
		output.Status = "FAILURE"
		output.ErrorCode = "InternalError"
		output.ErrorDesc = "Error while Fetching Data"
	}
	return &output, err
}

func (hndlr *SyncDbHandler) EditSyncReln(syncRelnInput *syncObjects.EditSyncRelnInput) (*syncDbObjects.DbEditSyncRelnOutput, error) {
	out, err := json.Marshal(*syncRelnInput)
	if err != nil {
		return nil, err
	}
	var dbInput syncDbObjects.DbEditSyncRelnInput
	err = json.Unmarshal(out, &dbInput)
	var output syncDbObjects.DbEditSyncRelnOutput

	exists, err := hndlr.doesSyncRelnExists(dbInput.Name, dbInput.TenantName)
	if err == nil {
		if exists == false {
			if err == nil {
				output.Status = "FAILURE"
				output.ErrorCode = "DOESNOTEXISTS"
				output.ErrorDesc = "No Sync Reln exists with given Name/Id"
			} else {
				output.Status = "FAILURE"
				output.ErrorCode = "InternalError"
				output.ErrorDesc = "Error while Writing Sync Relation into DB"
			}
		} else {
			err = hndlr.updateSyncReln(&dbInput.DbSyncReln)
            if (err != nil) {
		    	output.Status = "FAILURE"
			    output.ErrorCode = "InternalError"
			    output.ErrorDesc = "Error while wrting to DB"
            } else {
		    	output.Status = "SUCCESS"
			    output.ErrorCode = "NONE"
			    output.ErrorDesc = "NONE"
            }
		}
	} else {
		output.Status = "FAILURE"
		output.ErrorCode = "InternalError"
		output.ErrorDesc = "Error while Fetching Data"
	}
	return &output, err
}

func (hndlr *SyncDbHandler) GetSyncRelnDetails(syncRelnInput *syncObjects.GetSyncRelnInput) (*syncDbObjects.DbGetSyncRelnOutput, error) {
	out, err := json.Marshal(*syncRelnInput)
	if err != nil {
		return nil, err
	}
	var dbInput syncDbObjects.DbGetSyncRelnInput
	err = json.Unmarshal(out, &dbInput)
	var output syncDbObjects.DbGetSyncRelnOutput

	syncRelnObj, err := hndlr.GetSyncReln(&dbInput.SyncId, &dbInput.TenantName)
	if syncRelnObj == nil {
        output.Status = "FAILURE"
        if err == nil {
           output.ErrorCode = "NONE"
           output.ErrorDesc = "NONE"
        } else {
           output.ErrorCode = "INTERNALERROR"
           output.ErrorDesc = "Error while reading from DB"
        }
	} else {
		//here do validations like src cpe path and set of dstCpePaths
        output.DbSyncReln = *syncRelnObj
        output.Status = "SUCCESS"
        output.ErrorCode = "DOESNOTEXISTS"
        output.ErrorDesc = "DOESNOTEXISTS"
	}
	return &output, err
}

func (hndlr *SyncDbHandler) createNewSyncReln(dbInput *syncDbObjects.DbCreateSyncRelnInput) error {
	if hndlr.isMongo {
        err := hndlr.updateSyncReln(&dbInput.DbSyncReln)
        return err
    }
	return nil
}


func (hndlr *SyncDbHandler) updateSyncReln(dbInput *syncDbObjects.DbSyncReln) error {
	if hndlr.isMongo {
		session := hndlr.session
		coll := session.DB(dbInput.TenantName).C("syncrelations")
		//ret,err := coll.UpsertId(dbInput.Id, dbInput)
		ret,err := coll.Upsert(bson.M{"syncid":dbInput.SyncId}, *dbInput)
        fmt.Println("err11",err, "ret",ret)
		return err
	}
    return nil
}

func (hndlr *SyncDbHandler) doesSyncRelnExists(syncRelnName, tenantName string) (bool, error) {
	syncRelnObj, err := hndlr.GetSyncReln(&syncRelnName, &tenantName)
	if syncRelnObj == nil {
		return false, err
	} else {
		//here do validations like src cpe path and set of dstCpePaths
		return true, nil
	}
}

func (hndlr *SyncDbHandler) GetSyncReln(relnName *string, tenantName *string) (*syncDbObjects.DbSyncReln, error) {
	session := hndlr.session
	coll := session.DB(*tenantName).C(SYNC_RELATION_TABLE)

	var syncRelnObj syncDbObjects.DbSyncReln
	count, err := coll.Find(bson.M{"tenantname":tenantName, "name":relnName}).Count()
    fmt.Println("count",count,"err",err)
    if err == nil && count == 0 {
       return nil, nil
    }
	err = coll.Find(bson.M{"tenantname":tenantName, "name":relnName}).One(&syncRelnObj)
    if err != nil {
       return nil, err
    } else {
	   return &syncRelnObj, nil
    }
    
    /*if count > 0 {
	    err := coll.Find(bson.M{"tenantname":tenantName, "name":relnName}).One(&syncRelnObj)
	    return &syncRelnObj, err
    } else {
        err = errors.New("Record Not Found")
    }*/
	return nil, err
}

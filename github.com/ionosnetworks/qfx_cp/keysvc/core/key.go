package core

import (
	"strings"

	"github.com/ionosnetworks/qfx_cmn/blog"
	idao "github.com/ionosnetworks/qfx_cp/keysvc/dao"
	o "github.com/ionosnetworks/qfx_cp/keysvc/objects"
	u "github.com/ionosnetworks/qfx_cp/keysvc/utils"
	cnts "github.com/ionosnetworks/qfx_cp/qfxConsts"
)

type (
	CoreKeyCreateIn     o.KeyCreateIn
	CoreKeyIn           o.KeyIn
	CoreClientKeyIn     o.ClientKeyIn
	CoreKeyOut          o.KeyOut
	CoreKeyEntryOut     o.KeyEntryOut
	CoreKeyEntryOutList o.KeyEntryOutList
)

var (
	ctx    = ""
	logger blog.Logger
)

const (
	LOCAL_KEY_CACHE_SIZE = 30
)

func SetLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
	idao.SetLogger(context, log)
}

func (key CoreKeyCreateIn) Process(token string) CoreKeyOut {
	var err idao.DAOErr

	// Check if client is asking for more preveleges than what it has.
	if ret := ValidateCredentials(token, key.Credential); ret != "OK" {
		logger.Err(ctx, "Invalid Credentials", blog.Fields{"Err": ret})
		return CoreKeyOut{ErrCode: ret}
	}

	key.AccessKey = u.SecureRandomAlphaNumeringString(cnts.ACCESS_KEY_LENGTH)
	key.AccessSecret = u.SecureRandomAlphaNumeringString(cnts.ACCESS_SECRET_LENGTH)
	if err = (idao.DaoKeyCreateIn(key)).DAOSave(); err != idao.DAO_OK {
		logger.Err(ctx, "Failed to save key", blog.Fields{"Err": err})
		return CoreKeyOut{ErrCode: "DAO Save Error"}
	}

	return CoreKeyOut{Name: key.Name, AccessKey: key.AccessKey,
		AccessSecret: key.AccessSecret, ErrCode: "OK"}
}

func (key CoreKeyIn) Delete() o.ObjectDeleteOut {
	var err idao.DAOErr

	if _, err = (idao.DaoKeyIn(key)).DAODelete(); err != idao.DAO_OK {
		logger.Err(ctx, "Failed to delete", blog.Fields{"Err": err})

		return o.ObjectDeleteOut{"DAO Delete Error"}
	}

	return o.ObjectDeleteOut{"OK"}
}

func (key CoreKeyIn) Load() CoreKeyEntryOut {

	if keyout, err := (idao.DaoKeyIn(key)).DAOLoad(); err != idao.DAO_OK {
		logger.Err(ctx, "Failed to load", blog.Fields{"Err": string(err)})

		return CoreKeyEntryOut{ErrCode: "Key does not exist"}
	} else {
		return CoreKeyEntryOut(keyout)
	}
}

func (key CoreClientKeyIn) Load() CoreKeyEntryOutList {

	if keyout, err := (idao.DaoClientKeyIn(key)).DAOLoad(); err != idao.DAO_OK {
		logger.Err(ctx, "Failed to load key ", blog.Fields{"Err": string(err)})
		return CoreKeyEntryOutList{ErrCode: "Key does not exist"}
	} else {
		return CoreKeyEntryOutList(keyout)
	}
}

func ValidateCredentials(callerToken string, reqCred o.KeyCredential) string {
	key := CoreKeyIn{AccessKey: callerToken}
	keyout := key.Load()

	keyCred := keyout.Credential
	for i := 0; i < len(reqCred.ApiList); i++ {
		allowed := false
		for j := 0; i < len(keyCred.ApiList); j++ {

			if strings.HasPrefix(reqCred.ApiList[i], keyCred.ApiList[j]) {
				logger.Debug(ctx, "API Found ", blog.Fields{"Has": keyCred.ApiList[j], "Req": reqCred.ApiList[i]})

				allowed = true
				break
			}
		}
		if !allowed {
			return "INVALID_ACCESS_REQUESTED"
		}
	}

	for i := 0; i < len(reqCred.FeatureList); i++ {
		allowed := false
		for j := 0; i < len(keyCred.FeatureList); j++ {

			if reqCred.FeatureList[i] == keyCred.FeatureList[j] {
				logger.Debug(ctx, "Feature Found", blog.Fields{"Has": keyCred.FeatureList[j],
					"Req": reqCred.FeatureList[i]})

				allowed = true
				break
			}
		}
		if !allowed {
			return "INVALID_ACCESS_REQUESTED"
		}
	}

	return "OK"
}

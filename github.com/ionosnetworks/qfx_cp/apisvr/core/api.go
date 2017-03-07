package core

import (
	"fmt"
	"strings"
	"sync"
	t "time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	idao "github.com/ionosnetworks/qfx_cp/apisvr/dao"
	o "github.com/ionosnetworks/qfx_cp/apisvr/objects"
	kcli "github.com/ionosnetworks/qfx_cp/keysvc/keycli"
)

type (
	CoreApiIn  o.ApiIn
	CoreApiOut o.ApiOut
	CoreApiKey o.ApiKey
)

type RegisteredApi struct {
	Uri     string
	RegTime t.Time
}

var (
	ApiMap   map[string]RegisteredApi = map[string]RegisteredApi{}
	ApiMutex                          = &sync.RWMutex{}

	ctx       = ""
	logger    blog.Logger
	keyClient *kcli.KeyCli
)

func Initialyze(key, secret, context string, log blog.Logger) {

	var err error

	setLogger(context, log)
	if keyClient, err = kcli.New(key, secret); err != nil {
		fmt.Println("Failed to intialyze key client")
	}
}

func setLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
	idao.SetLogger(context, log)
}

func PopulateHandlers() {

	reglist, err := idao.DAOGetAllRegistrations()
	if err != idao.DAO_OK {
		logger.Err(ctx, "Failed to populate handlers", blog.Fields{"err": err})
	}

	for _, regapi := range reglist {
		addHandler(regapi.AccessPoint, regapi.Uri)
	}
}

func SetLoggerHandle(l blog.Logger, c string) {
	logger = l
	ctx = c
}

func GetHandler(command string) string {

	if handle := GetHandlerFromCache(command); handle == "" {
		// Try populating from database again.
		PopulateHandlers()
	}

	return GetHandlerFromCache(command)
}

func GetHandlerFromCache(command string) string {

	ApiMutex.RLock()
	for p, u := range ApiMap {
		if strings.HasPrefix(command, p) {
			logger.Debug(ctx, "Access point found", blog.Fields{"Cmd": command, "AP": p})

			ApiMutex.RUnlock()
			return u.Uri
		}
	}
	ApiMutex.RUnlock()
	logger.Err(ctx, "Access point not found", blog.Fields{"Cmd": command})

	return ""
}

func addHandler(path, uri string) {

	logger.Debug(ctx, "Access point added", blog.Fields{"path": path, "AP": uri})
	ApiMutex.Lock()
	ApiMap[path] = RegisteredApi{Uri: uri, RegTime: t.Now()}
	ApiMutex.Unlock()
}

func deleteHandler(path string) {
	logger.Debug(ctx, "Access point deleted", blog.Fields{"path": path})

	ApiMutex.Lock()
	delete(ApiMap, path)
	ApiMutex.Unlock()
}

func (api CoreApiIn) Process() CoreApiOut {
	var err idao.DAOErr

	if err = (idao.DaoApiIn(api)).DAOSave(); err != idao.DAO_OK {

		logger.Err(ctx, "Failed to save", blog.Fields{"err": err})
		apiout := CoreApiOut{ErrCode: "DAO Save Error"}
		return apiout
	}

	addHandler(api.AccessPoint, api.Uri)
	return CoreApiOut{ErrCode: "OK"}
}

func (apikey CoreApiKey) Delete() o.ObjectDeleteOut {
	var err idao.DAOErr

	if _, err = (idao.DaoApiKey(apikey)).DAODelete(); err != idao.DAO_OK {

		logger.Err(ctx, "Failed to delete handler", blog.Fields{"err": err})
		return o.ObjectDeleteOut{"DAO Delete Error"}
	}

	deleteHandler(apikey.AccessPoint)
	return o.ObjectDeleteOut{"OK"}
}

func (apikey CoreApiKey) Load() CoreApiOut {

	if api, err := (idao.DaoApiKey(apikey)).DAOLoad(); err != idao.DAO_OK {
		logger.Err(ctx, "Failed to load handler", blog.Fields{"err": err})
		return CoreApiOut{}
	} else {
		return CoreApiOut(api)
	}
}

func ValidateRequest(key, secret, command string) (bool, []string) {
	return keyClient.ValidateApiRequest(key, secret, command)
}

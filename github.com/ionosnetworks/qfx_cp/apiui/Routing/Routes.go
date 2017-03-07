package Routing

import (
	ihttp "github.com/ionosnetworks/qfx_cmn/httplib/svr"
	ApiHandler "github.com/ionosnetworks/qfx_cp/apiui/Handler"
)

type Routes []ihttp.Routes

var ApiRoutes = ihttp.Routes{
	ihttp.Route{
		"login",
		"POST",
		"/login",
		ApiHandler.CheckLogin,
	},
	ihttp.Route{
		"cpe",
		"POST",
		"/cpe/{category}",
		ApiHandler.GetCpe,
	},
	ihttp.Route{
		"updateCpeLogLevel",
		"POST",
		"/updateCpeLogLevel",
		ApiHandler.UpdateCpeLogLevel,
	},
}

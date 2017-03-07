package Routing

import (
	"net/http"

	CPHandler "github.com/ionosnetworks/qfx_dp/cpewizard/Handler"
)

type Route struct {
	Name        string
	Method      string
	Path        string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var cpRoutes = Routes{
	Route{
		"cpeid",
		"POST",
		"/api/getCpeId",
		CPHandler.GetCpeId,
	},
	Route{
		"Login",
		"POST",
		"/api/Login",
		CPHandler.CheckLogin,
	},
	Route{
		"GetCpDetails",
		"POST",
		"/api/getCpDetails",
		CPHandler.GetCPDetails,
	},
	Route{
		"GetPortsInfo",
		"POST",
		"/api/getPortsInfo",
		CPHandler.GetPortInfoDetails,
	},
	Route{
		"UpdatePortsInfo",
		"POST",
		"/api/updatePortsInfo",
		CPHandler.UpdatePortsInfoWithInfo,
	},
	Route{
		"Logout",
		"POST",
		"/api/Logout",
		CPHandler.Logout,
	},
	Route{
		"Wan Status",
		"POST",
		"/api/wanStatus",
		CPHandler.GetWanStatusForInterfaces,
	},
	Route{
		"General Connectivity",
		"POST",
		"/api/checkConnectivity",
		CPHandler.GetConnectivity,
	},
	Route{
		"Controller Connectivity",
		"POST",
		"/api/controllerConnectivity",
		CPHandler.GetControllerConnectivity,
	},
	Route{
		"Logger",
		"POST",
		"api/getLogger",
		CPHandler.GetLoggerFile,
	},
	Route{
		"GetRootDirTree",
		"POST",
		"/api/getRootDirTree",
		CPHandler.GetRootDirTree,
	},
	Route{
		"getDirInfo",
		"POST",
		"/api/getDirInfo",
		CPHandler.GetDirInfo,
	},
	Route{
		"reloadDir",
		"POST",
		"/api/reloadDir",
		CPHandler.ReloadDir,
	},
}

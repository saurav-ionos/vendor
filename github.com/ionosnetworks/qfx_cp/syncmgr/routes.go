package main

import (
	ihttp "github.com/ionosnetworks/qfx_cmn/httplib/svr"
	hndlr "github.com/ionosnetworks/qfx_cp/syncmgr/handler"
)

var (
	routes = ihttp.Routes{

		ihttp.Route{"Key", "POST", "/sync/createSyncReln", hndlr.CreateSyncReln},
		ihttp.Route{"Key", "POST", "/sync/delSyncReln", hndlr.DeleteSyncReln},
		ihttp.Route{"Key", "GET", "/sync/getSyncReln", hndlr.GetSyncReln},
		ihttp.Route{"Key", "POST", "/sync/editSyncReln", hndlr.EditSyncReln},
	}
)

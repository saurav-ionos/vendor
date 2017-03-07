package Routing

import (
	"net/http"

	"github.com/gorilla/mux"
)

func AddNewRoute(router *mux.Router, newRoute Route) {
	var requestHandler http.HandlerFunc
	requestHandler = newRoute.HandlerFunc
	router.Methods(newRoute.Method).Path(newRoute.Path).Name(newRoute.Name).Handler(requestHandler)
}

func AddNewRoutes(router *mux.Router, routes []Route) {
	for _, route := range routes {
		AddNewRoute(router, route)
	}
}

func CreateNewRouter() *mux.Router {
	newRouter := mux.NewRouter().StrictSlash(true)
	AddNewRoutes(newRouter, cpRoutes)
	return newRouter
}

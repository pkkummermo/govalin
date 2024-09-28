package govalin

import (
	"net/http"
)

type ServeHTTPFunc func(w http.ResponseWriter, r *http.Request)

// HttpServe registers a ServeHttpFunc to a path which adheres to the http.Handler interface.
func (server *App) HTTPServe(path string, httpServeFunc ServeHTTPFunc) {
	fullPath := server.currentFragment + path
	handler := server.getOrCreatePathHandlerByPath(fullPath)

	handlerFunc := func(call *Call) {
		call.bypassLifecycle = true
		httpServeFunc(*call.Raw.W, call.Raw.Req)
	}

	handler.Get = handlerFunc
	handler.Post = handlerFunc
	handler.Put = handlerFunc
	handler.Patch = handlerFunc
	handler.Delete = handlerFunc
	handler.Options = handlerFunc
	handler.Head = handlerFunc

	for _, onRouteAdded := range server.config.server.events.onRouteAdded {
		onRouteAdded("ServeHTTP", fullPath, handlerFunc)
	}
}

package govalin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/pkkummermo/govalin/pkg/validation"
)

const (
	// govalin default port.
	defaultPort = 6060
	// maximum read timeout for requests.
	maxReadTimeout = 10
	// TODO: Find a nice max body read size.
	maxBodyReadSize int64 = 4096
	// Max time for shutdown.
	shutdownTimeoutInMS = 200
)

type HandlerFunc func(call *Call)
type BeforeFunc func(call *Call) bool
type AfterFunc func(call *Call)

type App struct {
	createdTime     time.Time
	started         bool
	port            uint16
	mux             *http.ServeMux
	server          http.Server
	currentFragment string
	pathHandlers    []pathHandler
}

// New creates a new Govalin App instance.
func New() (*App, error) {
	return &App{createdTime: time.Now(), port: defaultPort, currentFragment: "", mux: http.NewServeMux()}, nil
}

// Add a route to the given path
//
// Add a route which provides a scoped route function for which you can add
// methods or even more routes into. This allows for hierarchical building
// of routes and methods.
func (server *App) Route(path string, scopeFunc func()) *App {
	server.currentFragment += path

	scopeFunc()

	server.currentFragment = server.currentFragment[:len(server.currentFragment)-1]

	return server
}

func (server *App) addMethod(method string, fullPath string, methodHandler HandlerFunc) {
	var handler = server.getOrCreatePathHandlerByPath(fullPath)

	switch method {
	case http.MethodGet:
		if handler.Get != nil {
			log.Fatalf("GET already exists on path %s.", fullPath)
		}
		handler.Get = methodHandler
	case http.MethodPost:
		if handler.Post != nil {
			log.Fatalf("POST already exists on path %s.", fullPath)
		}
		handler.Post = methodHandler
	case http.MethodPut:
		if handler.Put != nil {
			log.Fatalf("PUT already exists on path %s.", fullPath)
		}
		handler.Put = methodHandler
	case http.MethodPatch:
		if handler.Patch != nil {
			log.Fatalf("PATCH already exists on path %s.", fullPath)
		}
		handler.Patch = methodHandler
	case http.MethodDelete:
		if handler.Delete != nil {
			log.Fatalf("DELETE already exists on path %s.", fullPath)
		}
		handler.Patch = methodHandler
	case http.MethodOptions:
		if handler.Options != nil {
			log.Fatalf("OPTIONS already exists on path %s.", fullPath)
		}
		handler.Options = methodHandler
	case http.MethodHead:
		if handler.Head != nil {
			log.Fatalf("HEAD already exists on path %s.", fullPath)
		}
		handler.Head = methodHandler
	default:
		log.Warnf("Unhandled method %s on path %s", method, fullPath)
		return
	}
}

// Add a before handler to given path
//
// Add a before handler that will run before any endpoint handler which matches
// the same request. If the before handler returns false, the request will be
// short circuited.
func (server *App) Before(path string, beforeFunc BeforeFunc) {
	fullPath := server.currentFragment + path
	var handler = server.getOrCreatePathHandlerByPath(fullPath)

	if handler.Before != nil {
		log.Fatalf("Before already exists on path %s.", fullPath)
	}

	handler.Before = beforeFunc
}

// Add an after handler to given path
//
// Add an after handler that will run after any endpoint handler which matches
// the same request.
func (server *App) After(path string, afterFunc AfterFunc) {
	fullPath := server.currentFragment + path
	var handler = server.getOrCreatePathHandlerByPath(fullPath)

	if handler.After != nil {
		log.Fatalf("Before already exists on path %s.", fullPath)
	}

	handler.After = afterFunc
}

// Add a GET handler
//
// Add a GET handler based on where you are in a hierarchy composed from
// other method handlers or route handlers.
func (server *App) Get(path string, handler HandlerFunc) *App {
	server.addMethod(http.MethodGet, server.currentFragment+path, handler)
	return server
}

// Add a POST handler
//
// Add a POST handler based on where you are in a hierarchy composed from
// other method handlers or route handlers.
func (server *App) Post(path string, handler HandlerFunc) *App {
	server.addMethod(http.MethodPost, server.currentFragment+path, handler)
	return server
}

// Add a PUT handler
//
// Add a PUT handler based on where you are in a hierarchy composed from
// other method handlers or route handlers.
func (server *App) Put(path string, handler HandlerFunc) *App {
	server.addMethod(http.MethodPut, server.currentFragment+path, handler)
	return server
}

// Add a PATCH handler
//
// Add a PATCH handler based on where you are in a hierarchy composed from
// other method handlers or route handlers.
func (server *App) Patch(path string, handler HandlerFunc) *App {
	server.addMethod(http.MethodPatch, server.currentFragment+path, handler)
	return server
}

// Add a OPTIONS handler
//
// Add a OPTIONS handler based on where you are in a hierarchy composed from
// other method handlers or route handlers.
func (server *App) Options(path string, handler HandlerFunc) *App {
	server.addMethod(http.MethodOptions, server.currentFragment+path, handler)
	return server
}

// Add a HEAD handler
//
// Add a HEAD handler based on where you are in a hierarchy composed from
// other method handlers or route handlers.
func (server *App) Head(path string, handler HandlerFunc) *App {
	server.addMethod(http.MethodHead, server.currentFragment+path, handler)
	return server
}

// Start the server
//
// Start the server based on given configuration.
func (server *App) Start(port ...uint16) error {
	if server.started {
		log.Warn("Server is already started")
		return fmt.Errorf("server has already started")
	}
	server.started = true

	if len(port) > 0 {
		server.port = port[0]
	}

	server.mux.HandleFunc("/", server.rootHandlerFunc)

	server.server = http.Server{
		ReadHeaderTimeout: time.Second * maxReadTimeout,
		Addr:              fmt.Sprintf(":%d", server.port),
		Handler:           server.mux,
	}

	log.Infof("Started govalin on port %d. Startup took %s 💪", server.port, time.Since(server.createdTime))
	if err := server.server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}

	return nil
}

// Shutdown the govalin server
//
// Start a graceful shutdown of the govalin instance.
func (server *App) Shutdown() error {
	if !server.started {
		log.Warn("Server was not started")
		return nil
	}

	log.Infof("Shutting down govalin. Server ran for %v 👋", time.Since(server.createdTime))

	ctx, closeFunc := context.WithTimeout(context.Background(), shutdownTimeoutInMS*time.Millisecond)
	defer closeFunc()

	return server.server.Shutdown(ctx)
}

func (server *App) getOrCreatePathHandlerByPath(path string) *pathHandler {
	if existingPathHandler, pathNotFoundErr := server.getPathHandlerByPath(path); pathNotFoundErr == nil {
		return existingPathHandler
	}
	newHandler, pathHandlerErr := newPathHandlerFromPathFragment(path)
	if pathHandlerErr != nil {
		log.Fatalf("Failed to create before handler for path '%s'. Err %v", path, pathHandlerErr)
	}

	server.pathHandlers = append(server.pathHandlers, newHandler)
	handler, err := server.getPathHandlerByPath(path)
	if err != nil {
		log.Fatalf("Failed to retrieve newly created handler for path '%s'. Err %v", path, err)
	}
	return handler
}

func (server *App) getPathHandlerByPath(path string) (*pathHandler, error) {
	for i := range server.pathHandlers {
		if server.pathHandlers[i].PathFragment == path {
			return &server.pathHandlers[i], nil
		}
	}

	return &pathHandler{}, fmt.Errorf(
		fmt.Sprintf("No pathHandler found for given path %s", path),
	)
}

func (server *App) rootHandlerFunc(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Server", "govalin")

	handled := false

	call := newCallFromRequest(
		w,
		req,
		map[string]string{},
	)

	// Look for before handlers
	for _, pathHandler := range server.pathHandlers {
		if pathHandler.Before != nil && pathHandler.PathMatcher.MatchesURL(req.URL.Path) {
			call.pathParams = pathHandler.PathMatcher.PathParams(req.URL.Path)
			handled = true
			if !pathHandler.Before(&call) {
				return
			}
		}
	}

	// Look for endpoint handler
	for _, pathHandler := range server.pathHandlers {
		if pathHandler.GetHandlerByMethod(req.Method) != nil && pathHandler.PathMatcher.MatchesURL(req.URL.Path) {
			var handler = pathHandler.GetHandlerByMethod(req.Method)
			call.pathParams = pathHandler.PathMatcher.PathParams(req.URL.Path)
			handler(&call)
			handled = true
			break
		}
	}

	// Look for After handlers
	for _, pathHandler := range server.pathHandlers {
		if pathHandler.After != nil && pathHandler.PathMatcher.MatchesURL(req.URL.Path) {
			call.pathParams = pathHandler.PathMatcher.PathParams(req.URL.Path)
			handled = true
			pathHandler.After(&call)
		}
	}

	if handled {
		return
	}

	server.notFoundHandler(&call)
}

func (server *App) notFoundHandler(call *Call) {
	call.Status(http.StatusNotFound)
	call.JSON(validation.NewError(
		validation.NewErrorResponse(
			http.StatusNotFound,
			validation.NewParameterErrorDetail(
				"path",
				fmt.Sprintf("The path '%s' doesn't exist", call.Raw().Req.URL),
			),
		),
	).ErrorResponse)
}

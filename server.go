package govalin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"log/slog"

	"github.com/pkkummermo/govalin/internal/validation"
)

type HandlerFunc func(call *Call)
type BeforeFunc func(call *Call) bool
type AfterFunc func(call *Call)
type StaticHandlerFunc func(call *Call, staticConfig *StaticConfig)

type App struct {
	config          *Config
	createdTime     time.Time
	started         bool
	port            uint16
	mux             *http.ServeMux
	server          http.Server
	currentFragment string
	pathHandlers    []pathHandler
}

// New creates a new Govalin App instance.
func New(config ...ConfigFunc) *App {
	initConfig := newConfig()

	if len(config) > 0 {
		config[0](initConfig)
	}

	for _, plugin := range initConfig.server.plugins {
		slog.Debug(fmt.Sprintf("Plugins: Running OnInit for '%s'", plugin.Name()))
		plugin.OnInit(initConfig)
	}

	return &App{
		config:          initConfig,
		createdTime:     time.Now(),
		currentFragment: "",
		mux:             http.NewServeMux(),
	}
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
			slog.Error(fmt.Sprintf("GET already exists on path %s.", fullPath))
			os.Exit(1)
		}
		handler.Get = methodHandler
	case http.MethodPost:
		if handler.Post != nil {
			slog.Error(fmt.Sprintf("POST already exists on path %s.", fullPath))
			os.Exit(1)
		}
		handler.Post = methodHandler
	case http.MethodPut:
		if handler.Put != nil {
			slog.Error(fmt.Sprintf("PUT already exists on path %s.", fullPath))
			os.Exit(1)
		}
		handler.Put = methodHandler
	case http.MethodPatch:
		if handler.Patch != nil {
			slog.Error(fmt.Sprintf("PATCH already exists on path %s.", fullPath))
			os.Exit(1)
		}
		handler.Patch = methodHandler
	case http.MethodDelete:
		if handler.Delete != nil {
			slog.Error(fmt.Sprintf("DELETE already exists on path %s.", fullPath))
			os.Exit(1)
		}
		handler.Delete = methodHandler
	case http.MethodOptions:
		if handler.Options != nil {
			slog.Error(fmt.Sprintf("OPTIONS already exists on path %s.", fullPath))
			os.Exit(1)
		}
		handler.Options = methodHandler
	case http.MethodHead:
		if handler.Head != nil {
			slog.Error(fmt.Sprintf("HEAD already exists on path %s.", fullPath))
			os.Exit(1)
		}
		handler.Head = methodHandler
	default:
		slog.Warn(fmt.Sprintf("Unhandled method %s on path %s", method, fullPath))
		return
	}

	for _, onRouteAdded := range server.config.server.events.onRouteAdded {
		onRouteAdded(method, fullPath, methodHandler)
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
		slog.Error(fmt.Sprintf("Before already exists on path %s.", fullPath))
		os.Exit(1)
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
		slog.Error(fmt.Sprintf("Before already exists on path %s.", fullPath))
		os.Exit(1)
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

// Add a DELETE handler
//
// Add a DELETE handler based on where you are in a hierarchy composed from
// other method handlers or route handlers.
func (server *App) Delete(path string, handler HandlerFunc) *App {
	server.addMethod(http.MethodDelete, server.currentFragment+path, handler)
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
		slog.Warn("Server is already started")
		return fmt.Errorf("server has already started")
	}
	server.started = true

	if len(port) > 0 {
		server.port = port[0]
	} else {
		server.port = server.config.server.port
	}

	// Reserve port and buffer incoming connections
	listener, listenerErr := net.Listen("tcp", fmt.Sprintf(":%d", server.port))
	if listenerErr != nil {
		return listenerErr
	}

	// Initialize all plugins
	for _, plugin := range server.config.server.plugins {
		slog.Debug(fmt.Sprintf("Plugins: Running Apply for '%s'", plugin.Name()))
		plugin.Apply(server)
	}

	server.mux.HandleFunc("/", server.rootHandlerFunc)

	server.server = http.Server{
		ReadHeaderTimeout: time.Second * time.Duration(server.config.server.maxReadTimeout),
		Handler:           server.mux,
	}

	if server.config.server.startupLogEnabled {
		slog.Info(fmt.Sprintf("Started govalin on port %d. Startup took %s 💪", server.port, time.Since(server.createdTime)))
		slog.Info(fmt.Sprintf("Server can be accessed at http://localhost:%d", server.port))
	}

	for _, onServerStartup := range server.config.server.events.onServerStartup {
		onServerStartup()
	}

	for _, onServerStartup := range server.config.server.events.onServerStartup {
		onServerStartup()
	}

	if serveErr := server.server.Serve(listener); serveErr != nil {
		if errors.Is(serveErr, http.ErrServerClosed) {
			return nil
		}
		return serveErr
	}

	return nil
}

// Shutdown the govalin server
//
// Start a graceful shutdown of the govalin instance.
func (server *App) Shutdown() error {
	if !server.started {
		slog.Warn("Server was not started")
		return nil
	}

	for _, onServerShutdown := range server.config.server.events.onServerShutdown {
		onServerShutdown()
	}

	if server.config.server.startupLogEnabled {
		slog.Info(fmt.Sprintf("Shutting down govalin. Server ran for %v 👋", time.Since(server.createdTime)))
	}

	ctx, closeFunc := context.WithTimeout(
		context.Background(),
		time.Duration(server.config.server.shutdownTimeoutInMS)*time.Millisecond,
	)
	defer closeFunc()

	return server.server.Shutdown(ctx)
}

func (server *App) getOrCreatePathHandlerByPath(path string) *pathHandler {
	if existingPathHandler, pathNotFoundErr := server.getPathHandlerByPath(path); pathNotFoundErr == nil {
		return existingPathHandler
	}
	newHandler, pathHandlerErr := newPathHandlerFromPathFragment(path)
	if pathHandlerErr != nil {
		slog.Error(fmt.Sprintf("Failed to create handler for path '%s'. Err: %v", path, pathHandlerErr))
		os.Exit(1)
	}

	server.pathHandlers = append(server.pathHandlers, newHandler)
	handler, err := server.getPathHandlerByPath(path)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to retrieve newly created handler for path '%s'. Err %v", path, err))
		os.Exit(1)
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

func (server *App) matchBeforeHandlers(call *Call) bool {
	for _, pathHandler := range server.pathHandlers {
		if pathHandler.Before != nil && pathHandler.PathMatcher.MatchesURL(call.URL().Path) {
			call.pathParams = pathHandler.PathMatcher.PathParams(call.URL().Path)

			// Return false means short circuit, return false
			if !pathHandler.Before(call) {
				return false
			}
		}
	}

	return true
}

func (server *App) matchHandlers(call *Call) {
	for _, pathHandler := range server.pathHandlers {
		if pathHandler.GetHandlerByMethod(call.Method()) != nil && pathHandler.PathMatcher.MatchesURL(call.URL().Path) {
			var handler = pathHandler.GetHandlerByMethod(call.Method())
			call.pathParams = pathHandler.PathMatcher.PathParams(call.URL().Path)
			handler(call)
			break
		}
	}
}

func (server *App) matchAfterHandlers(call *Call) {
	for _, pathHandler := range server.pathHandlers {
		if pathHandler.After != nil && pathHandler.PathMatcher.MatchesURL(call.URL().Path) {
			call.pathParams = pathHandler.PathMatcher.PathParams(call.URL().Path)
			pathHandler.After(call)
		}
	}
}

func (server *App) rootHandlerFunc(w http.ResponseWriter, req *http.Request) {
	incomingRequestTime := time.Now()
	w.Header().Add("Server", "govalin")

	call := newCallFromRequest(
		w,
		req,
		server.config,
		map[string]string{},
	)

	// Look for before handlers
	if !server.matchBeforeHandlers(&call) {
		// Before handler returned false, meaning short circuit, meaning we need to log access log here
		server.logAccessLog(&call, float64(time.Since(incomingRequestTime))/float64(time.Millisecond))
		return
	}

	// Look for endpoint handler
	server.matchHandlers(&call)

	// Look for After handlers
	server.matchAfterHandlers(&call)

	// No status set, meaning no handlers have handled the request properly,
	// ie 404 / not found
	if call.Status() == 0 {
		server.notFoundHandler(&call)
	}

	if server.config.server.accessLogEnabled {
		server.logAccessLog(&call, float64(time.Since(incomingRequestTime))/float64(time.Millisecond))
	}
}

func (server *App) logAccessLog(call *Call, durationInMS float64) {
	slog.Info(
		"incoming request",
		slog.String("id", call.ID()),
		slog.String("method", call.Method()),
		slog.Float64("duration_in_ms", durationInMS),
		slog.String("path", call.URL().Path),
		slog.Int("status", call.status),
		slog.String(
			"user_agent",
			call.UserAgent(),
		),
	)
}

func (server *App) notFoundHandler(call *Call) {
	call.Status(http.StatusNotFound)
	call.JSON(validation.NewError(
		validation.NewErrorResponse(
			http.StatusNotFound,
			validation.NewParameterErrorDetail(
				"path",
				fmt.Sprintf("The path '%s' doesn't exist", call.URL()),
			),
		),
	).ErrorResponse)
}

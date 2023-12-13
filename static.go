package govalin

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// StaticConfig contains configuration for a static handler.
type StaticConfig struct {
	hostPath   string
	spaMode    bool
	staticPath string
	fsContent  fs.FS
}

func newStaticConfig() *StaticConfig {
	return &StaticConfig{
		hostPath:   "/",
		spaMode:    false,
		staticPath: "static",
		fsContent:  nil,
	}
}

func (config *StaticConfig) serveIndexFS(call *Call) {
	// file does not exist, serve index.html
	file, openIndexErr := config.fsContent.Open("index.html")
	if openIndexErr != nil {
		slog.Error(`Failed to open index.html in bundled embedded files.
This might be due to a misconfigured embedded bundle or simply 
that the index.html files doesn't exist.`)
		call.Error(openIndexErr)
		return
	}

	index, readIndexErr := io.ReadAll(file)
	if readIndexErr != nil {
		slog.Error(`Failed to read index.html in bundled embedded files.
This might be due to a misconfigured embedded bundle or simply 
that the index.html files doesn't exist.`)
		call.Error(openIndexErr)
		return
	}

	call.Status(http.StatusOK)
	call.HTML(string(index))
}

func (config *StaticConfig) serveIndexStatic(call *Call) {
	indexPath := filepath.Join(config.staticPath, "index.html")
	if _, indexFileDoesNotExist := os.Stat(
		indexPath,
	); os.IsNotExist(indexFileDoesNotExist) {
		slog.Error(fmt.Sprintf(`Failed to read the index.html file in the given static file folder. 
Are you sure it exists on the given path: '%s'`, indexPath))
		call.Error(indexFileDoesNotExist)
		return
	}

	// file does not exist, serve index.html
	call.Status(http.StatusOK)
	http.ServeFile(*call.Raw.W, call.Raw.Req, filepath.Join(config.staticPath, "index.html"))
}

func (config *StaticConfig) handle(call *Call) {
	isFS := config.fsContent != nil
	isStatic := config.fsContent == nil

	path := call.URL().Path

	// remove host path
	path = strings.TrimPrefix(path, config.hostPath)

	if isStatic {
		// prepend the path with the path to the static directory
		path = filepath.Join(config.staticPath, path)
	}

	// check whether a file exists at the given path
	var statErr error

	if isFS {
		_, statErr = fs.Stat(config.fsContent, strings.TrimPrefix(path, "/"))
	} else {
		_, statErr = os.Stat(path)
	}

	var pathErr *fs.PathError

	isNotFoundError := errors.Is(statErr, fs.ErrNotExist) || errors.As(statErr, &pathErr)

	// Serve index if:
	// 1. If path is empty (slash root)
	// 2. if SPA mode is enabled, and if the file doesn't exist
	if path == "" || (config.spaMode && isNotFoundError) {
		if isFS {
			config.serveIndexFS(call)
		} else {
			config.serveIndexStatic(call)
		}
		return
	}

	switch {
	case isNotFoundError:
		call.Status(http.StatusNotFound)
	case statErr != nil:
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		call.Status(http.StatusInternalServerError)
		call.Error(statErr)
		return
	default:
		call.Status(http.StatusOK)
	}

	// otherwise, use http.FileServer to serve the file system
	var hostedFileSystem http.FileSystem
	if isFS {
		hostedFileSystem = http.FS(config.fsContent)
	} else {
		hostedFileSystem = http.Dir(config.staticPath)
	}

	http.StripPrefix(
		config.hostPath,
		http.FileServer(hostedFileSystem),
	).ServeHTTP(*call.Raw.W, call.Raw.Req)
}

// HostPath sets the host path for the static handler. This is trimmed from the
// URL before serving the static files.
func (config *StaticConfig) HostPath(hostPath string) *StaticConfig {
	config.hostPath = hostPath

	return config
}

// WithStaticPath sets the path to a directory containing static files.
//
// The directory will be served at the given path, relative to where the server
// is started.
func (config *StaticConfig) WithStaticPath(staticPath string) *StaticConfig {
	config.staticPath = staticPath

	return config
}

// WithFS sets the bundled FS to serve static files from.
func (config *StaticConfig) WithFS(fsContent fs.FS) *StaticConfig {
	config.fsContent = fsContent

	return config
}

// EnableSPAMode enables SPA mode for the static handler.
//
// SPA mode will serve the index.html file for all requests that doesn't match
// a static file.
func (config *StaticConfig) EnableSPAMode(spaMode bool) *StaticConfig {
	config.spaMode = spaMode

	return config
}

// Add a Static endpoint
//
// Add a static endpoint which will serve static files from the given path or bundled FS.
func (server *App) Static(path string, staticHandlerFunc StaticHandlerFunc) *App {
	// TODO: this doesn't feel right to override the path like this
	normalizedPath := strings.TrimRight(path, "/*")
	wildcardPath := normalizedPath + "/*"

	staticGetHandler := func(call *Call) {
		internalConfig := newStaticConfig()
		internalConfig.HostPath(normalizedPath)

		staticHandlerFunc(call, internalConfig)

		internalConfig.handle(call)
	}

	// TODO: this should be handled by a single handler, not two
	server.addMethod(http.MethodGet, server.currentFragment+normalizedPath+"/", staticGetHandler)
	server.addMethod(http.MethodGet, server.currentFragment+wildcardPath, staticGetHandler)

	return server
}
